package zcacheredis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/milagre/zote/go/zcache"
)

var (
	// lockScript takes or renews KEYS[1] for holder ARGV[1] for ARGV[2]
	// milliseconds, returning 1, or returns 0 while another holder has it.
	lockScript = redis.NewScript(`
local held = redis.call('GET', KEYS[1])
if held == false or held == ARGV[1] then
	redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
	return 1
end
return 0
`)

	// unlockScript deletes KEYS[1] only while holder ARGV[1] has it.
	unlockScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
end
return 0
`)
)

type redisCache struct {
	client redis.UniversalClient
}

// Client is what one redis serves here: cached entries, and the locks sharing
// their keyspace.
type Client interface {
	zcache.Cache
	zcache.Locker
}

func New(c redis.UniversalClient) Client {
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

func (c redisCache) Lock(ctx context.Context, holder string, key string, ttl time.Duration) (bool, error) {
	if ttl < time.Millisecond {
		return false, fmt.Errorf("lock ttl %s is under the millisecond redis expires by", ttl)
	}

	taken, err := lockScript.Run(
		ctx,
		c.client,
		[]string{key},
		holder,
		ttl.Milliseconds(),
	).Int()
	if err != nil {
		return false, fmt.Errorf("taking redis lock: %w", err)
	}

	return taken == 1, nil
}

func (c redisCache) Unlock(ctx context.Context, holder string, key string) (bool, error) {
	released, err := unlockScript.Run(
		ctx,
		c.client,
		[]string{key},
		holder,
	).Int()
	if err != nil {
		return false, fmt.Errorf("releasing redis lock: %w", err)
	}

	return released == 1, nil
}

func (c redisCache) attr(namespace string, key string) string {
	return fmt.Sprintf("%s:%s", namespace, key)
}
