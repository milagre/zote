package zormtest

import (
	"context"
	"testing"

	"github.com/milagre/zote/go/zorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RunGetTests(t *testing.T, setup SetupFunc) {
	t.Helper()

	t.Run("GetAccount", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

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
	})

	t.Run("GetAccountFields", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &Account{
				ID: "1",
			}
			err := zorm.Get[Account](ctx, r, []*Account{obj}, zorm.GetOptions{
				Include: zorm.Include{
					Fields: []string{"Company"},
				},
			})
			require.NoError(t, err)

			assert.Equal(t, "1", obj.ID)
			assert.Zero(t, obj.Created)
			assert.Nil(t, obj.Modified)
			assert.Equal(t, "Acme, Inc.", obj.Company)
		})
	})

	t.Run("GetUser", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &User{
				ID: "1",
			}
			err := zorm.Get[User](ctx, r, []*User{obj}, zorm.GetOptions{})
			require.NoError(t, err)

			assert.Equal(t, "1", obj.ID)
			assert.Equal(t, "1", obj.AccountID)
			assert.NotNil(t, obj.Created)
			assert.Nil(t, obj.Modified)
			assert.Equal(t, "Daffy", obj.FirstName)
		})
	})

	t.Run("GetUserRelation", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &User{
				ID: "1",
			}
			err := zorm.Get[User](ctx, r, []*User{obj}, zorm.GetOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Account": zorm.Relation{},
					},
				},
			})
			require.NoError(t, err)

			assert.Equal(t, "1", obj.ID)

			if assert.NotNil(t, obj.Account) {
				assert.Equal(t, "1", obj.Account.ID)
				assert.NotNil(t, obj.Account.Created)
				assert.NotNil(t, obj.Account.Modified)
				assert.Equal(t, "Acme, Inc.", obj.Account.Company)
			}
		})
	})

	t.Run("GetUserRelations", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &User{
				ID: "2",
			}
			err := zorm.Get[User](ctx, r, []*User{obj}, zorm.GetOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Account": zorm.Relation{},
						"Address": zorm.Relation{},
					},
				},
			})
			require.NoError(t, err)

			assert.Equal(t, "2", obj.ID)

			if assert.NotNil(t, obj.Account) {
				assert.Equal(t, "2", obj.Account.ID)
				assert.NotNil(t, obj.Account.Created)
				assert.Equal(t, "Dunder Mifflin", obj.Account.Company)
			}

			if assert.NotNil(t, obj.Address) {
				assert.Equal(t, obj.ID, obj.Address.UserID)
				assert.NotNil(t, obj.Address.Created)
				assert.Equal(t, "1725 Slough Avenue", obj.Address.Street)
				assert.Equal(t, "Scranton", obj.Address.City)
				assert.Equal(t, "PA", obj.Address.State)
			}
		})
	})
}
