package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/blobstore/blobstoretest"
	"github.com/tiamiru/omnistash/internal/blobstore/fs"
)

func TestGetBlob_StagingIsolation(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	s := fs.NewFilesystemBlobStore(baseDir, blobstoretest.DefaultPartition)

	stagingDir := filepath.Join(baseDir, string(blobstoretest.DefaultPartition), ".staging")
	err := os.MkdirAll(stagingDir, 0o750)
	require.NoError(t, err)

	stagingFile := filepath.Join(stagingDir, blobstoretest.TestDigest.Hex())
	err = os.WriteFile(stagingFile, []byte(blobstoretest.TestContent), 0o600)
	require.NoError(t, err)

	_, _, err = s.GetBlob(blobstoretest.TestDigest)
	require.ErrorIs(t, err, blobstore.ErrBlobUnknown)
}
