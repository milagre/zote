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

type (
	Loader[T any]       func(ctx context.Context) (T, error)
	Marshaller[T any]   func(T) ([]byte, error)
	Unmarshaller[T any] func(data []byte) (T, error)
)

func ReadThrough[T any](
	ctx context.Context,
	cache Cache,
	namespace string,
	key string,
	expiration time.Duration,
	loader Loader[T],
	marshal Marshaller[T],
	unmarshal Unmarshaller[T],
) (T, zwarn.Warning, error) {
	var warnings zwarn.Warnings
	var err error
	var result T
	var loaded bool

	getCtx, getCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer getCancel()

	// Load data from cache first
	ch, err := cache.Get(getCtx, namespace, key)
	if err != nil {
		warnings = append(warnings, zwarn.Warnf("loading read-through cached data: %v", err))
	} else {
		if data, ok := <-ch; ok {
			result, err = unmarshal(data)
			if err != nil {
				warnings = append(warnings, zwarn.Warnf("parsing read-through cached data: %v", err))
			} else {
				loaded = true
			}
		}
	}

	// If not found in cache, load from source
	if !loaded {
		result, err = loader(ctx)
		if err != nil {
			return result, warnings, fmt.Errorf("fetching read-through from source: %w", err)
		}

		data, err := marshal(result)
		if err != nil {
			warnings = append(warnings, zwarn.Warnf("caching read-through data from source: %v", err))
		} else {
			setCtx, setCancel := context.WithTimeout(ctx, 500*time.Millisecond)
			defer setCancel()

			err = cache.Set(setCtx, namespace, key, expiration, data)
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
