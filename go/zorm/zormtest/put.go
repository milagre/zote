package zormtest

import (
	"context"
	"testing"

	"github.com/milagre/zote/go/zorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RunPutTests(t *testing.T, setup SetupFunc) {
	t.Helper()

	t.Run("PutAccountNew", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &Account{
				Company: "NewCo",
			}
			err := zorm.Put[Account](ctx, r, []*Account{obj}, zorm.PutOptions{})
			require.NoError(t, err)

			assert.NotZero(t, obj.ID)
			assert.NotNil(t, obj.Created)
			assert.Nil(t, obj.Modified)
			assert.Equal(t, "NewCo", obj.Company)
		})
	})

	t.Run("PutAccountUpdate", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &Account{
				ID:      "1",
				Company: "Acme, Incorporated",
			}
			err := zorm.Put[Account](ctx, r, []*Account{obj}, zorm.PutOptions{})
			require.NoError(t, err)

			assert.NotZero(t, obj.ID)
			assert.NotNil(t, obj.Created)
			assert.NotNil(t, obj.Modified)
			assert.Equal(t, "Acme, Incorporated", obj.Company)
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
			err := zorm.Put[User](ctx, r, []*User{obj}, zorm.PutOptions{
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
