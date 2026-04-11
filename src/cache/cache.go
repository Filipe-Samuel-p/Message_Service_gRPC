package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	rdb *redis.Conn
}

func (rdb *RedisClient) SetUserOnline(userID uuid.UUID) error {
	ctx := context.Background()
	return rdb.rdb.Set(ctx, "online:"+userID.String(), "online", 60*time.Second).Err()
}
