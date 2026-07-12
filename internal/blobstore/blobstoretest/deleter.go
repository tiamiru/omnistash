package blobstoretest

import (
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/blobstore"
)

func ExerciseBlobDeleterContract(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("DeleteBlob", func(t *testing.T) {
		t.Parallel()
		exerciseDeleteBlobTable(t, newStore)
	})

	t.Run("BatchDeleteBlobs", func(t *testing.T) {
		t.Parallel()
		exerciseBatchDeleteBlobsTable(t, newStore)
	})
}

func exerciseDeleteBlobTable(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	testCases := []struct {
		name    string
		seed    bool
		digest  digest.Digest
		wantErr error
	}{
		{
			name:    "error path: malformed digest returns ErrInvalidDigest",
			digest:  MalformedDigest,
			wantErr: blobstore.ErrInvalidDigest,
		},
		{
			name:    "error path: absent digest returns ErrBlobUnknown",
			digest:  TestDigest,
			wantErr: blobstore.ErrBlobUnknown,
		},
		{
			name:   "happy path: deletes existing blob",
			seed:   true,
			digest: TestDigest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(t, t.Name(), DefaultPartition)
			if tc.seed {
				seedTestBlob(t, s)
			}

			err := s.DeleteBlob(t.Context(), tc.digest)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				_, statErr := s.StatBlob(tc.digest)
				assert.ErrorIs(t, statErr, blobstore.ErrBlobUnknown)
			}
		})
	}
}

func exerciseBatchDeleteBlobsTable(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	testCases := []struct {
		name            string
		seed            bool
		digests         []digest.Digest
		wantErr         error
		wantBlobsAbsent []digest.Digest
	}{
		{
			name:    "error path: malformed digest returns ErrInvalidDigest",
			digests: []digest.Digest{MalformedDigest},
			wantErr: blobstore.ErrInvalidDigest,
		},
		{
			name:    "edge case: absent digest is a no-op and does not error",
			digests: []digest.Digest{TestDigest},
		},
		{
			name:            "happy path: deletes existing blob",
			seed:            true,
			digests:         []digest.Digest{TestDigest},
			wantBlobsAbsent: []digest.Digest{TestDigest},
		},
		{
			name:            "edge case: mixed batch deletes the known digest and returns ErrPartialDeletion",
			seed:            true,
			digests:         []digest.Digest{TestDigest, MalformedDigest},
			wantErr:         blobstore.ErrPartialDeletion,
			wantBlobsAbsent: []digest.Digest{TestDigest},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(t, t.Name(), DefaultPartition)
			if tc.seed {
				seedTestBlob(t, s)
			}

			err := s.BatchDeleteBlobs(t.Context(), tc.digests)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			for _, d := range tc.wantBlobsAbsent {
				_, statErr := s.StatBlob(d)
				assert.ErrorIs(t, statErr, blobstore.ErrBlobUnknown)
			}
		})
	}
}
