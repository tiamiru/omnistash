package blobstoretest

import (
	"testing"

	"github.com/tiamiru/omnistash/internal/blobstore"
)

type BlobStoreSetupFunc func(t *testing.T, prefix string, partition blobstore.PartitionKey) blobstore.BlobStore

// ExerciseBlobStoreContract runs the full blobstore.BlobStore contract tests.
func ExerciseBlobStoreContract(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	ExerciseBlobWriterContract(t, newStore)
	ExerciseBlobReaderContract(t, newStore)
	ExerciseBlobDeleterContract(t, newStore)
	ExerciseBlobUploaderContract(t, newStore)
}
