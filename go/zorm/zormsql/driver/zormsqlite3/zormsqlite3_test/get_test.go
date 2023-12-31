package zormsqlite3_test

import (
	"context"
	"testing"

	"github.com/milagre/zote/go/zorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleGet(t *testing.T) {
	setup(t, func(ctx context.Context, r zorm.Repository) {
		obj := &Account{
			ID: "1",
		}
		err := zorm.Get[Account](ctx, r, []*Account{obj}, zorm.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, "1", obj.ID)
		assert.NotNil(t, obj.Created)
		assert.NotNil(t, obj.Modified)
		assert.Equal(t, "Acme, Inc.", obj.Company)
	})
}
