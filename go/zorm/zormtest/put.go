package zormtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zorm"
)

func RunPutTests(t *testing.T, setup SetupFunc) {
	t.Helper()

	t.Run("PutAccountNew", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &Account{
				Company: "NewCo",
			}
			err := zorm.Put(ctx, r, []*Account{obj}, zorm.PutOptions{})
			require.NoError(t, err)

			assert.NotZero(t, obj.ID)
			assert.NotNil(t, obj.Created)
			assert.Nil(t, obj.Modified)
			assert.Equal(t, "NewCo", obj.Company)
		})
	})

	t.Run("PutAccountUpdateUniqueKey", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &Account{
				Company:      "Acme, Inc.",
				ContactEmail: "updated@acme.example",
			}
			err := zorm.Put(ctx, r, []*Account{obj}, zorm.PutOptions{})
			require.NoError(t, err)

			assert.NotZero(t, obj.ID)
			assert.NotNil(t, obj.Modified)
			assert.Equal(t, "updated@acme.example", obj.ContactEmail)
		})
	})

	t.Run("PutAccountInsertByUniqueKey", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &Account{
				Company:      "NewCo",
				ContactEmail: "contact@newco.test",
			}
			err := zorm.Put(ctx, r, []*Account{obj}, zorm.PutOptions{})
			require.NoError(t, err)

			assert.NotZero(t, obj.ID)
			assert.NotZero(t, obj.Created)
		})
	})

	t.Run("PutUserFields", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &User{
				ID:        "1",
				FirstName: "Duffy",
				AccountID: "2",
			}
			err := zorm.Put(ctx, r, []*User{obj}, zorm.PutOptions{
				Include: zorm.Include{
					Fields: zorm.Fields{
						"FirstName",
					},
				},
			})
			require.NoError(t, err)

			assert.Equal(t, "1", obj.AccountID)
			assert.Equal(t, "Duffy", obj.FirstName)
		})
	})
}
