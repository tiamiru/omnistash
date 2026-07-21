package metastore

import (
	"context"
	"errors"
	"time"

	"github.com/opencontainers/go-digest"
)

var (
	ErrMetastoreClose = errors.New("metastore close")
	ErrNameExists     = errors.New("namespace already exists")
	ErrNameUnknown    = errors.New("namespace unknown")
	ErrBlobUnknown    = errors.New("blob unknown")
)

// NamespaceRow is the metastore representation of a namespace row.
type NamespaceRow struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NamespaceOps scopes namespace operations to a repository name.
type NamespaceOps interface {
	// CreateNamespace creates name if it does not exist.
	//
	// If the namespace already exists, returns ErrNameExists.
	CreateNamespace(ctx context.Context, namespace string) (NamespaceRow, error)

	// DeleteNamespace removes name and returns the deleted row.
	//
	// If the namespace does not exist, returns ErrNameUnknown.
	DeleteNamespace(ctx context.Context, namespace string) (NamespaceRow, error)
}

// BlobOps scopes blob operations to a repository name.
type BlobOps interface {
	// InsertNamespaceBlob registers d under namespace with the given size.
	// Reactivates d if it was previously marked for deletion.
	InsertNamespaceBlob(ctx context.Context, namespace string, d digest.Digest, size int64) error

	// GetNamespaceBlob returns the registered size of d scoped to name.
	// Returns ErrBlobUnknown if d is absent or pending deletion.
	GetNamespaceBlob(ctx context.Context, namespace string, d digest.Digest) (size int64, err error)

	// StatNamespaceBlob returns the size of d scoped to name and the blobs partition.
	// Returns ErrBlobUnknown if d is absent or is pending deletion.
	StatNamespaceBlob(
		ctx context.Context,
		namespace string,
		d digest.Digest,
	) (size int64, err error)
}

// ManifestRow is the metastore representation of a manifest row.
type ManifestRow struct {
	Namespace string
	Digest    digest.Digest
	MediaType string
	Size      int64
}

// ManifestOps scopes manifest and tag operations to a repository namespace.
type ManifestOps interface {
	// InsertManifest records manifest metadata. Idempotent on (namespace, digest).
	InsertManifest(ctx context.Context, namespace string, d digest.Digest, mediaType string, size int64) error

	// GetManifestByDigest returns metadata for one manifest.
	// Returns ocierror.ErrManifestUnknown if absent or soft-deleted.
	GetManifestByDigest(ctx context.Context, namespace string, d digest.Digest) (ManifestRow, error)

	// DeleteManifestByDigest soft-deletes the manifest row.
	// Returns ocierror.ErrManifestUnknown if already absent.
	DeleteManifestByDigest(ctx context.Context, namespace string, d digest.Digest) error
}

// ReferrerRow is the stored descriptor for one referrer entry.
type ReferrerRow struct {
	SubjectDigest digest.Digest
	Digest        digest.Digest
	MediaType     string
	ArtifactType  string
	Size          int64
	Annotations   map[string]string
}

// ReferrerOps manages the referrer index inside a transaction.
type ReferrerOps interface {
	// UpsertReferrer records that row.Digest refers to row.SubjectDigest.
	// This operation is idempotent as the same referrer digest overwrites the prior entry.
	UpsertReferrer(ctx context.Context, namespace string, row ReferrerRow) error

	// ListReferrers returns all referrer entries pointing to subjectDigest.
	// Returns an empty slice when none exist.
	ListReferrers(ctx context.Context, namespace string, subjectDigest digest.Digest) ([]ReferrerRow, error)

	// DeleteReferrer removes the referrer entry for referrerDigest.
	// No-op when no entry exists.
	DeleteReferrer(ctx context.Context, namespace string, referrerDigest digest.Digest) error
}

// TxOps is the full set of operations available inside an Atomic transaction.
type TxOps interface {
	NamespaceOps
	BlobOps
	ManifestOps
	ReferrerOps
}

// MetadataStore is the top-level store handle.
type MetadataStore interface {
	// Atomic runs fn in a serializable transaction, committing on success and rolling back on error or panic.
	// All writes and check-then-write sequences must go through Atomic.
	Atomic(ctx context.Context, fn func(ctx context.Context, tx TxOps) error) error

	// NamespaceExists reports whether name has been created.
	NamespaceExists(ctx context.Context, namespace string) (bool, error)
}
