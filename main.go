package main

import (
	"context"
	"log"
	"mmAntiGamblersBot/botLogic"
	"mmAntiGamblersBot/config"
	"time"

	tgbotapi "github.com/sotarevid/telegram-bot-api"
)

func main() {

	configuration := config.LoadConfig()

	bot, err := tgbotapi.NewBotAPI(configuration.BotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(2 * time.Second)
	}()
	botLogic.ListenUpdates(updates, bot, ctx)
}
