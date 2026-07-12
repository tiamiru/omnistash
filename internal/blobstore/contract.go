package blobstore

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/opencontainers/go-digest"
)

type PartitionKey string

const (
	PartitionBlobs     PartitionKey = "blobs"
	PartitionManifests PartitionKey = "manifests"
	PartitionReferrers PartitionKey = "referrers"
)

var (
	ErrBlobUnknown         = errors.New("blob unknown")
	ErrInvalidDigest       = errors.New("invalid digest")
	ErrDigestMismatch      = errors.New("digest mismatch")
	ErrSizeInvalid         = errors.New("invalid size")
	ErrBlobCommitted       = errors.New("blob already committed")
	ErrInvalidRange        = errors.New("invalid range")
	ErrRangeNotSatisfiable = errors.New("range not satisfiable")
	ErrPartialDeletion     = errors.New("partial deletion")
)

type BlobStore interface {
	BlobReader
	BlobWriter
	BlobDeleter
}

type BlobReader interface {
	// GetBlob returns a reader for the committed blob and its stored byte-size.
	GetBlob(digest digest.Digest) (io.ReadCloser, int64, error)

	// GetBlobRange writes the inclusive byte range [first, last] of the blob to w.
	GetBlobRange(digest digest.Digest, first, last int64, w io.Writer) error
}

type BlobDeleter interface {
	// DeleteBlob removes the blob identified by digest. Returns ErrBlobUnknown if not present.
	DeleteBlob(ctx context.Context, digest digest.Digest) error

	// BatchDeleteBlobs removes the blobs identified by digests. Unknown digests are silently skipped. Invalid digests
	// are returned as errors.
	BatchDeleteBlobs(ctx context.Context, digests []digest.Digest) error
}

type BlobWriter interface {
	// PutBlob stores r under digest. Returns ErrBlobCommitted if the digest was already committed concurrently.
	PutBlob(digest digest.Digest, size int64, r io.Reader) (int64, error)

	// StatBlob returns the size of the blob without reading its content.
	StatBlob(digest digest.Digest) (int64, error)
}

// BlobVacuumer manages the lifecycle of the background vacuum process and exposes
// on-demand staging cleanup.
type BlobVacuumer interface {
	// StartVacuumProcess starts the background worker pool that processes deletions.
	StartVacuumProcess()

	// StopVacuumProcess stops the vacuum process and waits for it to exit.
	StopVacuumProcess() error

	// VacuumStagingBlobs removes staging data older than gracePeriod.
	VacuumStagingBlobs(ctx context.Context, gracePeriod time.Duration) error
}
