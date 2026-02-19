package cache

import (
	"context"
	"time"

	"duskforge-api/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func New(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	logger.Logger.Info("Redis client connected", zap.String("addr", opts.Addr))
	return client, nil
}
