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
	"github.com/tiamiru/omnistash/internal/ocierror"
)

// InitiateUpload starts a new blob upload session and returns the upload ID.
func (s *Service) InitiateUpload(ctx context.Context, name string) (string, error) {
	err := validateNamespace(ctx, s.meta, name)
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
func (s *Service) MonolithicUpload(
	ctx context.Context,
	name string,
	d digest.Digest,
	size int64,
	r io.Reader,
) error {
	err := validateNamespaceDigest(ctx, s.meta, name, d)
	if err != nil {
		return fmt.Errorf("MonolithicUpload: %w", err)
	}

	actualSize, err := s.blobs.PutBlob(name, d, size, r)
	if err != nil {
		if errors.Is(err, blobstore.ErrSizeInvalid) {
			return fmt.Errorf("MonolithicUpload: %w", ocierror.ErrSizeInvalid)
		}

		if errors.Is(err, blobstore.ErrDigestMismatch) || errors.Is(err, blobstore.ErrInvalidDigest) {
			return fmt.Errorf("MonolithicUpload: %w", ocierror.ErrDigestInvalid)
		}

		return fmt.Errorf("MonolithicUpload: %w", err)
	}

	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		return tx.InsertNamespaceBlob(ctx, name, d, actualSize)
	})
	if err != nil {
		delErr := s.blobs.DeleteBlob(ctx, name, d)
		if delErr != nil {
			return errors.Join(
				fmt.Errorf("MonolithicUpload: %w", err),
				fmt.Errorf("MonolithicUpload: cleanup: %w", delErr),
			)
		}

		return fmt.Errorf("MonolithicUpload: %w", err)
	}

	return nil
}

// AppendChunk appends r at offset to the upload session and returns the new total offset.
func (s *Service) AppendChunk(
	ctx context.Context,
	name, uploadID string,
	offset int64,
	r io.Reader,
) (int64, error) {
	err := validateNamespace(ctx, s.meta, name)
	if err != nil {
		return 0, fmt.Errorf("AppendChunk: %w", err)
	}

	newOffset, err := s.blobs.AppendBlobChunk(name, uploadID, offset, r)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return 0, fmt.Errorf("AppendChunk: %w", ocierror.ErrBlobUploadUnknown)
		}

		if errors.Is(err, blobstore.ErrRangeNotSatisfiable) {
			return 0, fmt.Errorf("AppendChunk: %w", ocierror.ErrRangeNotSatisfiable)
		}

		return 0, fmt.Errorf("AppendChunk: %w", err)
	}

	return newOffset, nil
}

// CommitUpload finalizes the upload session, optionally appending finalChunk first.
// Pass nil for finalChunk when the PUT body is empty.
func (s *Service) CommitUpload(
	ctx context.Context,
	name, uploadID string,
	d digest.Digest,
	finalChunk io.Reader,
) error {
	err := validateNamespaceDigest(ctx, s.meta, name, d)
	if err != nil {
		return fmt.Errorf("CommitUpload: %w", err)
	}

	totalSize, err := s.appendFinalChunk(name, uploadID, finalChunk)
	if err != nil {
		return fmt.Errorf("CommitUpload: %w", err)
	}

	err = s.blobs.FinalizeBlobUpload(name, uploadID, d, totalSize)
	if err != nil {
		return s.handleFinalizeError(name, uploadID, err)
	}

	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		return tx.InsertNamespaceBlob(ctx, name, d, totalSize)
	})
	if err != nil {
		delErr := s.blobs.DeleteBlob(ctx, name, d)
		if delErr != nil {
			return errors.Join(fmt.Errorf("CommitUpload: %w", err), fmt.Errorf("CommitUpload: cleanup: %w", delErr))
		}

		return fmt.Errorf("CommitUpload: %w", err)
	}

	return nil
}

// GetUploadStatus returns the number of bytes received so far for the upload session.
func (s *Service) GetUploadStatus(ctx context.Context, name, uploadID string) (int64, error) {
	err := validateNamespace(ctx, s.meta, name)
	if err != nil {
		return 0, fmt.Errorf("GetUploadStatus: %w", err)
	}

	offset, err := s.blobs.GetBlobUploadOffset(name, uploadID)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return 0, fmt.Errorf("GetUploadStatus: %w", ocierror.ErrBlobUploadUnknown)
		}

		return 0, fmt.Errorf("GetUploadStatus: %w", err)
	}

	return offset, nil
}

// CancelUpload discards the upload session.
func (s *Service) CancelUpload(ctx context.Context, name, uploadID string) error {
	err := validateNamespace(ctx, s.meta, name)
	if err != nil {
		return fmt.Errorf("CancelUpload: %w", err)
	}

	err = s.blobs.CancelBlobUpload(name, uploadID)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return fmt.Errorf("CancelUpload: %w", ocierror.ErrBlobUploadUnknown)
		}

		return fmt.Errorf("CancelUpload: %w", err)
	}

	return nil
}

