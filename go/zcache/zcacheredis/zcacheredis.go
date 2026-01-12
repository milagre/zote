package zcacheredis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/milagre/zote/go/zcache"
)

type RedisClient interface {
	SetEx(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, key ...string) *redis.IntCmd
}

type redisCache struct {
	client RedisClient
}

func NewRedisCache(c RedisClient) zcache.Cache {
	return redisCache{
		client: c,
	}
}

func (c redisCache) Set(ctx context.Context, namespace string, key string, expiration time.Duration, value []byte) error {
	err := c.client.SetEx(
		ctx,
		c.attr(namespace, key),
		value,
		expiration,
	).Err()
	if err != nil {
		return fmt.Errorf("setting redis cache entry: %w", err)
	}

	return nil
}

func (c redisCache) Get(ctx context.Context, namespace string, key string) (<-chan []byte, error) {
	res := make(chan []byte, 1)
	defer close(res)

	val, err := c.client.Get(
		ctx,
		c.attr(namespace, key),
	).Result()
	if err != nil {
		if err != redis.Nil {
			return res, fmt.Errorf("getting redis cache entry: %w", err)
		}
	} else {
		res <- []byte(val)
	}

	return res, nil
}

func (c redisCache) Clear(ctx context.Context, namespace string, key string) error {
	err := c.client.Del(
		ctx,
		c.attr(namespace, key),
	).Err()
	if err != nil {
		return fmt.Errorf("clearing redis cache entry: %w", err)
	}

	return nil
}

func (c redisCache) attr(namespace string, key string) string {
	return fmt.Sprintf("%s:%s", namespace, key)
}
