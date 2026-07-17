package fs_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/blobstore/blobstoretest"
	"github.com/tiamiru/omnistash/internal/blobstore/fs"
)

func TestFilesystemBlobStore_Contract(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	newStore := func(t *testing.T, prefix string) blobstore.BlobStore {
		t.Helper()
		s := fs.NewFilesystemBlobStore(filepath.Join(baseDir, prefix))
		s.StartVacuumProcess()
		t.Cleanup(func() { assert.NoError(t, s.StopVacuumProcess()) })

		return s
	}
	blobstoretest.ExerciseBlobStoreContract(t, newStore)
}