// MountBlob attempts to cross-mount d from sourceName into targetName.
// Returns (uploadID, false, nil) when the blob is absent in source (fall back to upload).
// Returns ("", true, nil) when the mount succeeds.
func (s *Service) MountBlob(
	ctx context.Context,
	sourceName, targetName string,
	d digest.Digest,
) (string, bool, error) {
	err := namespace.ValidateName(sourceName)
	if err != nil {
		return "", false, fmt.Errorf("MountBlob: %w", err)
	}

	// Target must exist; absent source is a fallback condition, not an error.
	err = validateNamespaceDigest(ctx, s.meta, targetName, d)
	if err != nil {
		return "", false, fmt.Errorf("MountBlob: %w", err)
	}

	sourceExists, err := s.meta.NamespaceExists(ctx, sourceName)
	if err != nil {
		return "", false, fmt.Errorf("MountBlob: %w", err)
	}

	if !sourceExists {
		return s.startFallbackUpload(targetName)
	}

	rc, sourceSize, err := s.blobs.GetBlob(sourceName, d)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUnknown) {
			return s.startFallbackUpload(targetName)
		}

		return "", false, fmt.Errorf("MountBlob: %w", err)
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

func (s *Service) handleFinalizeError(name, uploadID string, err error) error {
	if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
		return fmt.Errorf("CommitUpload: %w", ocierror.ErrBlobUploadUnknown)
	}

	if errors.Is(err, blobstore.ErrSizeInvalid) {
		return fmt.Errorf("CommitUpload: %w", ocierror.ErrSizeInvalid)
	}

	if errors.Is(err, blobstore.ErrDigestMismatch) || errors.Is(err, blobstore.ErrInvalidDigest) {
		cancelErr := s.blobs.CancelBlobUpload(name, uploadID)
		if cancelErr != nil {
			return errors.Join(
				fmt.Errorf("CommitUpload: %w", ocierror.ErrDigestInvalid),
				fmt.Errorf("CommitUpload: cancel: %w", cancelErr),
			)
		}

		return fmt.Errorf("CommitUpload: %w", ocierror.ErrDigestInvalid)
	}

	return fmt.Errorf("CommitUpload: %w", err)
}

// statBlob uses Atomic because StatNamespaceBlob is only reachable through TxOps.
func (s *Service) statBlob(ctx context.Context, name string, d digest.Digest) (int64, error) {
	var size int64

	err := s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		var statErr error

		size, statErr = tx.StatNamespaceBlob(ctx, name, d)
		if errors.Is(statErr, metastore.ErrBlobUnknown) {
			return fmt.Errorf("%w: name=%s digest=%s", ocierror.ErrBlobUnknown, name, d)
		}

		return statErr
	})

	return size, err
}

func (s *Service) appendFinalChunk(name, uploadID string, finalChunk io.Reader) (int64, error) {
	offset, err := s.blobs.GetBlobUploadOffset(name, uploadID)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return 0, ocierror.ErrBlobUploadUnknown
		}

		return 0, err
	}

	if finalChunk == nil {
		return offset, nil
	}

	newOffset, err := s.blobs.AppendBlobChunk(name, uploadID, offset, finalChunk)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUploadUnknown) {
			return 0, ocierror.ErrBlobUploadUnknown
		}

		return 0, err
	}

	return newOffset, nil
}

func (s *Service) startFallbackUpload(targetName string) (string, bool, error) {
	uid, err := s.blobs.InitiateBlobUpload(targetName)
	if err != nil {
		return "", false, fmt.Errorf("MountBlob: fallback upload: %w", err)
	}

	return uid, false, nil
}

func (s *Service) mountBlobCopy(
	ctx context.Context,
	targetName string,
	d digest.Digest,
	rc io.Reader,
	size int64,
) error {
	actualSize, putErr := s.blobs.PutBlob(targetName, d, size, rc)
	if putErr != nil && !errors.Is(putErr, blobstore.ErrBlobCommitted) {
		return fmt.Errorf("MountBlob: put: %w", putErr)
	}

	// PutBlob returns 0 for actualSize on ErrBlobCommitted, so stat to get the real size.
	var registeredSize int64
	if errors.Is(putErr, blobstore.ErrBlobCommitted) {
		var statErr error
		registeredSize, statErr = s.blobs.StatBlob(targetName, d)
		if statErr != nil {
			return fmt.Errorf("MountBlob: stat existing: %w", statErr)
		}
	} else {
		registeredSize = actualSize
	}

	regErr := s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		return tx.InsertNamespaceBlob(ctx, targetName, d, registeredSize)
	})
	if regErr != nil {
		if putErr == nil {
			delErr := s.blobs.DeleteBlob(ctx, targetName, d)
			if delErr != nil {
				return errors.Join(
					fmt.Errorf("MountBlob: register: %w", regErr),
					fmt.Errorf("MountBlob: cleanup: %w", delErr),
				)
			}
		}

		return fmt.Errorf("MountBlob: register: %w", regErr)
	}

	return nil
}
