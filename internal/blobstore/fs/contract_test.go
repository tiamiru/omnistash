package fs_test

import (
	"path/filepath"
	"testing"

	blobs "github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/blobstore/blobstoretest"
	"github.com/tiamiru/omnistash/internal/blobstore/fs"
)

func TestFilesystemBlobStore_Contract(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	newStore := func(t *testing.T, prefix string, partition blobs.PartitionKey) blobs.BlobStore {
		t.Helper()

		return fs.NewFilesystemBlobStore(filepath.Join(baseDir, prefix), partition)
	}
	blobstoretest.ExerciseBlobStoreContract(t, newStore)
}
