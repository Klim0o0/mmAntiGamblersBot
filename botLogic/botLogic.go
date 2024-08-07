package botLogic

import (
	"context"
	"log"
	"slices"
	"time"

	tgbotapi "github.com/sotarevid/telegram-bot-api"
)

func ListenUpdates(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, ctx context.Context) {
	for update := range updates {
		if update.Message == nil {
			continue
		}

		//send a sticker on user join
		if update.Message.NewChatMembers != nil {
			var sticker = tgbotapi.NewSticker(update.Message.Chat.ID, tgbotapi.FileID(joinStickerId))
			bot.Send(sticker)
		}

		// Klim0o0 is allowed to shitpost
		if update.Message.From != nil && update.Message.From.UserName == "Klim0o0" {
			continue
		}
		// otherwise let's check for violations
		if update.Message.Dice != nil {
			go checkGambling(update.Message, bot)
		} else {
			go checkBots(update.Message, bot)
		}

	}
}

func checkGambling(message *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if message.MessageThreadID == gamblingTopicId {
		return
	}

	if message.Dice != nil {
		log.Printf("Gambling policy violation detected by user: %s\n", message.From.UserName)

		_, _ = muteUser(bot, message.Chat.ID, message.From.ID)
		_, _ = bot.Send(tgbotapi.NewDeleteMessage(message.Chat.ID, message.MessageID))
	}
}

func checkBots(message *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if message.MessageThreadID == botsTopicId {
		return
	}

	if message.From.IsBot || message.IsCommand() || message.ViaBot != nil {
		if message.ViaBot != nil && slices.Contains(whitelist, message.ViaBot.UserName) {
			return
		}

		if !message.From.IsBot {
			log.Printf("Bot policy violation detected by user: %s\n", message.From.UserName)

			_, _ = muteUser(bot, message.Chat.ID, message.From.ID)
		}

		_, _ = bot.Send(tgbotapi.NewDeleteMessage(message.Chat.ID, message.MessageID))
	}
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
