package zormtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zorm"
)

func RunDeleteTests(t *testing.T, setup SetupFunc) {
	t.Helper()

	t.Run("DeleteAccount", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			// Delete by primary key
			obj := &Account{ID: "1"}
			err := zorm.Delete(ctx, r, []*Account{obj}, zorm.DeleteOptions{})
			require.NoError(t, err)

			// Verify it's gone
			verify := &Account{ID: "1"}
			err = zorm.Get(ctx, r, []*Account{verify}, zorm.GetOptions{})
			require.ErrorIs(t, err, zorm.ErrNotFound)
		})
	})

	t.Run("DeleteAccountByUniqueKey", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			// Delete using unique key instead of primary key
			obj := &Account{Company: "Acme, Inc."}
			err := zorm.Delete(ctx, r, []*Account{obj}, zorm.DeleteOptions{})
			require.NoError(t, err)

			// Verify it's gone
			verify := &Account{ID: "1"}
			err = zorm.Get(ctx, r, []*Account{verify}, zorm.GetOptions{})
			require.ErrorIs(t, err, zorm.ErrNotFound)
		})
	})

	t.Run("DeleteAccountNotFound", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			// Attempt to delete non-existent record
			obj := &Account{ID: "999"}
			err := zorm.Delete(ctx, r, []*Account{obj}, zorm.DeleteOptions{})
			require.ErrorIs(t, err, zorm.ErrNotFound)
		})
	})

	t.Run("DeleteMultipleUsers", func(t *testing.T) {
		setup(t, func(ctx context.Context, r zorm.Repository) {
			ctx = makeContext(ctx)

			// Delete multiple users at once
			objs := []*User{
				{ID: "1"},
				{ID: "2"},
			}
			err := zorm.Delete(ctx, r, objs, zorm.DeleteOptions{})
			require.NoError(t, err)

			// Verify they're gone using Find
			var results []*User
			err = zorm.Find(ctx, r, &results, zorm.FindOptions{
				Where: zelem.In(
					[]zelement.Element{zelem.Field("ID")},
					[][]zelement.Element{{zelem.Value("1")}, {zelem.Value("2")}},
				),
			})
			require.NoError(t, err)
			require.Empty(t, results)
		})
	})

}
