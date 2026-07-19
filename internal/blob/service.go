package blob

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/ocierror"
)

var ErrMountFailed = errors.New("blob mount not available")

type Service struct {
	meta  metastore.MetadataStore
	blobs blobstore.BlobStore
}

func NewService(meta metastore.MetadataStore, blobs blobstore.BlobStore) *Service {
	return &Service{meta: meta, blobs: blobs}
}

// StatBlob returns the registered size of d scoped to name. Used for HEAD responses.
func (s *Service) StatBlob(ctx context.Context, name string, d digest.Digest) (int64, error) {
	err := validateNamespaceDigest(ctx, s.meta, name, d)
	if err != nil {
		return 0, fmt.Errorf("StatBlob: %w", err)
	}

	size, err := s.statBlob(ctx, name, d)
	if err != nil {
		return 0, fmt.Errorf("StatBlob: %w", err)
	}

	return size, nil
}

// GetBlob returns a reader for the full blob content and its size.
func (s *Service) GetBlob(ctx context.Context, name string, d digest.Digest) (io.ReadCloser, int64, error) {
	err := validateNamespaceDigest(ctx, s.meta, name, d)
	if err != nil {
		return nil, 0, fmt.Errorf("GetBlob: %w", err)
	}

	size, err := s.statBlob(ctx, name, d)
	if err != nil {
		return nil, 0, fmt.Errorf("GetBlob: %w", err)
	}

	rc, _, err := s.blobs.GetBlob(name, d)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUnknown) {
			return nil, 0, fmt.Errorf("GetBlob: %w", ocierror.ErrBlobUnknown)
		}

		return nil, 0, fmt.Errorf("GetBlob: %w", err)
	}

	return rc, size, nil
}

// GetBlobRange returns a reader for the range [first, last] of the provided digest
// and the total blob size.
// Returns ErrRangeNotSatisfiable (with non-zero totalSize) if the range exceeds the blob.
func (s *Service) GetBlobRange(
	ctx context.Context,
	name string,
	d digest.Digest,
	first, last int64,
) (io.ReadCloser, int64, error) {
	err := validateNamespaceDigest(ctx, s.meta, name, d)
	if err != nil {
		return nil, 0, fmt.Errorf("GetBlobRange: %w", err)
	}

	totalSize, err := s.statBlob(ctx, name, d)
	if err != nil {
		return nil, 0, fmt.Errorf("GetBlobRange: %w", err)
	}

	if last >= totalSize {
		return nil, totalSize, fmt.Errorf(
			"GetBlobRange: last=%d size=%d: %w",
			last,
			totalSize,
			ocierror.ErrRangeNotSatisfiable,
		)
	}

	pr, pw := io.Pipe()

	go func() {
		pw.CloseWithError(s.blobs.GetBlobRange(name, d, first, last, pw))
	}()

	return pr, totalSize, nil
}
