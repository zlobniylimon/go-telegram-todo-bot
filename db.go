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
	db, _ := strconv.ParseInt(os.Getenv("REDIS_DB"), 10, 32)
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       int(db),
	})

	return redisClient
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
