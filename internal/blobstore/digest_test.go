package blobstore_test

import (
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"

	blobs "github.com/tiamiru/omnistash/internal/blobstore"
	blobstest "github.com/tiamiru/omnistash/internal/blobstore/blobstoretest"
)

func TestValidateDigest(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		input   digest.Digest
		wantErr error
	}{
		{
			name:  "happy path: valid sha256 digest",
			input: blobstest.TestDigest,
		},
		{
			name:    "error path: missing sha256 prefix",
			input:   "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
			wantErr: blobs.ErrInvalidDigest,
		},
		{
			name:    "error path: hex too short",
			input:   "sha256:abc123",
			wantErr: blobs.ErrInvalidDigest,
		},
		{
			name:    "error path: invalid hex character",
			input:   "sha256:x4136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
			wantErr: blobs.ErrInvalidDigest,
		},
		{
			name:    "error path: uppercase hex rejected",
			input:   "sha256:44136FA355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
			wantErr: blobs.ErrInvalidDigest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := blobs.ValidateDigest(tc.input)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMatchDigest(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		expected digest.Digest
		actual   digest.Digest
		wantErr  error
	}{
		{
			name:     "happy path: identical digests",
			expected: blobstest.TestDigest,
			actual:   blobstest.TestDigest,
		},
		{
			name:     "error path: mismatched digests",
			expected: blobstest.TestDigest,
			actual:   blobstest.OtherDigest,
			wantErr:  blobs.ErrDigestMismatch,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := blobs.MatchDigest(tc.expected, tc.actual)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
