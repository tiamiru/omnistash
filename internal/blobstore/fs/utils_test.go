package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveFileIfExists(t *testing.T) {
	t.Parallel()

	t.Run("happy path: removes existing file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		p := filepath.Join(dir, "file.txt")
		err := os.WriteFile(p, []byte("data"), 0o600)
		require.NoError(t, err)

		err = removeFileIfExists(p)
		require.NoError(t, err)
		assert.NoFileExists(t, p)
	})

	t.Run("edge case: no error when file absent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		p := filepath.Join(dir, "nonexistent.txt")
		err := removeFileIfExists(p)
		require.NoError(t, err)
	})
}
