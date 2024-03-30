package botLogic

import (
	"context"
	"log"
	"mmAntiGamblersBot/sqlCache"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
)

func getGamblingMessageInfo(message *tgbotapi.Message) sqlCache.GamblingMessageInfo {
	messageDate := time.Unix(int64(message.Date), 0)
	messageDateOnly := time.Date(messageDate.Year(), messageDate.Month(), messageDate.Day(), 0, 0, 0, 0, time.Local)

	return sqlCache.GamblingMessageInfo{
		UserChatIndicator: sqlCache.UserChatIndicator{
			UserId: message.From.ID,
			ChatId: message.Chat.ID,
			Emoji:  message.Dice.Emoji,
		},
		MessageDate: messageDateOnly,
		EmojiValue:  message.Dice.Value,
	}
}

func ListenUpdates(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, conn *pgx.Conn, ctx context.Context) {
	cache := sqlCache.CreateCache(conn, ctx)

	for update := range updates {
		// disable on friday
		today := time.Now().Weekday()
		if today == time.Saturday {
			continue
		}
		// Klim0o0 is allowed to gamble
		if update.Message != nil && update.Message.From != nil && update.Message.From.UserName == "Klim0o0" {
			continue
		}
		// otherwise let's check for gambling
		if update.Message != nil && update.Message.Dice != nil {
			go calculateDice(update, cache, bot)
		}

	}
}

func calculateDice(update tgbotapi.Update, cache *sqlCache.GamblingMessageCache, bot *tgbotapi.BotAPI) {
	message := update.Message

	messageInfo := getGamblingMessageInfo(message)

	_, ok := cache.Get(messageInfo)

	if !ok {
		cache.Set(messageInfo)
		return
	}

	log.Printf("Gambling policy violation detected by user: %s\n", message.From.UserName)
	_, _ = bot.Send(tgbotapi.NewDeleteMessage(messageInfo.ChatId, update.Message.MessageID))
	_, _ = muteUser(bot, messageInfo.ChatId, messageInfo.UserId)
	return

}

func muteUser(bot *tgbotapi.BotAPI, chatId int64, userId int64) (tgbotapi.Message, error) {
	muteTime := time.Minute
	today := time.Now()
	if today.Month() == time.April && today.Day() == 1 {
		muteTime = time.Hour * 2
	}
	return bot.Send(
		tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: chatId, UserID: userId},
			UntilDate:        time.Now().Add(muteTime).Unix(),
			Permissions: &tgbotapi.ChatPermissions{
				CanSendMessages: false,
			}})
}

func UnMuteUser(bot *tgbotapi.BotAPI, chatId int64, userId int64) (tgbotapi.Message, error) {

	return bot.Send(
		tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: chatId, UserID: userId},
			//UntilDate:        time.Now().Add(time.Minute).Unix(),
			Permissions: &tgbotapi.ChatPermissions{
				CanSendMessages:      true,
				CanSendMediaMessages: true,
				CanSendPolls:         true,
				CanSendOtherMessages: true,
				CanChangeInfo:        true,
				CanInviteUsers:       true,
			}})
}
