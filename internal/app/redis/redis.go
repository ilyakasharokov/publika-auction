package redis_service

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

var ctx = context.Background()

type RedisClient struct {
	*redis.Client
}

func (rc *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := rc.Client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, err
}

func (rc *RedisClient) Set(ctx context.Context, key string, data string, exp time.Duration) error {
	err := rc.Client.Set(ctx, key, data, exp).Err()
	return err
}

func NewRedisClient(addr string, pwd string, db int) *RedisClient {
	return &RedisClient{
		redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: pwd,
			DB:       db,
		}),
	}
}
