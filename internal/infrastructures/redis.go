package infrastructures

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func NewRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0, // use default DB
	})

	// Test the connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		logrus.Fatalf("failed to connect redis: %v", err)
	}

	return client
}
