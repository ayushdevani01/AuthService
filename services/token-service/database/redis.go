package database

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(redisURL string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v", err)
	}

	log.Println("Token Service: Connected to Redis")
	return client
}
