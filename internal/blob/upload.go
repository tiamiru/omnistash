package blob

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/namespace"
)

type UploadService struct {
	meta  metastore.MetadataStore
	blobs blobstore.BlobStore
}

func NewUploadService(meta metastore.MetadataStore, blobs blobstore.BlobStore) *UploadService {
	return &UploadService{meta: meta, blobs: blobs}
}

// InitiateUpload starts a new blob upload session and returns the upload ID.
func (s *UploadService) InitiateUpload(ctx context.Context, name string) (string, error) {
	err := namespace.ValidateName(name)
	if err != nil {
		return "", fmt.Errorf("InitiateUpload: %w", namespace.ErrNameInvalid)
	}

	err = checkNamespaceExists(ctx, s.meta, name)
	if err != nil {
		return "", fmt.Errorf("InitiateUpload: %w", err)
	}

	uploadID, err := s.blobs.InitiateBlobUpload(name)
	if err != nil {
		return "", fmt.Errorf("InitiateUpload: %w", err)
	}

	return uploadID, nil
}

// MonolithicUpload stores r directly as blob d in namespace name.
func (s *UploadService) MonolithicUpload(
	ctx context.Context,
	name string,
	d digest.Digest,
	size int64,
	r io.Reader,
) error {
	err := validateNameAndDigest(name, d)
	if err != nil {
		return fmt.Errorf("MonolithicUpload: %w", err)
	}

	err = checkNamespaceExists(ctx, s.meta, name)
	if err != nil {
		return fmt.Errorf("MonolithicUpload: %w", err)
	}

	actualSize, err := s.blobs.PutBlob(name, d, size, r)
	if err != nil {
		if errors.Is(err, blobstore.ErrSizeInvalid) {
			return fmt.Errorf("MonolithicUpload: %w", ErrSizeInvalid)
		}

		if errors.Is(err, blobstore.ErrDigestMismatch) || errors.Is(err, blobstore.ErrInvalidDigest) {
			return fmt.Errorf("MonolithicUpload: %w", ErrDigestInvalid)
		}

		return fmt.Errorf("MonolithicUpload: %w", err)
	}

	return s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		return tx.InsertNamespaceBlob(ctx, name, d, actualSize)
	})
}

// AppendChunk appends r at offset to the upload session and returns the new total offset.
func (s *UploadService) AppendChunk(
	ctx context.Context,
	name, uploadID string,
	offset int64,
	r io.Reader,
) (int64, error) {
	newOffset, err := s.blobs.AppendBlobChunk(name, uploadID, offset, r)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return 0, fmt.Errorf("AppendChunk: %w", ErrBlobUploadUnknown)
		}

		if errors.Is(err, blobstore.ErrRangeNotSatisfiable) {
			return 0, fmt.Errorf("AppendChunk: %w", ErrRangeNotSatisfiable)
		}

		return 0, fmt.Errorf("AppendChunk: %w", err)
	}

	return newOffset, nil
}

// CommitUpload finalizes the upload session, optionally appending finalChunk first.
// Pass nil for finalChunk when the PUT body is empty.
func (s *UploadService) CommitUpload(
	ctx context.Context,
	name, uploadID string,
	d digest.Digest,
	finalChunk io.Reader,
) error {
	err := blobstore.ValidateDigest(d)
	if err != nil {
		return fmt.Errorf("CommitUpload: %w", ErrDigestInvalid)
	}

	if finalChunk != nil {
		err = s.appendFinalChunk(name, uploadID, finalChunk)
		if err != nil {
			return err
		}
	}

	totalSize, err := s.blobs.GetBlobUploadOffset(name, uploadID)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return fmt.Errorf("CommitUpload: %w", ErrBlobUploadUnknown)
		}

		return fmt.Errorf("CommitUpload: %w", err)
	}

	err = s.blobs.FinalizeBlobUpload(name, uploadID, d, totalSize)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return fmt.Errorf("CommitUpload: %w", ErrBlobUploadUnknown)
		}

		if errors.Is(err, blobstore.ErrSizeInvalid) {
			return fmt.Errorf("CommitUpload: %w", ErrSizeInvalid)
		}

		if errors.Is(err, blobstore.ErrDigestMismatch) || errors.Is(err, blobstore.ErrInvalidDigest) {
			_ = s.blobs.CancelBlobUpload(name, uploadID)

			return fmt.Errorf("CommitUpload: %w", ErrDigestInvalid)
		}

		return fmt.Errorf("CommitUpload: %w", err)
	}

	return s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		return tx.InsertNamespaceBlob(ctx, name, d, totalSize)
	})
}

