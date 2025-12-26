package zcache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/zcache"
)

type testData struct {
	Value string
}

// mockCache is a mock implementation of the Cache interface for testing
type mockCache struct {
	getError error
	getData  []byte
	setError error
	setCalls []setCalls
}

type setCalls struct {
	namespace  string
	key        string
	expiration time.Duration
	value      []byte
}

func testDataMarshaller(data testData) ([]byte, error) {
	return []byte(data.Value), nil
}

func testDataUnmarshaller(data []byte) (testData, error) {
	return testData{Value: string(data)}, nil
}

func (m *mockCache) Get(ctx context.Context, namespace string, key string) (<-chan []byte, error) {
	ch := make(chan []byte, 1)
	defer close(ch)

	if m.getError != nil {
		return ch, m.getError
	}

	if m.getData != nil {
		ch <- m.getData
	}
	return ch, nil
}

func (m *mockCache) Set(ctx context.Context, namespace string, key string, expiration time.Duration, value []byte) error {
	m.setCalls = append(m.setCalls, setCalls{
		namespace:  namespace,
		key:        key,
		expiration: expiration,
		value:      value,
	})
	return m.setError
}

func TestReadThrough_CacheHit(t *testing.T) {
	ctx := context.Background()
	cache := &mockCache{
		getData: []byte("cached-value"),
	}

	loaderCalled := false
	loader := func(ctx context.Context) (testData, error) {
		loaderCalled = true
		return testData{Value: "loaded-value"}, nil
	}

	result, warnings, err := zcache.ReadThrough(
		ctx,
		cache,
		"test-namespace",
		"test-key",
		time.Hour,
		loader,
		testDataMarshaller,
		testDataUnmarshaller,
	)

	assert.NoError(t, err)
	assert.Nil(t, warnings)
	assert.Equal(t, "cached-value", result.Value)
	assert.False(t, loaderCalled, "loader should not be called on cache hit")
	assert.Empty(t, cache.setCalls, "cache.Set should not be called on cache hit")
}

func TestReadThrough_CacheMiss(t *testing.T) {
	ctx := context.Background()
	cache := &mockCache{
		getData: nil, // No data in cache
	}

	loaderCalled := false
	loader := func(ctx context.Context) (testData, error) {
		loaderCalled = true
		return testData{Value: "loaded-value"}, nil
	}

	result, warnings, err := zcache.ReadThrough(
		ctx,
		cache,
		"test-namespace",
		"test-key",
		time.Hour,
		loader,
		testDataMarshaller,
		testDataUnmarshaller,
	)

	assert.NoError(t, err)
	assert.Nil(t, warnings)
	assert.Equal(t, "loaded-value", result.Value)
	assert.True(t, loaderCalled, "loader should be called on cache miss")

	// Verify cache was updated
	assert.Len(t, cache.setCalls, 1)
	assert.Equal(t, "test-namespace", cache.setCalls[0].namespace)
	assert.Equal(t, "test-key", cache.setCalls[0].key)
	assert.Equal(t, time.Hour, cache.setCalls[0].expiration)
	assert.Equal(t, []byte("loaded-value"), cache.setCalls[0].value)
}

func TestReadThrough_CacheGetError(t *testing.T) {
	ctx := context.Background()
	cache := &mockCache{
		getError: errors.New("cache connection failed"),
	}

	loaderCalled := false
	loader := func(ctx context.Context) (testData, error) {
		loaderCalled = true
		return testData{Value: "loaded-value"}, nil
	}

	result, warnings, err := zcache.ReadThrough(
		ctx,
		cache,
		"test-namespace",
		"test-key",
		time.Hour,
		loader,
		testDataMarshaller,
		testDataUnmarshaller,
	)

	assert.NoError(t, err)
	assert.NotNil(t, warnings)
	assert.Contains(t, warnings.Warning(), "loading read-through cached data")
	assert.Equal(t, "loaded-value", result.Value)
	assert.True(t, loaderCalled, "loader should be called when cache.Get fails")

	// Verify cache was updated with fresh data
	assert.Len(t, cache.setCalls, 1)
}

func TestReadThrough_UnmarshalError(t *testing.T) {
	ctx := context.Background()
	cache := &mockCache{
		getData: []byte("invalid-data"),
	}

	loaderCalled := false
	loader := func(ctx context.Context) (testData, error) {
		loaderCalled = true
		return testData{Value: "loaded-value"}, nil
	}

	result, warnings, err := zcache.ReadThrough(
		ctx,
		cache,
		"test-namespace",
		"test-key",
		time.Hour,
		loader,
		testDataMarshaller,
		func(data []byte) (testData, error) {
			return testData{}, errors.New("unmarshal failed")
		},
	)

	assert.NoError(t, err)
	assert.NotNil(t, warnings)
	assert.Contains(t, warnings.Warning(), "parsing read-through cached data")
	assert.Equal(t, "loaded-value", result.Value)
	assert.True(t, loaderCalled, "loader should be called when unmarshal fails")

	// Verify cache was updated with fresh data
	assert.Len(t, cache.setCalls, 1)
}

