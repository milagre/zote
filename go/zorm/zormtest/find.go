package zormtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zorm"
)

type SetupFunc func(*testing.T, func(ctx context.Context, r zorm.Repository))

func RunFindTests(t *testing.T, setup SetupFunc) {
	t.Helper()

	t.Run("FindAccounts", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

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
	})

	t.Run("FindAccountsDeep", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			list := make([]*Account, 0, 2)
			err := zorm.Find[Account](ctx, r, &list, zorm.FindOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Users": zorm.Relation{
							Include: zorm.Include{
								Relations: zorm.Relations{
									"Account": zorm.Relation{},
									"Address": zorm.Relation{},
									"Auths":   zorm.Relation{},
								},
							},
						},
					},
				},
			})
			require.NoError(t, err)

			require.Len(t, list, 2)
		})
	})

	t.Run("FindUsersWithNilRelation", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			list := make([]*User, 0, 10)
			err := zorm.Find[User](ctx, r, &list, zorm.FindOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Account": zorm.Relation{},
						"Address": zorm.Relation{},
					},
				},
			})
			require.NoError(t, err)

			require.GreaterOrEqual(t, len(list), 2, "Should have at least 2 users")

			foundNilAddress := false
			foundNonNilAddress := false
			for _, user := range list {
				if assert.NotNil(t, user.Account, "User %s should have Account", user.ID) {
					assertAccount(t, user.AccountID, user.Account)
				}

				if user.Address == nil {
					foundNilAddress = true
				} else {
					foundNonNilAddress = true
					assertAddress(t, user.Address)
				}
			}

			// Verify we have both cases: at least one nil and one non-nil Address
			assert.True(t, foundNilAddress, "Should have at least one user with nil Address")
			assert.True(t, foundNonNilAddress, "Should have at least one user with non-nil Address")
		})
	})
}
