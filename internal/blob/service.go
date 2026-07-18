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

func validateNameAndDigest(name string, d digest.Digest) error {
	err := namespace.ValidateName(name)
	if err != nil {
		return fmt.Errorf("%w: %s", namespace.ErrNameInvalid, name)
	}

	err = blobstore.ValidateDigest(d)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrDigestInvalid, d)
	}

	return nil
}

func checkNamespaceExists(ctx context.Context, meta metastore.MetadataStore, name string) error {
	exists, err := meta.NamespaceExists(ctx, name)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%w: %s", namespace.ErrNameUnknown, name)
	}

	return nil
}

type Service struct {
	meta  metastore.MetadataStore
	blobs blobstore.BlobStore
}

func NewService(meta metastore.MetadataStore, blobs blobstore.BlobStore) *Service {
	return &Service{meta: meta, blobs: blobs}
}

// StatBlob returns the registered size of d scoped to name. Used for HEAD responses.
func (s *Service) StatBlob(ctx context.Context, name string, d digest.Digest) (int64, error) {
	err := validateNameAndDigest(name, d)
	if err != nil {
		return 0, fmt.Errorf("StatBlob: %w", err)
	}

	err = checkNamespaceExists(ctx, s.meta, name)
	if err != nil {
		return 0, fmt.Errorf("StatBlob: %w", err)
	}

	var size int64

	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		var statErr error

		size, statErr = tx.StatNamespaceBlob(ctx, name, d)
		if errors.Is(statErr, metastore.ErrBlobUnknown) {
			return fmt.Errorf("%w: name=%s digest=%s", ErrBlobUnknown, name, d)
		}

		return statErr
	})
	if err != nil {
		return 0, fmt.Errorf("StatBlob: %w", err)
	}

	return size, nil
}

// GetBlob returns a reader for the full blob content and its size.
func (s *Service) GetBlob(ctx context.Context, name string, d digest.Digest) (io.ReadCloser, int64, error) {
	err := validateNameAndDigest(name, d)
	if err != nil {
		return nil, 0, fmt.Errorf("GetBlob: %w", err)
	}

	err = checkNamespaceExists(ctx, s.meta, name)
	if err != nil {
		return nil, 0, fmt.Errorf("GetBlob: %w", err)
	}

	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		_, statErr := tx.GetNamespaceBlob(ctx, name, d)
		if errors.Is(statErr, metastore.ErrBlobUnknown) {
			return fmt.Errorf("%w: name=%s digest=%s", ErrBlobUnknown, name, d)
		}

		return statErr
	})
	if err != nil {
		return nil, 0, fmt.Errorf("GetBlob: %w", err)
	}

	rc, size, err := s.blobs.GetBlob(name, d)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUnknown) {
			return nil, 0, fmt.Errorf("GetBlob: %w", ErrBlobUnknown)
		}

		return nil, 0, fmt.Errorf("GetBlob: %w", err)
	}

	return rc, size, nil
}

// GetBlobRange writes the range [first, last] of the provided digest
// to the blobstore and returns the total blob size.
// Returns ErrRangeNotSatisfiable (with non-zero totalSize) if the range exceeds the blob.
func (s *Service) GetBlobRange(
	ctx context.Context,
	name string,
	d digest.Digest,
	first, last int64,
	w io.Writer,
) (int64, error) {
	err := validateNameAndDigest(name, d)
	if err != nil {
		return 0, fmt.Errorf("GetBlobRange: %w", err)
	}

	err = checkNamespaceExists(ctx, s.meta, name)
	if err != nil {
		return 0, fmt.Errorf("GetBlobRange: %w", err)
	}

	var totalSize int64

	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		var statErr error

		totalSize, statErr = tx.StatNamespaceBlob(ctx, name, d)
		if errors.Is(statErr, metastore.ErrBlobUnknown) {
			return fmt.Errorf("%w: name=%s digest=%s", ErrBlobUnknown, name, d)
		}

		return statErr
	})
	if err != nil {
		return 0, fmt.Errorf("GetBlobRange: %w", err)
	}

	if last >= totalSize {
		return totalSize, fmt.Errorf("GetBlobRange: %w: last=%d size=%d", ErrRangeNotSatisfiable, last, totalSize)
	}

	err = s.blobs.GetBlobRange(name, d, first, last, w)
	if err != nil {
		return 0, fmt.Errorf("GetBlobRange: %w", err)
	}

	return totalSize, nil
}
