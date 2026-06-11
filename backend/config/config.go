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

	RabbitMQUser     string
	RabbitMQPassword string
	RabbitMQHost     string
	RabbitMQPort     string
	RabbitMQURLRaw   string
}

func getEnv(key string) string {
	return os.Getenv(key)
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

		RabbitMQUser:     getEnv("RABBITMQ_USER"),
		RabbitMQPassword: getEnv("RABBITMQ_PASSWORD"),
		RabbitMQHost:     getEnv("RABBITMQ_HOST"),
		RabbitMQPort:     getEnv("RABBITMQ_PORT"),
		RabbitMQURLRaw:   getEnv("RABBITMQ_URL"),
	}

	fmt.Println("Config loaded!")
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func (c *Config) RabbitMQURL() string {
	if c.RabbitMQURLRaw != "" {
		return c.RabbitMQURLRaw
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%s/", c.RabbitMQUser, c.RabbitMQPassword, c.RabbitMQHost, c.RabbitMQPort)
}
