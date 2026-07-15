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

	t.Run("edge case: returns ErrNameExists when namespace already exists", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)

		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			_, err := tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			_, err := tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		require.ErrorIs(t, err, metastore.ErrNameExists)
	})

	t.Run("happy path: returns namespace on first call", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)

		var ns metastore.NamespaceRow
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			var err error
			ns, err = tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		assert.Equal(t, DefaultName, ns.Name)
		assert.False(t, ns.CreatedAt.IsZero())
		assert.False(t, ns.UpdatedAt.IsZero())

		exists, err := store.NamespaceExists(t.Context(), DefaultName)
		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func exerciseDeleteNamespace(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("edge case: returns ErrNameUnknown when namespace does not exist", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			_, err := tx.DeleteNamespace(ctx, DefaultName)

			return err
		})

		require.ErrorIs(t, err, metastore.ErrNameUnknown)
	})

	t.Run("happy path: returns deleted namespace and namespace no longer exists", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			_, err := tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		var ns metastore.NamespaceRow
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			var err error
			ns, err = tx.DeleteNamespace(ctx, DefaultName)

			return err
		})

		assert.Equal(t, DefaultName, ns.Name)
		assert.False(t, ns.CreatedAt.IsZero())
		assert.False(t, ns.UpdatedAt.IsZero())

		exists, err := store.NamespaceExists(t.Context(), DefaultName)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
