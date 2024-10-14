package zcache

import (
	"context"
	"fmt"
	"time"

	"github.com/milagre/zote/go/zwarn"
)

type Cache interface {
	Set(ctx context.Context, namespace string, key string, expiration time.Duration, value []byte) error
	Get(ctx context.Context, namespace string, key string) (<-chan []byte, error)
}

type Loader func(ctx context.Context) ([]byte, error)
type Parser[T any] func(data []byte) (T, error)

func ReadThrough[T any](
	ctx context.Context,
	cache Cache,
	namespace string,
	key string,
	expiration time.Duration,
	loader Loader,
	parser Parser[T],
) (T, zwarn.Warning, error) {
	var warnings zwarn.Warnings
	var err error
	var result T
	var loaded bool

	// Load data from cache first
	ch, err := cache.Get(ctx, namespace, key)
	if err != nil {
		warnings = append(warnings, zwarn.Warnf("loading read-through cached data: %v", err))
	} else {
		if data, ok := <-ch; ok {
			var found T
			found, err = parser(data)
			if err != nil {
				warnings = append(warnings, zwarn.Warnf("parsing read-through cached data: %v", err))
			} else {
				loaded = true
				result = found
			}
		}
	}

	// If not found in cache, load from source
	if !loaded {
		content, err := loader(ctx)
		if err != nil {
			return result, warnings, fmt.Errorf("fetching read-through from source: %w", err)
		}

		var found T
		found, err = parser(content)
		if err != nil {
			return result, warnings, fmt.Errorf("parsing read-through from source: %w", err)
		} else {
			loaded = true
			result = found

			err := cache.Set(ctx, namespace, key, expiration, content)
			if err != nil {
				warnings = append(warnings, zwarn.Warnf("caching read-through data from source: %v", err))
			}
		}
	}

	if len(warnings) != 0 {
		return result, warnings, nil
	}

	return result, nil, nil

}
