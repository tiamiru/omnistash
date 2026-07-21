package metastoretest

import (
	"context"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
)

const (
	TestReferrerArtifactType = "application/vnd.example.referrer"
	TestReferrerMediaType    = "application/vnd.oci.image.manifest.v1+json"
)

var (
	TestSubjectDigest  = digest.FromString("metastoretest-subject")  //nolint:gochecknoglobals
	TestReferrerDigest = digest.FromString("metastoretest-referrer") //nolint:gochecknoglobals
)

func ExerciseReferrerOpsContract(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("ReferrerOps", func(t *testing.T) {
		t.Parallel()

		t.Run("UpsertReferrer", func(t *testing.T) {
			t.Parallel()
			exerciseUpsertReferrer(t, newStore)
		})

		t.Run("ListReferrers", func(t *testing.T) {
			t.Parallel()
			exerciseListReferrers(t, newStore)
		})

		t.Run("DeleteReferrer", func(t *testing.T) {
			t.Parallel()
			exerciseDeleteReferrer(t, newStore)
		})
	})
}

func seedReferrer(t *testing.T, store metastore.MetadataStore, row metastore.ReferrerRow) {
	t.Helper()
	mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
		return tx.UpsertReferrer(ctx, DefaultName, row)
	})
}

func defaultReferrerRow() metastore.ReferrerRow {
	return metastore.ReferrerRow{
		SubjectDigest: TestSubjectDigest,
		Digest:        TestReferrerDigest,
		MediaType:     TestReferrerMediaType,
		ArtifactType:  TestReferrerArtifactType,
		Size:          TestSize,
		Annotations:   map[string]string{"key": "value"},
	}
}

func exerciseUpsertReferrer(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: insert then list returns row", func(t *testing.T) {
		t.Parallel()
		exerciseUpsertReferrerHappyPath(t, newStore)
	})

	t.Run("edge case: upsert is idempotent — overwrites prior entry", func(t *testing.T) {
		t.Parallel()
		exerciseUpsertReferrerIdempotent(t, newStore)
	})

	t.Run("edge case: nil annotations stored and returned as empty map", func(t *testing.T) {
		t.Parallel()
		exerciseUpsertReferrerNilAnnotations(t, newStore)
	})
}

func exerciseUpsertReferrerHappyPath(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()
	store := newStore(t)
	seedNamespace(t, store)
	seedReferrer(t, store, defaultReferrerRow())

	err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
		rows, listErr := tx.ListReferrers(ctx, DefaultName, TestSubjectDigest)
		if listErr != nil {
			return listErr
		}

		require.Len(t, rows, 1)
		assert.Equal(t, TestSubjectDigest, rows[0].SubjectDigest)
		assert.Equal(t, TestReferrerDigest, rows[0].Digest)
		assert.Equal(t, TestReferrerMediaType, rows[0].MediaType)
		assert.Equal(t, TestReferrerArtifactType, rows[0].ArtifactType)
		assert.Equal(t, TestSize, rows[0].Size)
		assert.Equal(t, map[string]string{"key": "value"}, rows[0].Annotations)

		return nil
	})
	require.NoError(t, err)
}

func exerciseUpsertReferrerIdempotent(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()
	store := newStore(t)
	seedNamespace(t, store)
	seedReferrer(t, store, defaultReferrerRow())

	updated := defaultReferrerRow()
	updated.ArtifactType = "application/vnd.example.updated"
	seedReferrer(t, store, updated)

	err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
		rows, listErr := tx.ListReferrers(ctx, DefaultName, TestSubjectDigest)
		if listErr != nil {
			return listErr
		}

		require.Len(t, rows, 1)
		assert.Equal(t, "application/vnd.example.updated", rows[0].ArtifactType)

		return nil
	})
	require.NoError(t, err)
}

func exerciseUpsertReferrerNilAnnotations(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()
	store := newStore(t)
	seedNamespace(t, store)

	row := defaultReferrerRow()
	row.Annotations = nil
	seedReferrer(t, store, row)

	err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
		rows, listErr := tx.ListReferrers(ctx, DefaultName, TestSubjectDigest)
		if listErr != nil {
			return listErr
		}

		require.Len(t, rows, 1)
		assert.Equal(t, map[string]string{}, rows[0].Annotations)

		return nil
	})
	require.NoError(t, err)
}

func exerciseListReferrers(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: unknown subject returns empty slice", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedNamespace(t, store)

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			rows, listErr := tx.ListReferrers(ctx, DefaultName, UnknownDigest)
			if listErr != nil {
				return listErr
			}

			assert.Empty(t, rows)

			return nil
		})
		require.NoError(t, err)
	})

	t.Run("happy path: multiple referrers for same subject", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedNamespace(t, store)

		otherReferrerDigest := digest.FromString("metastoretest-referrer-2")

		row1 := defaultReferrerRow()
		row2 := defaultReferrerRow()
		row2.Digest = otherReferrerDigest

		seedReferrer(t, store, row1)
		seedReferrer(t, store, row2)

		const wantCount = 2

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			rows, listErr := tx.ListReferrers(ctx, DefaultName, TestSubjectDigest)
			if listErr != nil {
				return listErr
			}

			assert.Len(t, rows, wantCount)

			return nil
		})
		require.NoError(t, err)
	})
}

func exerciseDeleteReferrer(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: deleting a referrer removes it from the list", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedNamespace(t, store)
		seedReferrer(t, store, defaultReferrerRow())

		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			return tx.DeleteReferrer(ctx, DefaultName, TestReferrerDigest)
		})

		err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
			rows, listErr := tx.ListReferrers(ctx, DefaultName, TestSubjectDigest)
			if listErr != nil {
				return listErr
			}

			assert.Empty(t, rows)

			return nil
		})
		require.NoError(t, err)
	})

	t.Run("edge case: deleting a non-existent referrer is a no-op", func(t *testing.T) {
		t.Parallel()
		store := newStore(t)
		seedNamespace(t, store)

		mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
			return tx.DeleteReferrer(ctx, DefaultName, UnknownDigest)
		})
	})
}
