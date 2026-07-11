package blobstoretest

import (
	"bytes"
	"io"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/blobstore"
)

const (
	rangeFirstAfterLast  = 5
	rangeLastBeforeFirst = 3
	rangeBeyondFileSize  = 100
)

func ExerciseBlobReaderContract(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("GetBlob", func(t *testing.T) {
		t.Parallel()
		exerciseGetBlob(t, newStore)
	})

	t.Run("GetBlobRange", func(t *testing.T) {
		t.Parallel()
		exerciseGetBlobRange(t, newStore)
	})
}

func exerciseGetBlob(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	testCases := []struct {
		name        string
		seed        bool
		digest      digest.Digest
		wantContent string
		wantSize    int64
		wantErr     error
	}{
		{
			name:    "error path: malformed digest returns ErrInvalidDigest", //nolint:goconst
			digest:  MalformedDigest,
			wantErr: blobstore.ErrInvalidDigest,
		},
		{
			name:    "error path: unknown digest returns ErrBlobUnknown", //nolint:goconst
			digest:  TestDigest,
			wantErr: blobstore.ErrBlobUnknown,
		},
		{
			name:        "happy path: returns content and size",
			seed:        true,
			digest:      TestDigest,
			wantContent: TestContent,
			wantSize:    int64(len(TestContent)),
		},
		{
			name:        "happy path: returns content and size for a sha512 digest",
			seed:        true,
			digest:      TestDigest512,
			wantContent: TestContent,
			wantSize:    int64(len(TestContent)),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(t, t.Name(), DefaultPartition)
			if tc.seed {
				seedBlob(t, s, tc.digest, TestContent)
			}

			rc, size, err := s.GetBlob(tc.digest)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, rc)
				assert.Equal(t, int64(0), size)
			} else {
				require.NoError(t, err)
				t.Cleanup(func() { assert.NoError(t, rc.Close()) })
				assert.Equal(t, tc.wantSize, size)
				data, _ := io.ReadAll(rc)
				assert.Equal(t, tc.wantContent, string(data))
			}
		})
	}
}

//nolint:funlen
func exerciseGetBlobRange(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	testCases := []struct {
		name        string
		seed        bool
		digest      digest.Digest
		first       int64
		last        int64
		wantContent string
		wantErr     error
	}{
		{
			name:    "error path: malformed digest returns ErrInvalidDigest", //nolint:goconst
			digest:  MalformedDigest,
			first:   0,
			last:    0,
			wantErr: blobstore.ErrInvalidDigest,
		},
		{
			name:    "error path: unknown digest returns ErrBlobUnknown", //nolint:goconst
			digest:  TestDigest,
			first:   0,
			last:    0,
			wantErr: blobstore.ErrBlobUnknown,
		},
		{
			name:    "error path: negative first returns ErrInvalidRange",
			seed:    true,
			digest:  TestDigest,
			first:   -1,
			last:    0,
			wantErr: blobstore.ErrInvalidRange,
		},
		{
			name:    "error path: last < first returns ErrInvalidRange",
			seed:    true,
			digest:  TestDigest,
			first:   rangeFirstAfterLast,
			last:    rangeLastBeforeFirst,
			wantErr: blobstore.ErrInvalidRange,
		},
		{
			name:    "error path: last beyond file size returns ErrRangeNotSatisfiable",
			seed:    true,
			digest:  TestDigest,
			first:   0,
			last:    int64(len(TestContent)) + rangeBeyondFileSize,
			wantErr: blobstore.ErrRangeNotSatisfiable,
		},
		{
			name:        "happy path: full content",
			seed:        true,
			digest:      TestDigest,
			first:       0,
			last:        int64(len(TestContent)) - 1,
			wantContent: TestContent,
		},
		{
			name:        "happy path: first byte",
			seed:        true,
			digest:      TestDigest,
			first:       0,
			last:        0,
			wantContent: string(TestContent[0]),
		},
		{
			name:        "happy path: last byte",
			seed:        true,
			digest:      TestDigest,
			first:       int64(len(TestContent)) - 1,
			last:        int64(len(TestContent)) - 1,
			wantContent: string(TestContent[len(TestContent)-1]),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(t, t.Name(), DefaultPartition)
			if tc.seed {
				seedTestBlob(t, s)
			}

			var buf bytes.Buffer
			err := s.GetBlobRange(tc.digest, tc.first, tc.last, &buf)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantContent, buf.String())
			}
		})
	}
}
