package botLogic

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
	"log"
	"time"
)

type gamblingMessageInfo struct {
	userId      int64
	chatId      int64
	messageDate time.Time
	emoji       string
	emojiValue  int
}

func getGamblingMessageInfo(message *tgbotapi.Message) gamblingMessageInfo {
	return gamblingMessageInfo{
		userId:      message.From.ID,
		chatId:      message.Chat.ID,
		messageDate: time.Unix(int64(message.Date), 0).UTC(),
		emoji:       message.Dice.Emoji,
		emojiValue:  message.Dice.Value,
	}
}

func ListenUpdates(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, conn *pgx.Conn) {

	for update := range updates {
		if update.Message != nil && update.Message.Dice != nil {
			calculateDice(update, conn, bot)
		}
	}
}

func calculateDice(update tgbotapi.Update, conn *pgx.Conn, bot *tgbotapi.BotAPI) {
	message := update.Message

	messageInfo := getGamblingMessageInfo(message)
	isExist, err := isCanSend(messageInfo.userId, messageInfo.chatId, messageInfo.emoji, conn)
	if err != nil {
		log.Printf(err.Error())
	}

	if !isExist {
		log.Println("Уже есть")
		_, _ = bot.Send(tgbotapi.NewDeleteMessage(messageInfo.chatId, update.Message.MessageID))
		_, _ = muteUser(bot, messageInfo.chatId, messageInfo.userId)
		return
	}

	err = insertInfo(messageInfo.userId, messageInfo.chatId, messageInfo.messageDate, messageInfo.emoji, messageInfo.emojiValue, conn)
	if err != nil {
		log.Printf(err.Error())
	} else {
		log.Printf("Sucsess")
	}
}

func muteUser(bot *tgbotapi.BotAPI, chatId int64, userId int64) (tgbotapi.Message, error) {
	return bot.Send(
		tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: chatId, UserID: userId},
			UntilDate:        time.Now().Add(time.Minute).Unix(),
			Permissions:      &tgbotapi.ChatPermissions{CanSendMessages: false}})
}

func insertInfo(userId, chatId int64, messageDate time.Time, emojiText string, emojiValue int, conn *pgx.Conn) error {

	query, err := conn.Query(context.Background(),
		"insertInfo into tg_messages values($1, $2, $3, $4, $5)",
		userId, chatId, messageDate, emojiText, emojiValue)
	query.Close()
	return err
}
func isCanSend(userId, chatId int64, emojiText string, conn *pgx.Conn) (bool, error) {
	query, err := conn.Query(context.Background(),
		"select * from tg_messages where userId=$1 and chatId=$2 and emoji = $3 and masageDate > $4",
		userId, chatId, emojiText, time.Now().AddDate(0, 0, -1).UTC())
	defer query.Close()
	for query.Next() {
		return false, err
	}
	return true, err
}
