package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	knownDigest = digest.Digest("sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a")
	otherDigest = digest.Digest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
)

func TestRemoveFileIfExists(t *testing.T) {
	t.Parallel()

	t.Run("happy path: removes existing file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		p := filepath.Join(dir, "file.txt")
		err := os.WriteFile(p, []byte("data"), 0o644)
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
