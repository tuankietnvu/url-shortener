package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
}

// LoadConfig loads environment variables from `.env` (if present) and returns
// the application configuration.
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	return &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}