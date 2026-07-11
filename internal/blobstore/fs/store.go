package fs

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blobstore"
	blobs "github.com/tiamiru/omnistash/internal/blobstore"
)

var _ blobs.BlobStore = &FilesystemBlobStore{}

type FilesystemBlobStore struct {
	prefix    string
	partition blobs.PartitionKey
}

func NewFilesystemBlobStore(prefix string, partition blobs.PartitionKey) *FilesystemBlobStore {
	return &FilesystemBlobStore{
		prefix:    prefix,
		partition: partition,
	}
}

func (s *FilesystemBlobStore) PutBlob(d digest.Digest, size int64, r io.Reader) (int64, error) {
	err := blobs.ValidateDigest(d)
	if err != nil {
		return 0, fmt.Errorf("put blob %s: %w", d, err)
	}

	if size < 0 {
		return 0, fmt.Errorf("put blob %s: %w: got %d", d, blobs.ErrSizeInvalid, size)
	}

	tmpPath, n, err := s.writeBlobToStaging(d, size, r)
	if err != nil {
		return 0, fmt.Errorf("put blob %s: %w", d, err)
	}

	defer func() {
		_ = removeFileIfExists(tmpPath) //nolint:errcheck // will log with logger.Warn once logger is added
	}()

	err = s.linkStagedFile(tmpPath, d)
	if err != nil {
		return 0, fmt.Errorf("put blob %s: %w", d, err)
	}

	return n, nil
}

func (s *FilesystemBlobStore) StatBlob(d digest.Digest) (int64, error) {
	err := blobs.ValidateDigest(d)
	if err != nil {
		return 0, fmt.Errorf("stat blob %s: %w", d, err)
	}

	p := buildBlobPath(s.prefix, string(s.partition), d)
	fi, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("%w: digest=%s", blobs.ErrBlobUnknown, d)
		}

		return 0, fmt.Errorf("stat blob %s: stat: %w", d, err)
	}

	return fi.Size(), nil
}

func (s *FilesystemBlobStore) writeBlobToStaging(d digest.Digest, size int64, r io.Reader) (string, int64, error) {
	h, err := blobs.HasherFor(d)
	if err != nil {
		return "", 0, err
	}

	tmpPath, n, err := s.writeToStaging(io.TeeReader(r, h))
	if err != nil {
		return "", 0, err
	}

	if n != size {
		_ = os.Remove(tmpPath) //nolint:errcheck // will log with logger.Warn once logger is added
		return "", 0, fmt.Errorf("%w: got %d, want %d", blobs.ErrSizeInvalid, n, size)
	}

	actual := digest.NewDigest(d.Algorithm(), h)
	err = blobstore.MatchDigest(d, actual)
	if err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // will log with logger.Warn once logger is added
		return "", 0, err
	}

	return tmpPath, n, nil
}

func (s *FilesystemBlobStore) writeToStaging(r io.Reader) (_ string, _ int64, err error) {
	stagingDir := filepath.Join(s.prefix, string(s.partition), ".staging")
	err = os.MkdirAll(stagingDir, 0o750)
	if err != nil {
		return "", 0, fmt.Errorf("mkdir staging: %w", err)
	}

	tmpPath := buildStagingPath(s.prefix, string(s.partition), uuid.New().String())
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return "", 0, fmt.Errorf("create tmp: %w", err)
	}

	defer func() {
		closeErr := f.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close tmp: %w", closeErr)
		}
		if err != nil {
			_ = os.Remove(tmpPath) //nolint:errcheck // will log with logger.Warn once logger is added
		}
	}()

	n, err := io.Copy(f, r)
	if err != nil {
		return "", 0, fmt.Errorf("write: %w", err)
	}

	return tmpPath, n, nil
}

func (s *FilesystemBlobStore) linkStagedFile(tmpPath string, d digest.Digest) error {
	destPath := buildBlobPath(s.prefix, string(s.partition), d)
	destDir := filepath.Dir(destPath)

	err := os.MkdirAll(destDir, 0o750)
	if err != nil {
		return fmt.Errorf("mkdir blob dir: %w", err)
	}

	err = os.Link(tmpPath, destPath)
	if err != nil {
		if errors.Is(err, iofs.ErrExist) {
			return fmt.Errorf("%w: digest=%s", blobs.ErrBlobCommitted, d)
		}

		return fmt.Errorf("link: %w", err)
	}

	return nil
}
