package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SSLMode  string
	BotToken string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	config := Config{
		SSLMode:  getEnv("SSL_MODE", "disable"),
		BotToken: getEnv("BOT_TOKEN", ""),
	}

	return config
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	if defaultVal == "" {
		log.Fatalf("Environment variable %s not set", key)
	}

	return defaultVal
}
