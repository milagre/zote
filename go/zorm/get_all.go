package zorm

import (
	"context"
	"fmt"
)

func GetChunked[T any](ctx context.Context, repo Repository, list []*T, pageSize int, opts GetOptions) error {
	if len(list) == 0 {
		return nil
	}

	length := len(list)

	for start := 0; start < length; start += pageSize {
		end := start + pageSize
		if end > length {
			end = length
		}

		err := Get(ctx, repo, list[start:end], opts)
		if err != nil {
			return fmt.Errorf("getting chunked at index %d-%d out of %d: %w", start, end, length, err)
		}
	}

	return nil
}
