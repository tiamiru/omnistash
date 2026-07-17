package fs

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"
)

func buildBlobPath(prefix, namespace string, d digest.Digest) string {
	return filepath.Join(prefix, namespace, d.Algorithm().String(), d.Hex())
}

func buildStagingPath(prefix, namespace, id string) string {
	return filepath.Join(prefix, namespace, ".staging", id)
}

func removeFileIfExists(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", path, err)
	}

	return nil
}

func mapNotExistErr(err, notExistErr error) error {
	if errors.Is(err, iofs.ErrNotExist) {
		return notExistErr
	}

	return err
}
