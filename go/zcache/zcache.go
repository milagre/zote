// Package zcache provides a cache abstraction with read-through cache support.
//
// # Read-Through Pattern
//
// ReadThrough handles cache lookup, source fetching, and caching automatically:
//
//	user, warnings, err := zcache.ReadThrough(
//		ctx, cache,
//		"users",                        // namespace
//		fmt.Sprintf("user:%d", userID), // key
//		time.Hour,                      // expiration
//		func(ctx context.Context) (UserData, error) {
//			return database.FetchUser(ctx, userID)
//		},
//		json.Marshal,
//		func(data []byte) (UserData, error) {
//			var u UserData
//			return u, json.Unmarshal(data, &u)
//		},
//	)
//
// # Warning Handling
//
// ReadThrough returns (result, warning, error). Cache failures are non-fatal,
// returning warnings while still serving data from the source. Only source
// failures are fatal errors.
//
// # Locking
//
// Locker hands a key to one holder at a time. A lock is a lease: it lapses
// after its ttl unless its holder renews it, so a holder that dies cannot keep
// it. Lock keys share a keyspace with cached entries, so give them keys of
// their own:
//
//	held, err := locker.Lock(ctx, runID, "import:tenant:42", time.Hour)
//	if err != nil {
//		return fmt.Errorf("locking import: %w", err)
//	}
//	if !held {
//		return nil // another run has it
//	}
//	defer locker.Unlock(ctx, runID, "import:tenant:42")
//
// holder identifies one holder, and must be distinct per holder: two callers
// sharing a holder share the lock rather than excluding each other.
//
// A lease bounds how long a lock survives its holder, not how long its holder
// takes. A stalled process, a clock that jumps, or a failover to a replica
// that has not caught up can all leave two callers believing they hold the
// same key. Guard anything outside the lock - a file, a row, a remote call -
// with a check of its own rather than the lock alone.
//
// See zcacheredis for the Redis-based Cache and Locker implementations.
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
	Clear(ctx context.Context, namespace string, key string) error
}

type Locker interface {
	// Lock takes the lock on key for holder for ttl, reporting false while a
	// different holder has it. A holder that already has the lock keeps it,
	// for a fresh ttl, which is how a holder renews a lock it means to keep.
	Lock(ctx context.Context, holder string, key string, ttl time.Duration) (bool, error)

	// Unlock frees key if holder has it, reporting whether it did. False says
	// holder's lease was already gone.
	Unlock(ctx context.Context, holder string, key string) (bool, error)
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
			return result, nil, fmt.Errorf("fetching read-through from source: %w", err)
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
