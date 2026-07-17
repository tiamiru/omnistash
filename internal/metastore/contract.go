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

// TxOps is the full set of operations available inside an Atomic transaction.
type TxOps interface {
	NamespaceOps
	BlobOps
}

// MetadataStore is the top-level store handle.
type MetadataStore interface {
	// Atomic runs fn in a serializable transaction, committing on success and rolling back on error or panic.
	// All writes and check-then-write sequences must go through Atomic.
	Atomic(ctx context.Context, fn func(ctx context.Context, tx TxOps) error) error

	// NamespaceExists reports whether name has been created.
	NamespaceExists(ctx context.Context, namespace string) (bool, error)
}
