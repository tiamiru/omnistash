package fs

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/clock"
	"github.com/tiamiru/omnistash/internal/logtag"
)

var _ blobstore.BlobStore = &FilesystemBlobStore{}
var _ blobstore.BlobVacuumer = &FilesystemBlobStore{}

type FilesystemBlobStore struct {
	prefix        string
	logger        *slog.Logger
	clock         clock.Clock
	vacuumManager *VacuumManager
	stopVacuum    func() error
}

func NewFilesystemBlobStore(prefix string, opts ...Option) *FilesystemBlobStore {
	s := &FilesystemBlobStore{
		prefix: prefix,
		logger: slog.New(slog.DiscardHandler),
		clock:  clock.NewClock(),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.vacuumManager = newVacuumManager(s.logger)
	s.stopVacuum = func() error {
		s.logger.Warn("StopVacuumProcess: called before vacuum was started")

		return nil
	}

	return s
}

func (s *FilesystemBlobStore) PutBlob(namespace string, d digest.Digest, size int64, r io.Reader) (int64, error) {
	err := blobstore.ValidateDigest(d)
	if err != nil {
		return 0, fmt.Errorf("PutBlob: namespace=%s digest=%s: %w", namespace, d, err)
	}

	if size < 0 {
		return 0, fmt.Errorf("%w: namespace=%s digest=%s size=%d", blobstore.ErrSizeInvalid, namespace, d, size)
	}

	tmpPath := buildStagingPath(s.prefix, namespace, uuid.New().String())

	defer func() {
		removeErr := removeFileIfExists(tmpPath)
		if removeErr != nil {
			s.logger.Warn("PutBlob: remove staged file", logtag.Path(tmpPath), logtag.Err(removeErr))
		}
	}()

	n, err := s.writeBlob(namespace, tmpPath, d, size, r)
	if err != nil {
		return 0, fmt.Errorf("PutBlob: namespace=%s digest=%s: %w", namespace, d, err)
	}

	err = s.linkStagedFile(namespace, tmpPath, d)
	if err != nil {
		return 0, fmt.Errorf("PutBlob: namespace=%s digest=%s: %w", namespace, d, err)
	}

	return n, nil
}

func (s *FilesystemBlobStore) StatBlob(namespace string, d digest.Digest) (int64, error) {
	err := blobstore.ValidateDigest(d)
	if err != nil {
		return 0, fmt.Errorf("StatBlob: namespace=%s digest=%s: %w", namespace, d, err)
	}

	p := buildBlobPath(s.prefix, namespace, d)
	fi, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return 0, fmt.Errorf("%w: namespace=%s digest=%s", blobstore.ErrBlobUnknown, namespace, d)
		}

		return 0, fmt.Errorf("StatBlob: namespace=%s digest=%s: stat: %w", namespace, d, err)
	}

	return fi.Size(), nil
}

func (s *FilesystemBlobStore) writeBlob(
	namespace string,
	tmpPath string,
	d digest.Digest,
	size int64,
	r io.Reader,
) (int64, error) {
	h, err := blobstore.HasherFor(d)
	if err != nil {
		return 0, err
	}

	n, writeErr := s.writeToStaging(namespace, tmpPath, io.TeeReader(r, h))

	if n != size {
		removeErr := removeFileIfExists(tmpPath)
		if removeErr != nil {
			s.logger.Warn("writeBlob: remove staged file", logtag.Path(tmpPath), logtag.Err(removeErr))
		}

		return 0, fmt.Errorf("%w: got %d, want %d", blobstore.ErrSizeInvalid, n, size)
	}

	if writeErr != nil {
		return 0, writeErr
	}

	actual := digest.NewDigest(d.Algorithm(), h)
	err = blobstore.MatchDigest(d, actual)
	if err != nil {
		removeErr := removeFileIfExists(tmpPath)
		if removeErr != nil {
			s.logger.Warn("writeBlob: remove staged file", logtag.Path(tmpPath), logtag.Err(removeErr))
		}

		return 0, err
	}

	return n, nil
}

func (s *FilesystemBlobStore) writeToStaging(namespace, tmpPath string, r io.Reader) (_ int64, err error) {
	stagingDir := filepath.Join(s.prefix, namespace, ".staging")
	err = os.MkdirAll(stagingDir, 0o750)
	if err != nil {
		return 0, fmt.Errorf("mkdir staging: %w", err)
	}

	f, err := os.OpenFile( //nolint:gosec // path built from constructor args + UUID
		tmpPath,
		os.O_CREATE|os.O_WRONLY|os.O_EXCL,
		0o600,
	)
	if err != nil {
		return 0, fmt.Errorf("create tmp: %w", err)
	}

	defer func() {
		closeErr := f.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close tmp: %w", closeErr)
		}
		if err != nil {
			removeErr := removeFileIfExists(tmpPath)
			if removeErr != nil {
				s.logger.Warn("writeToStaging: remove staged file", logtag.Path(tmpPath), logtag.Err(removeErr))
			}
		}
	}()

	n, copyErr := io.Copy(f, r)
	if copyErr != nil {
		return n, fmt.Errorf("write: %w", copyErr)
	}

	err = f.Sync()
	if err != nil {
		return 0, fmt.Errorf("sync: %w", err)
	}

	return n, nil
}

func (s *FilesystemBlobStore) linkStagedFile(namespace, tmpPath string, d digest.Digest) error {
	destPath := buildBlobPath(s.prefix, namespace, d)
	destDir := filepath.Dir(destPath)

	err := os.MkdirAll(destDir, 0o750)
	if err != nil {
		return fmt.Errorf("mkdir blob dir: %w", err)
	}

	err = os.Link(tmpPath, destPath)
	if err != nil {
		if errors.Is(err, iofs.ErrExist) {
			return fmt.Errorf("%w: namespace=%s digest=%s", blobstore.ErrBlobCommitted, namespace, d)
		}

		return fmt.Errorf("link: %w", err)
	}

	return nil
}
