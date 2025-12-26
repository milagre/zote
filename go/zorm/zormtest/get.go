package zormtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zorm"
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

			assertAccount(t, "1", obj)
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

			assertAccount(t, "1", obj)
			assert.Equal(t, time.Time{}, obj.Created)
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

	t.Run("GetUserToOneRelation", func(t *testing.T) {
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
				assertAccount(t, obj.AccountID, obj.Account)
			}
		})
	})

	t.Run("GetUserMultipleToOneRelations", func(t *testing.T) {
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
				assertAccount(t, "2", obj.Account)
			}

			if assert.NotNil(t, obj.Address) {
				assertAddress(t, obj.Address)
			}
		})
	})

	t.Run("GetUserToManyRelation", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &User{
				ID: "1",
			}
			err := zorm.Get[User](ctx, r, []*User{obj}, zorm.GetOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Auths": zorm.Relation{
							Sort: zelem.Sorts(zelem.Asc(zelem.Field("ID"))),
						},
					},
				},
			})
			require.NoError(t, err)

			assert.Equal(t, "1", obj.ID)

			if assert.NotNil(t, obj.Auths) {
				require.Equal(t, 2, len(obj.Auths))
			}
		})
	})

	t.Run("GetUserAllRelations", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &User{
				ID: "1",
			}
			err := zorm.Get[User](ctx, r, []*User{obj}, zorm.GetOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Account": zorm.Relation{},
						"Address": zorm.Relation{},
						"Auths":   zorm.Relation{},
					},
				},
			})
			require.NoError(t, err)

			assert.Equal(t, "1", obj.ID)

			if assert.NotNil(t, obj.Account) {
				assertAccount(t, obj.AccountID, obj.Account)
			}

			if assert.NotNil(t, obj.Address) {
				assertAddress(t, obj.Address)
			}

			if assert.NotNil(t, obj.Auths) && assert.Equal(t, 2, len(obj.Auths)) {
				assertUserAuth(t, obj.ID, obj.Auths[0])
				assertUserAuth(t, obj.ID, obj.Auths[1])
			}
		})
	})
}
