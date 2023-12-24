package main

import (
	"context"
	"fmt"
	"log"
	"mmAntiGamblersBot/botLogic"
	"mmAntiGamblersBot/config"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
)

func main() {

	config := config.LoadConfig()
	connString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", config.DBUsername, config.DBPassword, config.DBAddress, config.DBName, config.SSLMode)

	m, err := migrate.New("file://db/migrations", connString)
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No new migrations to apply")
		} else {
			log.Fatal(err)
		}
	}

	conn, err := pgx.Connect(context.Background(), connString)
	defer conn.Close(context.Background())
	if err != nil {
		log.Panic(err)
	}

	bot, err := tgbotapi.NewBotAPI(config.BotToken)
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
