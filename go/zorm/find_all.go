package zorm

import (
	"context"
	"fmt"
)

func FindAll[T any](ctx context.Context, repo Repository, pageSize int, opts FindOptions, cb func(*T) error) error {
	list := make([]*T, 0, pageSize)
	opts.Offset = 0
	page := 0

	for {
		list = list[0:0]
		err := Find[T](ctx, repo, &list, opts)
		if err != nil {
			return fmt.Errorf("finding all on page %d: %w", page, err)
		}

		for i := range list {
			err = cb(list[i])
			if err != nil {
				return fmt.Errorf("from callback all on page %d, record %d: %w", page, i, err)
			}
		}

		if len(list) < pageSize {
			return nil
		}

		opts.Offset += pageSize
		page += 1
	}
}
