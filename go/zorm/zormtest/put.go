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
}
