package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/go-redis/redis/v8"
)

// Locker is implemented by both RedisLock (production) and MutexLock (tests).
type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (token string, ok bool, err error)
	Release(ctx context.Context, key, token string) error
}

type RedisLock struct {
	client *redis.Client
}

func New(client *redis.Client) *RedisLock {
	return &RedisLock{client: client}
}

func (l *RedisLock) Acquire(ctx context.Context, key string, ttl time.Duration) (token string, ok bool, err error) {
	token = randToken()
	ok, err = l.client.SetNX(ctx, key, token, ttl).Result()
	return
}

func (l *RedisLock) Release(ctx context.Context, key, token string) error {
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)
	return script.Run(ctx, l.client, []string{key}, token).Err()
}

func randToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
