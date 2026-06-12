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

	DBSslMode     string
	RedisPassword string
	RedisUseTLS   bool

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

	dbHost := getEnv("DB_HOST")
	redisHost := getEnv("REDIS_HOST")

	// Detect if running inside docker container
	inDocker := os.Getenv("RUNNING_IN_DOCKER") == "true"
	if !inDocker {
		if _, err := os.Stat("/.dockerenv"); err == nil {
			inDocker = true
		}
	}

	if inDocker {
		if dbHost == "localhost" || dbHost == "127.0.0.1" || dbHost == "" {
			dbHost = "db"
		}
		if redisHost == "localhost" || redisHost == "127.0.0.1" || redisHost == "" {
			redisHost = "redis"
		}
	}

	App = &Config{
		DBHost:        dbHost,
		DBUser:        getEnv("DB_USER"),
		DBPassword:    getEnv("DB_PASSWORD"),
		DBName:        getEnv("DB_NAME"),
		DBPort:        getEnv("DB_PORT"),
		DBSslMode:     getEnvDefault("DB_SSLMODE", "disable"),
		RedisHost:     redisHost,
		RedisPort:     getEnv("REDIS_PORT"),
		RedisPassword: getEnv("REDIS_PASSWORD"),
		RedisUseTLS:   getEnv("REDIS_USE_TLS") == "true",

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

