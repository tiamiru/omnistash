package metastoretest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
)

func ExerciseNamespaceOpsContract(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("NamespaceOps", func(t *testing.T) {
		t.Parallel()

		t.Run("CreateNamespace", func(t *testing.T) {
			t.Parallel()
			exerciseCreateNamespace(t, newStore)
		})

		t.Run("DeleteNamespace", func(t *testing.T) {
			t.Parallel()
			exerciseDeleteNamespace(t, newStore)
		})
	})
}

func exerciseCreateNamespace(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: returns created=true on first call", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)

		var created bool
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			var err error
			created, err = tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		assert.True(t, created)
		exists, err := store.NamespaceExists(t.Context(), DefaultName)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("edge case: returns created=false when namespace already exists", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)

		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			_, err := tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		var created bool
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			var err error
			created, err = tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		assert.False(t, created)
	})
}

func exerciseDeleteNamespace(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: returns deleted=true and namespace no longer exists", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			_, err := tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		var deleted bool
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			var err error
			deleted, err = tx.DeleteNamespace(ctx, DefaultName)

			return err
		})

		assert.True(t, deleted)
		exists, err := store.NamespaceExists(t.Context(), DefaultName)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("edge case: returns deleted=false when namespace does not exist", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)

		var deleted bool
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			var err error
			deleted, err = tx.DeleteNamespace(ctx, DefaultName)

			return err
		})

		assert.False(t, deleted)
	})
}
