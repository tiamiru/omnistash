package metastoretest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
)

func ExerciseBlobOpsContract(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("BlobOps", func(t *testing.T) {
		t.Parallel()

		t.Run("InsertNamespaceBlob", func(t *testing.T) {
			t.Parallel()
			exerciseInsertNamespaceBlob(t, newStore)
		})

		t.Run("GetNamespaceBlob", func(t *testing.T) {
			t.Parallel()
			exerciseGetNamespaceBlob(t, newStore)
		})

		t.Run("StatNamespaceBlob", func(t *testing.T) {
			t.Parallel()
			exerciseStatNamespaceBlob(t, newStore)
		})
	})
}

func exerciseInsertNamespaceBlob(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("edge case: idempotent — calling twice does not error", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedNamespaceBlob(t, store, TestDigest)
		seedNamespaceBlob(t, store, TestDigest)

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			_, getErr := tx.GetNamespaceBlob(ctx, DefaultName, TestDigest)

			return getErr
		})
		require.NoError(t, err)
	})

	t.Run("edge case: same digest can exist under multiple namespaces independently", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedNamespaceBlob(t, store, TestDigest)
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			_, err := tx.CreateNamespace(ctx, OtherName)
			if err != nil {
				return err
			}

			return tx.InsertNamespaceBlob(ctx, OtherName, TestDigest, TestSize)
		})

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			_, getErr := tx.GetNamespaceBlob(ctx, DefaultName, TestDigest)
			if getErr != nil {
				return getErr
			}

			_, getErr = tx.GetNamespaceBlob(ctx, OtherName, TestDigest)

			return getErr
		})
		require.NoError(t, err)
	})

	t.Run("happy path: InsertNamespaceBlob then GetNamespaceBlob returns size", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedNamespaceBlob(t, store, TestDigest)

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			got, getErr := tx.GetNamespaceBlob(ctx, DefaultName, TestDigest)
			assert.Equal(t, TestSize, got)

			return getErr
		})
		require.NoError(t, err)
	})
}

func exerciseGetNamespaceBlob(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()
	exerciseNamespaceBlobQuery(t, newStore, func(ctx context.Context, tx metastore.TxOps, ns string) (int64, error) {
		return tx.GetNamespaceBlob(ctx, ns, TestDigest)
	})
}

func exerciseStatNamespaceBlob(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()
	exerciseNamespaceBlobQuery(t, newStore, func(ctx context.Context, tx metastore.TxOps, ns string) (int64, error) {
		return tx.StatNamespaceBlob(ctx, ns, TestDigest)
	})
}

func exerciseNamespaceBlobQuery( //nolint:funlen
	t *testing.T,
	newStore MetadataStoreSetupFunc,
	query func(ctx context.Context, tx metastore.TxOps, ns string) (int64, error),
) {
	t.Helper()

	testCases := []struct {
		name     string
		setup    func(t *testing.T, store metastore.MetadataStore)
		ns       string
		wantSize int64
		wantErr  error
	}{
		{
			name:    "error path: absent digest returns ErrBlobUnknown",
			ns:      DefaultName,
			wantErr: metastore.ErrBlobUnknown,
		},
		{
			name: "error path: digest scoped to a different namespace returns ErrBlobUnknown",
			setup: func(t *testing.T, store metastore.MetadataStore) {
				t.Helper()
				seedNamespaceBlob(t, store, TestDigest)
			},
			ns:      OtherName,
			wantErr: metastore.ErrBlobUnknown,
		},
		{
			name: "happy path: returns size for an active blob",
			setup: func(t *testing.T, store metastore.MetadataStore) {
				t.Helper()
				seedNamespaceBlob(t, store, TestDigest)
			},
			ns:       DefaultName,
			wantSize: TestSize,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newStore(t)
			if tc.setup != nil {
				tc.setup(t, store)
			}

			var got int64
			err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
				var queryErr error
				got, queryErr = query(ctx, tx, tc.ns)

				return queryErr
			})

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantSize, got)
		})
	}
}
