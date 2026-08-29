package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

type RedisEmptyValue struct{}

func (m *RedisEmptyValue) Error() string {
	return "get empty value from redis"
}

func createRedisClient() *redis.Client {
	addr := getRequiredEnv("REDIS_ADDR")

	db := 0
	if raw := os.Getenv("REDIS_DB"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			log.Fatalf("invalid REDIS_DB %q: %v", raw, err)
		}
		db = int(parsed)
	}

	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})
}

func setValue(ctx context.Context, redisClient *redis.Client, key string, value interface{}) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	log.Printf("%s", value)
	return redisClient.Set(ctx, key, jsonData, 0).Err()
}

func getValue(ctx context.Context, redisClient *redis.Client, key string, result interface{}) (bool, error) {
	val, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, &RedisEmptyValue{}
	} else if err != nil {
		return false, errors.New("internal server error")
	}

	err = json.Unmarshal([]byte(val), result)
	if err != nil {
		return false, err
	}
	log.Printf("%s", result)

	return true, nil
}
