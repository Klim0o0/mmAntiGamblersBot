package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
	"log"
	"mmAntiGamblersBot/botLogic"
	"mmAntiGamblersBot/config"
	"time"
)

func main() {

	configuration := config.LoadConfig()
	connString := fmt.Sprintf("host=%s port=5432 dbname=%s user=%s password=%s sslmode=%s connect_timeout=10",
		configuration.DBAddress, configuration.DBName, configuration.DBUsername, configuration.DBPassword, configuration.SSLMode)

	conn, err := pgx.Connect(context.Background(), connString)
	defer conn.Close(context.Background())
	if err != nil {
		log.Panic(err)
	}

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
	botLogic.ListenUpdates(updates, bot, conn, ctx)
}
