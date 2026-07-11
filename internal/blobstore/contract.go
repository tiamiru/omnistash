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
	ErrBlobUnknown    = errors.New("blob unknown")
	ErrInvalidDigest  = errors.New("invalid digest")
	ErrDigestMismatch = errors.New("digest mismatch")
	ErrSizeInvalid    = errors.New("invalid size")
	ErrBlobCommitted  = errors.New("blob already committed")
)

type BlobStore interface {
	BlobWriter
}

type BlobWriter interface {
	// PutBlob stores r under digest. Returns ErrBlobCommitted if the digest was already committed concurrently.
	PutBlob(digest digest.Digest, size int64, r io.Reader) (int64, error)

	// StatBlob returns the size of the blob without reading its content.
	StatBlob(digest digest.Digest) (int64, error)
}
