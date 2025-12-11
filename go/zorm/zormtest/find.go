package zormtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zelement/zmethod"
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

	t.Run("FindAccountsViaNestedRelation", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			list := make([]*Account, 0, 10)
			err := zorm.Find[Account](ctx, r, &list, zorm.FindOptions{
				Where: zelem.Eq(zelem.Field("Users.Address.State"), zelem.Value("PA")),
			})
			require.NoError(t, err)
			require.Equal(t, len(list), 1, "Should have exactly 1 account")
		})
	})

	t.Run("FindAccountsSortedByNestedRelation", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			list := make([]*Account, 0, 10)
			err := zorm.Find[Account](ctx, r, &list, zorm.FindOptions{
				Sort: zelem.Sorts(zelem.Asc(zelem.Field("Users.Address.State"))),
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Users": zorm.Relation{
							Include: zorm.Include{
								Relations: zorm.Relations{
									"Address": zorm.Relation{},
								},
							},
						},
					},
				},
			})
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(list), 2, "Should have at least 2 accounts")

			// Verify results are sorted by Users.Address.State
			for i := 0; i < len(list)-1; i++ {
				currentAccount := list[i]
				nextAccount := list[i+1]

				currentMinState := getMinUserAddressState(currentAccount)
				nextMinState := getMinUserAddressState(nextAccount)

				if currentMinState != "" && nextMinState != "" {
					assert.LessOrEqual(t, currentMinState, nextMinState,
						"Account %s (min state: %s) should come before Account %s (min state: %s)",
						currentAccount.ID, currentMinState, nextAccount.ID, nextMinState)
				}
			}
		})
	})

	t.Run("FindAccountsViaNestedRelationWithInclude", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			list := make([]*Account, 0, 10)
			err := zorm.Find[Account](ctx, r, &list, zorm.FindOptions{
				Where: zelem.Eq(zelem.Field("Users.Address.State"), zelem.Value("PA")),
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Users": zorm.Relation{
							Include: zorm.Include{
								Relations: zorm.Relations{
									"Address": zorm.Relation{},
								},
							},
						},
					},
				},
			})
			require.NoError(t, err)
			require.Equal(t, len(list), 1, "Should have exactly 1 account")

			for _, account := range list {
				if assert.GreaterOrEqual(t, len(account.Users), 1, "Should have at least 1 user") {
					foundPA := false
					for _, user := range account.Users {
						if user.Address.State == "PA" {
							foundPA = true
						}
					}
					assert.True(t, foundPA, "Should have at least one user with Address.State = PA")
				}
			}
		})
	})

	t.Run("FindAccountsViaNestedRelationInMethod", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			list := make([]*Account, 0, 10)
			err := zorm.Find[Account](ctx, r, &list, zorm.FindOptions{
				Where: zelem.Truthy(
					zmethod.NewContains(
						zelem.Field("Users.Address.State"),
						zelem.Value("PA"),
					),
				),
			})
			require.NoError(t, err)
			require.Equal(t, len(list), 1, "Should have exactly 1 account")
		})
	})
}

// getMinUserAddressState returns the minimum (alphabetically first) State value
// from all users' addresses in the account, or empty string if none exist
func getMinUserAddressState(account *Account) string {
	if len(account.Users) == 0 {
		return ""
	}

	minState := ""
	for _, user := range account.Users {
		if user.Address != nil && user.Address.State != "" {
			if minState == "" || user.Address.State < minState {
				minState = user.Address.State
			}
		}
	}
	return minState
}
