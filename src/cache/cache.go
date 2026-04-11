package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(client *redis.Client) *RedisClient {
	return &RedisClient{
		client: client,
	}
}

func (r *RedisClient) SetUserOnline(ctx context.Context, userID uuid.UUID) error {
	return r.client.Set(ctx, "online:"+userID.String(), "true", 60*time.Second).Err()
}

func (r *RedisClient) KeepAlive(ctx context.Context, userID uuid.UUID) error {
	return r.client.Expire(ctx, "online:"+userID.String(), 60*time.Second).Err()
}

func (r *RedisClient) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	val, err := r.client.Exists(ctx, "online:"+userID.String()).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}
