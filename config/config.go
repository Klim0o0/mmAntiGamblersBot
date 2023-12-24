package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUsername string
	DBPassword string
	DBAddress  string
	DBName     string
	SSLMode    string
	BotToken   string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	config := Config{
		DBUsername: getEnv("DB_USERNAME", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBAddress:  getEnv("DB_ADDRESS", ""),
		DBName:     getEnv("DB_NAME", ""),
		SSLMode:    getEnv("SSL_MODE", "disable"),
		BotToken:   getEnv("BOT_TOKEN", ""),
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