// GetUploadStatus returns the number of bytes received so far for the upload session.
func (s *UploadService) GetUploadStatus(ctx context.Context, name, uploadID string) (int64, error) {
	offset, err := s.blobs.GetBlobUploadOffset(name, uploadID)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return 0, fmt.Errorf("GetUploadStatus: %w", ErrBlobUploadUnknown)
		}

		return 0, fmt.Errorf("GetUploadStatus: %w", err)
	}

	return offset, nil
}

// CancelUpload discards the upload session.
func (s *UploadService) CancelUpload(ctx context.Context, name, uploadID string) error {
	offset, err := s.blobs.GetBlobUploadOffset(name, uploadID)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return fmt.Errorf("CancelUpload: %w", ErrBlobUploadUnknown)
		}

		return fmt.Errorf("CancelUpload: %w", err)
	}

	// offset check just validates the session exists; ignore the value
	_ = offset

	err = s.blobs.CancelBlobUpload(name, uploadID)
	if err != nil {
		return fmt.Errorf("CancelUpload: %w", err)
	}

	return nil
}

// MountBlob attempts to cross-mount d from sourceName into targetName.
// Returns (uploadID, false, nil) when the blob is absent in source (fall back to upload).
// Returns ("", true, nil) when the mount succeeds.
func (s *UploadService) MountBlob(
	ctx context.Context,
	sourceName, targetName string,
	d digest.Digest,
) (string, bool, error) {
	err := validateNameAndDigest(targetName, d)
	if err != nil {
		return "", false, fmt.Errorf("MountBlob: %w", err)
	}

	err = checkNamespaceExists(ctx, s.meta, targetName)
	if err != nil {
		return "", false, fmt.Errorf("MountBlob: %w", err)
	}

	var (
		sourceSize  int64
		sourceFound bool
	)

	_ = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		var statErr error

		sourceSize, statErr = tx.StatNamespaceBlob(ctx, sourceName, d)
		if errors.Is(statErr, metastore.ErrBlobUnknown) {
			sourceFound = false

			return nil
		}
		if statErr != nil {
			return statErr
		}

		sourceFound = true

		return nil
	})

	if !sourceFound {
		return s.startFallbackUpload(targetName)
	}

	rc, _, readErr := s.blobs.GetBlob(sourceName, d)
	if readErr != nil {
		return s.startFallbackUpload(targetName)
	}

	mountErr := s.mountBlobCopy(ctx, targetName, d, rc, sourceSize)
	closeErr := rc.Close()

	if mountErr != nil {
		return "", false, mountErr
	}

	if closeErr != nil {
		return "", false, fmt.Errorf("MountBlob: close: %w", closeErr)
	}

	return "", true, nil
}

func (s *UploadService) appendFinalChunk(name, uploadID string, r io.Reader) error {
	offset, err := s.blobs.GetBlobUploadOffset(name, uploadID)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return fmt.Errorf("CommitUpload: %w", ErrBlobUploadUnknown)
		}

		return fmt.Errorf("CommitUpload: %w", err)
	}

	_, err = s.blobs.AppendBlobChunk(name, uploadID, offset, r)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return fmt.Errorf("CommitUpload: %w", ErrBlobUploadUnknown)
		}

		return fmt.Errorf("CommitUpload: %w", err)
	}

	return nil
}

func (s *UploadService) startFallbackUpload(targetName string) (string, bool, error) {
	uid, err := s.blobs.InitiateBlobUpload(targetName)
	if err != nil {
		return "", false, fmt.Errorf("MountBlob: fallback upload: %w", err)
	}

	return uid, false, nil
}

func (s *UploadService) mountBlobCopy(
	ctx context.Context,
	targetName string,
	d digest.Digest,
	rc io.Reader,
	size int64,
) error {
	_, putErr := s.blobs.PutBlob(targetName, d, size, rc)
	if putErr != nil && !errors.Is(putErr, blobstore.ErrBlobCommitted) {
		return fmt.Errorf("MountBlob: put: %w", putErr)
	}

	regErr := s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		return tx.InsertNamespaceBlob(ctx, targetName, d, size)
	})
	if regErr != nil {
		return fmt.Errorf("MountBlob: register: %w", regErr)
	}

	return nil
}
