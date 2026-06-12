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

	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
	S3BucketName       string
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

		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY"),
		AWSRegion:          getEnvDefault("AWS_REGION", "ap-southeast-1"),
		S3BucketName:       getEnv("S3_BUCKET_NAME"),
	}


	fmt.Println("Config loaded!")
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func (c *Config) IsS3Configured() bool {
	return c.AWSAccessKeyID != "" && c.AWSSecretAccessKey != "" && c.S3BucketName != ""
}

