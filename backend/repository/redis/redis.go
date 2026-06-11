package redis

import (
	"context"
	"fmt"
	"qris-latency-optimizer/config"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var RedisClient *goredis.Client
var RedisAvailable bool

// TTL constants
const (
	TTLTransaction = 10 * time.Minute
	TTLMerchant    = 30 * time.Minute
	TTLInquiry     = 2 * time.Minute
)

// ConnectRedis - koneksi ke Redis
func ConnectRedis() {
	RedisClient = goredis.NewClient(&goredis.Options{
		Addr: config.App.RedisAddr(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := RedisClient.Ping(ctx).Err()
	if err != nil {
		RedisAvailable = false
		fmt.Printf("Redis connection failed: %v (running without cache)\n", err)
		return
	}

	RedisAvailable = true
	fmt.Println("Redis connected successfully")
}

// Get - ambil data dari Redis
// Return:
// value, nil => cache HIT
// "", error  => cache MISS / redis error
func Get(key string) (string, error) {
	if !RedisAvailable {
		return "", fmt.Errorf("redis unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return RedisClient.Get(ctx, key).Result()
}

// Set - simpan data ke Redis
func Set(key string, value string, expiration time.Duration) error {
	if !RedisAvailable {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return RedisClient.Set(ctx, key, value, expiration).Err()
}

// Delete - hapus cache
func Delete(key string) error {
	if !RedisAvailable {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return RedisClient.Del(ctx, key).Err()
}

// Exists - cek apakah key ada
func Exists(key string) (bool, error) {
	if !RedisAvailable {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	count, err := RedisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
