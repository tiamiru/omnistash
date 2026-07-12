package fs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/logtag"
)

func (s *FilesystemBlobStore) InitiateBlobUpload() (string, error) {
	stagingDir := filepath.Join(s.prefix, string(s.partition), ".staging")
	err := os.MkdirAll(stagingDir, 0o750)
	if err != nil {
		return "", fmt.Errorf("initiate blob upload: mkdir staging: %w", err)
	}

	uploadID := uuid.New().String()
	stagingPath := buildStagingPath(s.prefix, string(s.partition), uploadID)

	f, err := os.OpenFile( //nolint:gosec // path built from constructor args + UUID
		stagingPath,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return "", fmt.Errorf("initiate blob upload: create staging: %w", err)
	}

	closeErr := f.Close()
	if closeErr != nil {
		return "", fmt.Errorf("initiate blob upload: close staging: %w", closeErr)
	}

	return uploadID, nil
}

func (s *FilesystemBlobStore) AppendBlobChunk(uploadID string, offset int64, r io.Reader) (_ int64, err error) {
	if offset < 0 {
		return 0, fmt.Errorf("%w: offset=%d", blobstore.ErrBlobUploadInvalid, offset)
	}

	stagingPath := buildStagingPath(s.prefix, string(s.partition), uploadID)

	f, err := os.OpenFile( //nolint:gosec // path built from constructor args + uploadID
		stagingPath,
		os.O_WRONLY|os.O_APPEND,
		0o600,
	)
	if err != nil {
		return 0, fmt.Errorf("append blob chunk: %w", mapNotExistErr(err, blobstore.ErrBlobUploadUnknown))
	}
	defer func() {
		closeErr := f.Close()
		if closeErr != nil {
			s.logger.Warn("AppendBlobChunk: close staging", logtag.Path(stagingPath), logtag.Err(closeErr))
		}
	}()

	fi, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("append blob chunk: stat: %w", err)
	}

	if fi.Size() != offset {
		return 0, fmt.Errorf("%w: offset=%d size=%d", blobstore.ErrRangeNotSatisfiable, offset, fi.Size())
	}

	written, err := io.Copy(f, r)
	if err != nil {
		return 0, fmt.Errorf("append blob chunk: write: %w", err)
	}

	err = f.Sync()
	if err != nil {
		return 0, fmt.Errorf("append blob chunk: sync: %w", err)
	}

	return offset + written, nil
}

func (s *FilesystemBlobStore) GetBlobUploadOffset(uploadID string) (int64, error) {
	stagingPath := buildStagingPath(s.prefix, string(s.partition), uploadID)

	fi, err := os.Stat(stagingPath)
	if err != nil {
		return 0, fmt.Errorf("get blob upload offset: %w", mapNotExistErr(err, blobstore.ErrBlobUploadUnknown))
	}

	return fi.Size(), nil
}

func (s *FilesystemBlobStore) FinalizeBlobUpload(uploadID string, d digest.Digest, size int64) error {
	err := blobstore.ValidateDigest(d)
	if err != nil {
		return fmt.Errorf("finalize blob upload: %w", err)
	}

	stagingPath := buildStagingPath(s.prefix, string(s.partition), uploadID)

	err = s.verifyStagingBlob(stagingPath, d, size)
	if err != nil {
		return fmt.Errorf("finalize blob upload: %w", err)
	}

	err = s.commitStagingBlob(stagingPath, d)
	if err != nil {
		return fmt.Errorf("finalize blob upload: %w", err)
	}

	return nil
}

func (s *FilesystemBlobStore) verifyStagingBlob(stagingPath string, d digest.Digest, size int64) error {
	fi, err := os.Stat(stagingPath)
	if err != nil {
		return mapNotExistErr(err, blobstore.ErrBlobUploadUnknown)
	}

	if fi.Size() != size {
		return fmt.Errorf("%w: got %d, want %d", blobstore.ErrSizeInvalid, fi.Size(), size)
	}

	h, err := blobstore.HasherFor(d)
	if err != nil {
		return err
	}

	f, err := os.Open(stagingPath) //nolint:gosec // path built from constructor args + uploadID
	if err != nil {
		return fmt.Errorf("open staging: %w", err)
	}
	defer func() {
		closeErr := f.Close()
		if closeErr != nil {
			s.logger.Warn("FinalizeBlobUpload: close staging", logtag.Path(stagingPath), logtag.Err(closeErr))
		}
	}()

	_, err = io.Copy(h, f)
	if err != nil {
		return fmt.Errorf("hash staging: %w", err)
	}

	actual := digest.NewDigest(d.Algorithm(), h)

	return blobstore.MatchDigest(d, actual)
}

func (s *FilesystemBlobStore) commitStagingBlob(stagingPath string, d digest.Digest) error {
	err := s.linkStagedFile(stagingPath, d)
	if err != nil && !errors.Is(err, blobstore.ErrBlobCommitted) {
		return err
	}

	removeErr := removeFileIfExists(stagingPath)
	if removeErr != nil {
		s.logger.Warn("FinalizeBlobUpload: remove staging", logtag.Path(stagingPath), logtag.Err(removeErr))
	}

	return err
}

func (s *FilesystemBlobStore) CancelBlobUpload(uploadID string) error {
	stagingPath := buildStagingPath(s.prefix, string(s.partition), uploadID)

	err := removeFileIfExists(stagingPath)
	if err != nil {
		return fmt.Errorf("cancel blob upload: %w", err)
	}

	return nil
}
