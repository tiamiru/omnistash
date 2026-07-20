package rest

import (
	"context"
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blob"
	"github.com/tiamiru/omnistash/internal/manifest"
	"github.com/tiamiru/omnistash/internal/namespace"
)

const headerOCISubject = "OCI-Subject"

var (
	_ NamespaceService = &namespace.Service{}
	_ BlobService      = &blob.Service{}
	_ ManifestService  = &manifest.Service{}
)

type NamespaceService interface {
	CreateNamespace(ctx context.Context, name string) (namespace.Namespace, error)
	DeleteNamespace(ctx context.Context, name string) (namespace.Namespace, error)
}

type BlobService interface {
	StatBlob(ctx context.Context, name string, d digest.Digest) (size int64, err error)
	GetBlob(ctx context.Context, name string, d digest.Digest) (rc io.ReadCloser, size int64, err error)
	GetBlobRange(
		ctx context.Context,
		name string,
		d digest.Digest,
		first, last int64,
	) (rc io.ReadCloser, totalSize int64, err error)
	InitiateUpload(ctx context.Context, name string) (uploadID string, err error)
	MonolithicUpload(ctx context.Context, name string, d digest.Digest, size int64, r io.Reader) error
	AppendChunk(ctx context.Context, name, uploadID string, offset int64, r io.Reader) (newOffset int64, err error)
	CommitUpload(ctx context.Context, name, uploadID string, d digest.Digest, finalChunk io.Reader) error
	GetUploadStatus(ctx context.Context, name, uploadID string) (offset int64, err error)
	CancelUpload(ctx context.Context, name, uploadID string) error
	MountBlob(
		ctx context.Context,
		sourceName, targetName string,
		d digest.Digest,
	) error
}

type ManifestService interface {
	PutManifest(
		ctx context.Context,
		namespace, reference, contentType string,
		body []byte,
	) (manifest.PutResult, error)
	GetManifest(ctx context.Context, namespace, reference string) (manifest.ManifestInfo, io.ReadCloser, error)
	HeadManifest(ctx context.Context, namespace, reference string) (manifest.ManifestInfo, error)
	DeleteManifest(ctx context.Context, namespace, reference string) error
}
