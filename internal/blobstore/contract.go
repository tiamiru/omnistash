package blobstore

import (
	"errors"
	"io"

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
)

type BlobStore interface {
	BlobWriter
	BlobReader
}

type BlobReader interface {
	// GetBlob returns a reader for the committed blob and its stored byte-size.
	GetBlob(digest digest.Digest) (io.ReadCloser, int64, error)

	// GetBlobRange writes the inclusive byte range [first, last] of the blob to w.
	GetBlobRange(digest digest.Digest, first, last int64, w io.Writer) error
}

type BlobWriter interface {
	// PutBlob stores r under digest. Returns ErrBlobCommitted if the digest was already committed concurrently.
	PutBlob(digest digest.Digest, size int64, r io.Reader) (int64, error)

	// StatBlob returns the size of the blob without reading its content.
	StatBlob(digest digest.Digest) (int64, error)
}
