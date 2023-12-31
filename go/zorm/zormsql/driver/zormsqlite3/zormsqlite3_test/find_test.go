package zormsqlite3_test

import (
	"context"
	"testing"

	"github.com/milagre/zote/go/zorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleFind(t *testing.T) {
	setup(t, func(ctx context.Context, r zorm.Repository) {
		list := make([]*Account, 0, 2)
		err := zorm.Find[Account](ctx, r, &list, zorm.FindOptions{})
		require.NoError(t, err)

		require.Len(t, list, 2)
		var acc *Account

		acc = list[0]
		assert.Equal(t, "1", acc.ID)
		assert.NotNil(t, acc.Created)
		assert.NotNil(t, acc.Modified)
		assert.Equal(t, "Acme, Inc.", acc.Company)

		acc = list[1]
		assert.Equal(t, "2", acc.ID)
		assert.NotNil(t, acc.Created)
		assert.Nil(t, acc.Modified)
		assert.Equal(t, "Dunder Mifflin", acc.Company)
	})
}
