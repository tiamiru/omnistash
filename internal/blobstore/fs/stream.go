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

func (s *FilesystemBlobStore) InitiateBlobUpload(namespace string) (string, error) {
	stagingDir := filepath.Join(s.prefix, namespace, ".staging")
	err := os.MkdirAll(stagingDir, 0o750)
	if err != nil {
		return "", fmt.Errorf("InitiateBlobUpload: namespace=%s: mkdir staging: %w", namespace, err)
	}

	uploadID := uuid.New().String()
	stagingPath := buildStagingPath(s.prefix, namespace, uploadID)

	f, err := os.OpenFile( //nolint:gosec // path built from constructor args + UUID
		stagingPath,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return "", fmt.Errorf("InitiateBlobUpload: namespace=%s: create staging: %w", namespace, err)
	}

	closeErr := f.Close()
	if closeErr != nil {
		return "", fmt.Errorf("InitiateBlobUpload: namespace=%s: close staging: %w", namespace, closeErr)
	}

	return uploadID, nil
}

func (s *FilesystemBlobStore) AppendBlobChunk(namespace, uploadID string, offset int64, r io.Reader) (int64, error) {
	if offset < 0 {
		return 0, fmt.Errorf("%w: offset=%d", blobstore.ErrBlobUploadInvalid, offset)
	}

	stagingPath := buildStagingPath(s.prefix, namespace, uploadID)

	f, err := os.OpenFile( //nolint:gosec // path built from constructor args + uploadID
		stagingPath,
		os.O_WRONLY|os.O_APPEND,
		0o600,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"AppendBlobChunk: namespace=%s upload_id=%s: %w",
			namespace,
			uploadID,
			mapNotExistErr(err, blobstore.ErrBlobUploadUnknown),
		)
	}
	defer func() {
		closeErr := f.Close()
		if closeErr != nil {
			s.logger.Warn("AppendBlobChunk: close staging", logtag.Path(stagingPath), logtag.Err(closeErr))
		}
	}()

	fi, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("AppendBlobChunk: namespace=%s upload_id=%s: stat: %w", namespace, uploadID, err)
	}

	if fi.Size() != offset {
		return 0, fmt.Errorf("%w: offset=%d size=%d", blobstore.ErrRangeNotSatisfiable, offset, fi.Size())
	}

	written, err := io.Copy(f, r)
	if err != nil {
		return 0, fmt.Errorf("AppendBlobChunk: namespace=%s upload_id=%s: write: %w", namespace, uploadID, err)
	}

	err = f.Sync()
	if err != nil {
		return 0, fmt.Errorf("AppendBlobChunk: namespace=%s upload_id=%s: sync: %w", namespace, uploadID, err)
	}

	return offset + written, nil
}

func (s *FilesystemBlobStore) GetBlobUploadOffset(namespace, uploadID string) (int64, error) {
	stagingPath := buildStagingPath(s.prefix, namespace, uploadID)

	fi, err := os.Stat(stagingPath)
	if err != nil {
		return 0, fmt.Errorf(
			"GetBlobUploadOffset: namespace=%s upload_id=%s: %w",
			namespace,
			uploadID,
			mapNotExistErr(err, blobstore.ErrBlobUploadUnknown),
		)
	}

	return fi.Size(), nil
}

func (s *FilesystemBlobStore) FinalizeBlobUpload(namespace, uploadID string, d digest.Digest, size int64) error {
	err := blobstore.ValidateDigest(d)
	if err != nil {
		return fmt.Errorf("FinalizeBlobUpload: namespace=%s upload_id=%s digest=%s: %w", namespace, uploadID, d, err)
	}

	stagingPath := buildStagingPath(s.prefix, namespace, uploadID)

	err = s.verifyStagingBlob(stagingPath, d, size)
	if err != nil {
		return fmt.Errorf("FinalizeBlobUpload: namespace=%s upload_id=%s digest=%s: %w", namespace, uploadID, d, err)
	}

	err = s.commitStagingBlob(namespace, stagingPath, d)
	if err != nil {
		return fmt.Errorf("FinalizeBlobUpload: namespace=%s upload_id=%s digest=%s: %w", namespace, uploadID, d, err)
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
			s.logger.Warn("verifyStagingBlob: close staging", logtag.Path(stagingPath), logtag.Err(closeErr))
		}
	}()

	_, err = io.Copy(h, f)
	if err != nil {
		return fmt.Errorf("hash staging: %w", err)
	}

	actual := digest.NewDigest(d.Algorithm(), h)

	return blobstore.MatchDigest(d, actual)
}

func (s *FilesystemBlobStore) commitStagingBlob(namespace, stagingPath string, d digest.Digest) error {
	err := s.linkStagedFile(namespace, stagingPath, d)
	if err != nil && !errors.Is(err, blobstore.ErrBlobCommitted) {
		return err
	}

	removeErr := removeFileIfExists(stagingPath)
	if removeErr != nil {
		s.logger.Warn("commitStagingBlob: remove staging", logtag.Path(stagingPath), logtag.Err(removeErr))
	}

	return err
}

func (s *FilesystemBlobStore) CancelBlobUpload(namespace, uploadID string) error {
	stagingPath := buildStagingPath(s.prefix, namespace, uploadID)

	err := removeFileIfExists(stagingPath)
	if err != nil {
		return fmt.Errorf("CancelBlobUpload: namespace=%s upload_id=%s: %w", namespace, uploadID, err)
	}

	return nil
}
