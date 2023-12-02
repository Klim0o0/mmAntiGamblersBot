package main

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
	"log"
	"mmAntiGamblersBot/botLogic"
	"os"
	"time"
)

func main() {
	argsWithoutProg := os.Args[1:]
	connString := argsWithoutProg[0]
	conn, err := pgx.Connect(context.Background(), connString)
	defer conn.Close(context.Background())
	if err != nil {
		log.Panic(err)
	}

	bot, err := tgbotapi.NewBotAPI(argsWithoutProg[1])
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
