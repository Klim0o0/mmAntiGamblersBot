package botLogic

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
	"log"
	"mmAntiGamblersBot/sqlCache"
	"time"
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

func ListenUpdates(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, conn *pgx.Conn) {
	cache := sqlCache.CreateCache(conn, context.Background())

	for update := range updates {
		if update.Message != nil && update.Message.Dice != nil {
			go calculateDice(update, cache, bot)
		}
	}
}

func calculateDice(update tgbotapi.Update, cache *sqlCache.GamblingMessageCache, bot *tgbotapi.BotAPI) {
	message := update.Message

	messageInfo := getGamblingMessageInfo(message)

	msg, ok := cache.Get(messageInfo)

	if ok && !msg.MessageDate.After(messageInfo.MessageDate) {
		log.Println("Уже есть")
		_, _ = bot.Send(tgbotapi.NewDeleteMessage(messageInfo.ChatId, update.Message.MessageID))
		_, _ = muteUser(bot, messageInfo.ChatId, messageInfo.UserId)
		return
	}

	cache.Set(messageInfo)
}

func muteUser(bot *tgbotapi.BotAPI, chatId int64, userId int64) (tgbotapi.Message, error) {

	return bot.Send(
		tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: chatId, UserID: userId},
			//UntilDate:        time.Now().Add(time.Minute).Unix(),
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
