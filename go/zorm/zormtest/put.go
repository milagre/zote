package zormtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zelem"
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

			// Only FirstName was included, so only FirstName is read back
			assert.Equal(t, "Duffy", obj.FirstName)
			assert.Equal(t, "2", obj.AccountID)
		})
	})

	// Cascading Put tests

	t.Run("PutUserWithNewAccount", func(t *testing.T) {
		// Tests cascading put where FK is local (account_id on users table).
		// Account should be inserted FIRST, then its generated ID copied to User.
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			newAccount := &Account{
				Company: "CascadeTestCo",
			}
			obj := &User{
				FirstName: "CascadeUser",
				Account:   newAccount,
			}
			err := zorm.Put(ctx, r, []*User{obj}, zorm.PutOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Account": zorm.Relation{},
					},
				},
			})
			require.NoError(t, err)

			// User should have been inserted with Account's ID as FK
			assert.NotZero(t, obj.ID)
			assert.NotZero(t, obj.AccountID)

			// Account should have been refreshed with its generated values
			require.NotNil(t, obj.Account)
			assert.Equal(t, "CascadeTestCo", obj.Account.Company)
			assert.NotZero(t, obj.Account.Created)
		})
	})

	t.Run("PutAccountWithNewUsers", func(t *testing.T) {
		// Tests cascading put of to-many relation where FK is remote.
		// Account should be inserted FIRST, then Users get Account's ID.
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			obj := &Account{
				Company: "MultiUserCo",
				Users: []*User{
					{FirstName: "User1"},
					{FirstName: "User2"},
				},
			}
			err := zorm.Put(ctx, r, []*Account{obj}, zorm.PutOptions{
				Include: zorm.Include{
					Relations: zorm.Relations{
						"Users": zorm.Relation{},
					},
				},
			})
			require.NoError(t, err)

			// Account should have been inserted
			assert.NotZero(t, obj.ID)

			// Users should have been refreshed with their generated values
			require.Len(t, obj.Users, 2)
			for _, user := range obj.Users {
				assert.NotZero(t, user.ID)
				assert.Equal(t, obj.ID, user.AccountID)
			}
		})
	})

	// Orphan deletion tests for to-many relations

	t.Run("PutToManyRelationSync", func(t *testing.T) {
		t.Run("UserAuthsDeletesOrphansWithFilter", func(t *testing.T) {
			// User 1 has auths: password (id=1), oauth2 (id=2)
			// Put with Where: provider="password" and empty list - should delete password auth only.
			// The oauth2 auth should remain untouched.
			setup(t, func(ctx context.Context, r zorm.Repository) {
				ctx = makeContext(ctx)

				user := getUserWithAuths(ctx, t, r, "1")
				require.Len(t, user.Auths, 2)

				// Put with filter for password provider and empty list
				passwordFilter := zelem.Eq(zelem.Field("Provider"), zelem.Value("password"))
				user.Auths = []*UserAuth{}
				err := zorm.Put(ctx, r, []*User{user}, zorm.PutOptions{Include: authsInclude(passwordFilter)})
				require.NoError(t, err)

				// Re-fetch to verify: should have only oauth2 remaining
				user2 := getUserWithAuths(ctx, t, r, "1")
				require.Len(t, user2.Auths, 1, "Should have only 1 auth remaining")
				assert.Equal(t, "oauth2", user2.Auths[0].Provider, "oauth2 auth should remain")
			})
		})

		t.Run("UserAuthsEmptySliceDeletesAll", func(t *testing.T) {
			// User 1 has auths: password (id=1), oauth2 (id=2)
			// Put with empty Auths slice - should delete all auths.
			setup(t, func(ctx context.Context, r zorm.Repository) {
				ctx = makeContext(ctx)

				user := getUserWithAuths(ctx, t, r, "1")
				require.Len(t, user.Auths, 2)

				// Put with empty slice
				user.Auths = []*UserAuth{}
				err := zorm.Put(ctx, r, []*User{user}, zorm.PutOptions{Include: authsInclude(nil)})
				require.NoError(t, err)

				// Re-fetch to verify: should have no auths
				user2 := getUserWithAuths(ctx, t, r, "1")
				assert.Empty(t, user2.Auths, "All auths should be deleted")
			})
		})

		t.Run("UserAuthsUpdatesExistingAndDeletesOrphans", func(t *testing.T) {
			// User 1 has auths: password (id=1), oauth2 (id=2)
			// Put with updated password auth and new sso auth - oauth2 should be deleted.
			setup(t, func(ctx context.Context, r zorm.Repository) {
				ctx = makeContext(ctx)

				user := getUserWithAuths(ctx, t, r, "1")
				require.Len(t, user.Auths, 2)
				originalAuths := authsByProvider(user.Auths)

				// Put with existing password (by ID) updated, plus new sso
				user.Auths = []*UserAuth{
					{ID: originalAuths["password"].ID, Provider: "password", Data: "updated-hash"},
					{Provider: "sso", Data: "new-sso-data"},
				}
				err := zorm.Put(ctx, r, []*User{user}, zorm.PutOptions{Include: authsInclude(nil)})
				require.NoError(t, err)
				require.Len(t, user.Auths, 2)

				// Re-fetch to verify
				user2 := getUserWithAuths(ctx, t, r, "1")
				require.Len(t, user2.Auths, 2)

				auths := authsByProvider(user2.Auths)
				assert.Equal(t, "updated-hash", auths["password"].Data, "password auth should be updated")
				assert.Equal(t, "new-sso-data", auths["sso"].Data, "sso auth should be added")
				assert.Nil(t, auths["oauth2"], "oauth2 should be deleted")
			})
		})

		t.Run("UserAuthsUpdateFieldOnly", func(t *testing.T) {
			// User 1 has auths: password (id=1), oauth2 (id=2)
			// Put with both auths, updating the Data field of password - both should remain.
			setup(t, func(ctx context.Context, r zorm.Repository) {
				ctx = makeContext(ctx)

				user := getUserWithAuths(ctx, t, r, "1")
				require.Len(t, user.Auths, 2)
				originalAuths := authsByProvider(user.Auths)

				// Update only the password auth's Data field, keep everything else
				user.Auths = []*UserAuth{
					{ID: originalAuths["password"].ID, Provider: "password", Data: "new-password-hash"},
					{ID: originalAuths["oauth2"].ID, Provider: "oauth2", Data: originalAuths["oauth2"].Data},
				}
				err := zorm.Put(ctx, r, []*User{user}, zorm.PutOptions{Include: authsInclude(nil)})
				require.NoError(t, err)

				// Re-fetch to verify
				user2 := getUserWithAuths(ctx, t, r, "1")
				require.Len(t, user2.Auths, 2, "Should still have 2 auths")

				auths := authsByProvider(user2.Auths)
				assert.Equal(t, originalAuths["password"].ID, auths["password"].ID, "password auth ID unchanged")
				assert.Equal(t, "new-password-hash", auths["password"].Data, "password auth data updated")
				assert.Equal(t, originalAuths["oauth2"].ID, auths["oauth2"].ID, "oauth2 auth ID unchanged")
				assert.Equal(t, originalAuths["oauth2"].Data, auths["oauth2"].Data, "oauth2 auth data unchanged")
			})
		})

		t.Run("UserAuthsInsertNewWithFilter", func(t *testing.T) {
			// User 1 has auths: password (id=1), oauth2 (id=2)
			// Put with Where: provider="sso" (no existing matches) and a new sso auth.
			// Existing password and oauth2 auths should be completely untouched.
			setup(t, func(ctx context.Context, r zorm.Repository) {
				ctx = makeContext(ctx)

				user := getUserWithAuths(ctx, t, r, "1")
				require.Len(t, user.Auths, 2)
				originalAuths := authsByProvider(user.Auths)

				// Put with filter for "sso" provider (doesn't exist yet) and add new sso auth
				ssoFilter := zelem.Eq(zelem.Field("Provider"), zelem.Value("sso"))
				user.Auths = []*UserAuth{{Provider: "sso", Data: "new-sso-token"}}
				err := zorm.Put(ctx, r, []*User{user}, zorm.PutOptions{Include: authsInclude(ssoFilter)})
				require.NoError(t, err)

				// Re-fetch ALL auths (no filter) to verify
				user2 := getUserWithAuths(ctx, t, r, "1")
				require.Len(t, user2.Auths, 3, "Should have 3 auths: password, oauth2, and new sso")

				auths := authsByProvider(user2.Auths)
				assert.Equal(t, originalAuths["password"].ID, auths["password"].ID, "password auth untouched")
				assert.Equal(t, originalAuths["oauth2"].ID, auths["oauth2"].ID, "oauth2 auth untouched")
				assert.NotEmpty(t, auths["sso"].ID, "sso auth should be added")
				assert.Equal(t, "new-sso-token", auths["sso"].Data, "sso auth has correct data")
			})
		})
	})
}

// getUserWithAuths fetches user by ID with Auths relation included.
func getUserWithAuths(ctx context.Context, t *testing.T, r zorm.Repository, userID string) *User {
	t.Helper()
	user := &User{ID: userID}
	err := zorm.Get(ctx, r, []*User{user}, zorm.GetOptions{
		Include: authsInclude(nil),
	})
	require.NoError(t, err)
	return user
}

// authsInclude creates an Include for the Auths relation with optional Where clause.
func authsInclude(where zclause.Clause) zorm.Include {
	return zorm.Include{
		Relations: zorm.Relations{
			"Auths": zorm.Relation{Where: where},
		},
	}
}

// authsByProvider converts a slice of UserAuth to a map keyed by Provider.
func authsByProvider(auths []*UserAuth) map[string]*UserAuth {
	result := make(map[string]*UserAuth)
	for _, auth := range auths {
		result[auth.Provider] = auth
	}
	return result
}
