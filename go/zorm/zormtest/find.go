package zormtest

import (
	"context"
	"sort"
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
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{})
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
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{
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
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{
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
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{
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
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{
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
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{
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
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{
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

	// Tests for relation-level Where and Sort

	t.Run("FindUserFilteredRelation", func(t *testing.T) {
		// Tests filtering a to-many relation using Where on the Relation.
		// User 1 has auths: password, oauth2
		// User 2 has auths: password, passkey
		// When filtering for provider="password", each user should only have their password auth.
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			list := make([]*User, 0, 10)
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Auths": zorm.Relation{
							Where: zelem.Eq(zelem.Field("Provider"), zelem.Value("password")),
						},
					},
				},
			})
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(list), 2, "Should have at least 2 users")

			foundUser1 := false
			for _, user := range list {
				for _, auth := range user.Auths {
					assert.Equal(
						t,
						"password",
						auth.Provider,
						"User %s auth %s should have provider=password, got %s",
						user.ID,
						auth.ID,
						auth.Provider,
					)
				}

				if user.ID == "1" {
					foundUser1 = true
					assert.Len(t, user.Auths, 1, "User 1 should have exactly 1 auth (password only)")
				}
			}

			assert.True(t, foundUser1, "User 1 should be in results")
		})
	})

	t.Run("FindUserSortedRelation", func(t *testing.T) {
		// Tests sorting a to-many relation using Sort on the Relation.
		// User 1 has auths: password (id=1), oauth2 (id=2)
		// When sorted by Provider ASC, oauth2 should come before password.
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			list := make([]*User, 0, 10)
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Auths": zorm.Relation{
							Sort: zelem.Sorts(zelem.Asc(zelem.Field("Provider"))),
						},
					},
				},
			})
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(list), 2, "Should have at least 2 users")

			// Find user 1 and verify auth order
			foundUser1 := false
			for _, user := range list {
				if user.ID == "1" {
					foundUser1 = true
					require.Len(t, user.Auths, 2, "User 1 should have 2 auths")
					assert.True(t, sort.SliceIsSorted(
						user.Auths,
						func(i, j int) bool { return user.Auths[i].Provider < user.Auths[j].Provider },
					), "User auths should be sorted by provider ASC")
					break
				}
			}

			assert.True(t, foundUser1, "User 1 should be in results")
		})
	})

	t.Run("FindUserFilteredAndSortedRelation", func(t *testing.T) {
		// Tests combining Where and Sort on a to-many relation.
		// Get users with auths where provider contains 'pass' (password, passkey), sorted by provider DESC.
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			list := make([]*User, 0, 10)
			err := zorm.Find(ctx, r, &list, zorm.FindOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Auths": zorm.Relation{
							Where: zelem.Truthy(
								zmethod.NewContains(
									zelem.Field("Provider"),
									zelem.Value("pass"),
								),
							),
							Sort: zelem.Sorts(zelem.Desc(zelem.Field("Provider"))),
						},
					},
				},
			})
			require.NoError(t, err)

			foundUser2 := false
			for _, user := range list {
				// Find user 2 who has password and passkey
				if user.ID == "2" {
					foundUser2 = true
					require.NotNil(t, user, "User 2 should be in results")
					require.Len(t, user.Auths, 2, "User 2 should have 2 auths matching 'pass'")

					assert.True(t, sort.SliceIsSorted(
						user.Auths,
						func(i, j int) bool { return user.Auths[i].Provider > user.Auths[j].Provider },
					), "User auths should be sorted by provider DESC")
					break
				}
			}
			assert.True(t, foundUser2, "User 2 should be in results")
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