func TestReadThrough_LoaderError(t *testing.T) {
	ctx := context.Background()
	cache := &mockCache{
		getData: nil, // Cache miss
	}

	expectedError := errors.New("source unavailable")
	loader := func(ctx context.Context) (testData, error) {
		return testData{}, expectedError
	}

	result, warnings, err := zcache.ReadThrough(
		ctx,
		cache,
		"test-namespace",
		"test-key",
		time.Hour,
		loader,
		testDataMarshaller,
		testDataUnmarshaller,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching read-through from source")
	assert.Contains(t, err.Error(), "source unavailable")
	assert.Equal(t, testData{}, result)

	// Warnings may exist from cache Get, but error is fatal
	if warnings != nil {
		t.Logf("Warnings present: %v", warnings)
	}

	// Cache should not be updated on loader error
	assert.Empty(t, cache.setCalls)
}

func TestReadThrough_MarshalError(t *testing.T) {
	ctx := context.Background()
	cache := &mockCache{
		getData: nil, // Cache miss
	}

	loader := func(ctx context.Context) (testData, error) {
		return testData{Value: "loaded-value"}, nil
	}

	result, warnings, err := zcache.ReadThrough(
		ctx,
		cache,
		"test-namespace",
		"test-key",
		time.Hour,
		loader,
		func(data testData) ([]byte, error) {
			return nil, errors.New("marshal failed")
		},
		testDataUnmarshaller,
	)

	assert.NoError(t, err, "marshal error should be non-fatal")
	assert.NotNil(t, warnings)
	assert.Contains(t, warnings.Warning(), "caching read-through data from source")
	assert.Equal(t, "loaded-value", result.Value)

	// Cache should not be updated when marshal fails
	assert.Empty(t, cache.setCalls)
}

func TestReadThrough_SetError(t *testing.T) {
	ctx := context.Background()
	cache := &mockCache{
		getData:  nil, // Cache miss
		setError: errors.New("cache write failed"),
	}

	loader := func(ctx context.Context) (testData, error) {
		return testData{Value: "loaded-value"}, nil
	}

	result, warnings, err := zcache.ReadThrough(
		ctx,
		cache,
		"test-namespace",
		"test-key",
		time.Hour,
		loader,
		testDataMarshaller,
		testDataUnmarshaller,
	)

	assert.NoError(t, err, "cache.Set error should be non-fatal")
	assert.NotNil(t, warnings)
	assert.Contains(t, warnings.Warning(), "caching read-through data from source")
	assert.Equal(t, "loaded-value", result.Value)

	// Verify Set was attempted
	assert.Len(t, cache.setCalls, 1)
}

func TestReadThrough_MultipleWarnings(t *testing.T) {
	type contextKey string
	key := contextKey("test-key")
	ctx := context.WithValue(context.Background(), key, "test-value")

	cache := &mockCache{
		getError: errors.New("cache get failed"),
		setError: errors.New("cache set failed"),
	}

	var receivedCtx context.Context
	loader := func(ctx context.Context) (testData, error) {
		receivedCtx = ctx
		return testData{Value: "loaded-value"}, nil
	}

	result, warnings, err := zcache.ReadThrough(
		ctx,
		cache,
		"test-namespace",
		"test-key",
		time.Hour,
		loader,
		testDataMarshaller,
		testDataUnmarshaller,
	)

	assert.NoError(t, err)
	assert.NotNil(t, warnings)

	// Should have both cache get and set warnings
	warningStr := warnings.Warning()
	assert.Contains(t, warningStr, "loading read-through cached data")
	assert.Contains(t, warningStr, "caching read-through data from source")

	// Data should still be returned successfully
	assert.Equal(t, "loaded-value", result.Value)

	// Verify Set was attempted despite Get failure
	assert.Len(t, cache.setCalls, 1)

	// Verify context was provided
	assert.Equal(t, "test-value", receivedCtx.Value(key))
}

func TestReadThrough_EmptyCacheData(t *testing.T) {
	ctx := context.Background()
	cache := &mockCache{
		getData: []byte(""), // Empty but valid data
	}

	loaderCalled := false
	loader := func(ctx context.Context) (testData, error) {
		loaderCalled = true
		return testData{Value: "loaded-value"}, nil
	}

	result, warnings, err := zcache.ReadThrough(
		ctx,
		cache,
		"test-namespace",
		"test-key",
		time.Hour,
		loader,
		testDataMarshaller,
		testDataUnmarshaller,
	)

	assert.NoError(t, err)
	assert.Nil(t, warnings)
	assert.Equal(t, "", result.Value)
	assert.False(t, loaderCalled, "loader should not be called when empty data is valid")
}
