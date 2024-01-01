package zormsqlite3_test

import (
	"context"
	"testing"

	"github.com/milagre/zote/go/zorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleGet_Account(t *testing.T) {
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

func TestSimpleGet_AccountFields(t *testing.T) {
	setup(t, func(ctx context.Context, r zorm.Repository) {
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
}

func TestSimpleGet_User(t *testing.T) {
	setup(t, func(ctx context.Context, r zorm.Repository) {
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
}

/*
func TestGetRelationObject(t *testing.T) {
	setup(t, func(ctx context.Context, r zorm.Repository) {
		obj := &User{
			ID: "1",
		}
		err := zorm.Get[User](ctx, r, []*User{obj}, zorm.GetOptions{
			Include: zorm.Include{
				Relations: zorm.Relations{
					"Account": zorm.Include{},
				},
			},
		})
		require.NoError(t, err)

		assert.Equal(t, "1", obj.ID)
		assert.NotNil(t, obj.Account)
	})
}
*/
