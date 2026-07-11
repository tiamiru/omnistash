package blobstoretest

import (
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/blobstore"
)

func ExerciseBlobWriterContract(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("PutBlob", func(t *testing.T) {
		t.Parallel()
		exercisePutBlobTable(t, newStore)
		exercisePutBlobCrossAlgorithm(t, newStore)
	})

	t.Run("StatBlob", func(t *testing.T) {
		t.Parallel()
		exerciseStatBlobTable(t, newStore)
	})
}

func exercisePutBlobTable(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	testCases := []struct {
		name           string
		seed           bool
		digest         digest.Digest
		size           int64
		content        string
		wantErr        error
		wantSize       int64
		wantBlobAbsent digest.Digest
	}{
		{
			name:    "error path: malformed digest returns ErrInvalidDigest",
			digest:  MalformedDigest,
			size:    int64(len(TestContent)),
			content: TestContent,
			wantErr: blobstore.ErrInvalidDigest,
		},
		{
			name:    "error path: negative size rejected",
			digest:  TestDigest,
			size:    -1,
			content: TestContent,
			wantErr: blobstore.ErrSizeInvalid,
		},
		{
			name:    "error path: size mismatch returns ErrSizeInvalid",
			digest:  TestDigest,
			size:    int64(len(TestContent)) + 1,
			content: TestContent,
			wantErr: blobstore.ErrSizeInvalid,
		},
		{
			name:           "error path: digest mismatch",
			digest:         ZeroDigest,
			size:           int64(len(TestContent)),
			content:        TestContent,
			wantErr:        blobstore.ErrDigestMismatch,
			wantBlobAbsent: ZeroDigest,
		},
		{
			name:    "error path: returns ErrBlobCommitted when blob already in CAS",
			seed:    true,
			digest:  TestDigest,
			size:    int64(len(TestContent)),
			content: TestContent,
			wantErr: blobstore.ErrBlobCommitted,
		},
		{
			name:     "happy path: stores blob",
			digest:   TestDigest,
			size:     int64(len(TestContent)),
			content:  TestContent,
			wantSize: int64(len(TestContent)),
		},
		{
			name:     "happy path: stores blob addressed with a sha512 digest",
			digest:   TestDigest512,
			size:     int64(len(TestContent)),
			content:  TestContent,
			wantSize: int64(len(TestContent)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(t, t.Name(), DefaultPartition)
			if tc.seed {
				seedTestBlob(t, s)
			}

			size, err := s.PutBlob(tc.digest, tc.size, strings.NewReader(tc.content))

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Equal(t, int64(0), size)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantSize, size)
			}
			if tc.wantBlobAbsent != "" {
				_, statErr := s.StatBlob(tc.wantBlobAbsent)
				require.ErrorIs(t, statErr, blobstore.ErrBlobUnknown)
			}
		})
	}
}

func exerciseStatBlobTable(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	testCases := []struct {
		name     string
		seed     bool
		digest   digest.Digest
		wantErr  error
		wantSize int64
	}{
		{
			name:    "error path: malformed digest returns ErrInvalidDigest",
			digest:  MalformedDigest,
			wantErr: blobstore.ErrInvalidDigest,
		},
		{
			name:    "error path: unknown digest returns ErrBlobUnknown",
			digest:  TestDigest,
			wantErr: blobstore.ErrBlobUnknown,
		},
		{
			name:     "happy path: returns size of known blob",
			seed:     true,
			digest:   TestDigest,
			wantSize: int64(len(TestContent)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(t, t.Name(), DefaultPartition)
			if tc.seed {
				seedTestBlob(t, s)
			}

			size, err := s.StatBlob(tc.digest)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Equal(t, int64(0), size)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantSize, size)
			}
		})
	}
}

func exercisePutBlobCrossAlgorithm(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	name := "happy path: same content is independently committable under a different digest algorithm"
	t.Run(name, func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		_, err := s.PutBlob(TestDigest, int64(len(TestContent)), strings.NewReader(TestContent))
		require.NoError(t, err)
		_, err = s.PutBlob(TestDigest512, int64(len(TestContent)), strings.NewReader(TestContent))
		require.NoError(t, err, "the same content under a different digest algorithm is a distinct CAS entry")

		sizeSHA256, err := s.StatBlob(TestDigest)
		require.NoError(t, err)
		assert.Equal(t, int64(len(TestContent)), sizeSHA256)

		sizeSHA512, err := s.StatBlob(TestDigest512)
		require.NoError(t, err)
		assert.Equal(t, int64(len(TestContent)), sizeSHA512)
	})
}
