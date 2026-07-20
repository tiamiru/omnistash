package metastoretest

import (
	"context"
	"errors"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/ocierror"
)

const TestMediaType = "application/vnd.oci.image.manifest.v1+json"

var TestManifestDigest = digest.FromString("metastoretest-manifest") //nolint:gochecknoglobals

func ExerciseManifestOpsContract(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("ManifestOps", func(t *testing.T) {
		t.Parallel()

		t.Run("InsertManifest", func(t *testing.T) {
			t.Parallel()
			exerciseInsertManifest(t, newStore)
		})

		t.Run("GetManifestByDigest", func(t *testing.T) {
			t.Parallel()
			exerciseGetManifestByDigest(t, newStore)
		})

		t.Run("DeleteManifestByDigest", func(t *testing.T) {
			t.Parallel()
			exerciseDeleteManifestByDigest(t, newStore)
		})
	})
}

func seedManifest(t *testing.T, store metastore.MetadataStore, d digest.Digest) {
	t.Helper()
	mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
		_, err := tx.CreateNamespace(ctx, DefaultName)
		if err != nil && !errors.Is(err, metastore.ErrNameExists) {
			return err
		}

		return tx.InsertManifest(ctx, DefaultName, d, TestMediaType, TestSize)
	})
}

func exerciseInsertManifest(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("edge case: idempotent — calling twice does not error", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedManifest(t, store, TestManifestDigest)
		seedManifest(t, store, TestManifestDigest)

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			_, getErr := tx.GetManifestByDigest(ctx, DefaultName, TestManifestDigest)

			return getErr
		})
		require.NoError(t, err)
	})

	t.Run("happy path: InsertManifest then GetManifestByDigest returns row", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedManifest(t, store, TestManifestDigest)

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			row, getErr := tx.GetManifestByDigest(ctx, DefaultName, TestManifestDigest)
			if getErr != nil {
				return getErr
			}

			assert.Equal(t, DefaultName, row.Namespace)
			assert.Equal(t, TestManifestDigest, row.Digest)
			assert.Equal(t, TestMediaType, row.MediaType)
			assert.Equal(t, TestSize, row.Size)

			return nil
		})
		require.NoError(t, err)
	})
}

func exerciseGetManifestByDigest(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("error path: absent digest returns ErrManifestUnknown", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			_, err := tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			_, getErr := tx.GetManifestByDigest(ctx, DefaultName, TestManifestDigest)

			return getErr
		})
		require.ErrorIs(t, err, ocierror.ErrManifestUnknown)
	})

	t.Run("error path: soft-deleted manifest returns ErrManifestUnknown", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedManifest(t, store, TestManifestDigest)

		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			return tx.DeleteManifestByDigest(ctx, DefaultName, TestManifestDigest)
		})

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			_, getErr := tx.GetManifestByDigest(ctx, DefaultName, TestManifestDigest)

			return getErr
		})
		require.ErrorIs(t, err, ocierror.ErrManifestUnknown)
	})
}

func exerciseDeleteManifestByDigest(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("error path: absent manifest returns ErrManifestUnknown", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			_, err := tx.CreateNamespace(ctx, DefaultName)

			return err
		})

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			return tx.DeleteManifestByDigest(ctx, DefaultName, TestManifestDigest)
		})
		require.ErrorIs(t, err, ocierror.ErrManifestUnknown)
	})

	t.Run("happy path: soft-deletes manifest and makes it unreachable by digest", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedManifest(t, store, TestManifestDigest)

		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			return tx.DeleteManifestByDigest(ctx, DefaultName, TestManifestDigest)
		})

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			_, getErr := tx.GetManifestByDigest(ctx, DefaultName, TestManifestDigest)

			return getErr
		})
		require.ErrorIs(t, err, ocierror.ErrManifestUnknown)
	})
}
