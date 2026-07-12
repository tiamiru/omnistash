package blobstoretest

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/blobstore"
)

func ExercisePartitionIsolationContract(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("PartitionIsolation", func(t *testing.T) {
		t.Parallel()
		exercisePartitionIsolation(t, newStore)
	})
}

func exercisePartitionIsolation(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	testCases := []struct {
		name           string
		writePartition blobstore.PartitionKey
		readPartition  blobstore.PartitionKey
		wantErr        error
	}{
		{
			name:           "happy path: blob written to a partition is visible within that partition",
			writePartition: DefaultPartition,
			readPartition:  DefaultPartition,
		},
		{
			name:           "edge case: blob written to one partition is not visible in another",
			writePartition: DefaultPartition,
			readPartition:  OtherPartition,
			wantErr:        blobstore.ErrBlobUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			prefix := t.Name()
			writer := newStore(t, prefix, tc.writePartition)
			reader := newStore(t, prefix, tc.readPartition)

			seedTestBlob(t, writer)

			rc, size, err := reader.GetBlob(TestDigest)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, rc)
				assert.Equal(t, int64(0), size)
			} else {
				require.NoError(t, err)
				t.Cleanup(func() {
					assert.NoError(t, rc.Close())
				})
				assert.Equal(t, int64(len(TestContent)), size)
				data, err := io.ReadAll(rc)
				require.NoError(t, err)
				assert.Equal(t, TestContent, string(data))
			}
		})
	}
}
