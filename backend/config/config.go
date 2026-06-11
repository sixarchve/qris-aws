package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var App *Config

type Config struct {
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     string

	RedisHost string
	RedisPort string

	ReceiptDir string
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func getEnvDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func Load() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	App = &Config{
		DBHost:     getEnv("DB_HOST"),
		DBUser:     getEnv("DB_USER"),
		DBPassword: getEnv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME"),
		DBPort:     getEnv("DB_PORT"),

		RedisHost: getEnv("REDIS_HOST"),
		RedisPort: getEnv("REDIS_PORT"),

		ReceiptDir: getEnvDefault("RECEIPT_DIR", "../receipts"),
	}

	fmt.Println("Config loaded!")
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}
