package metastore

import (
	"context"
	"errors"
	"time"
)

var (
	ErrMissingTables  = errors.New("missing tables")
	ErrMetastoreClose = errors.New("metastore close")
	ErrNameExists     = errors.New("namespace already exists")
	ErrNameUnknown    = errors.New("namespace unknown")
)

// NamespaceRow is the metastore representation of a namespace row.
type NamespaceRow struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NamespaceOps scopes blobs to a repository name.
//
//nolint:iface
type NamespaceOps interface {
	// CreateNamespace creates name if it does not exist.
	//
	// If the namespace already exists, returns ErrNameExists.
	CreateNamespace(ctx context.Context, name string) (NamespaceRow, error)

	// DeleteNamespace removes name and returns the deleted row.
	//
	// If the namespace does not exist, returns ErrNameUnknown.
	DeleteNamespace(ctx context.Context, name string) (NamespaceRow, error)
}

// TxOps is the full set of operations available inside an Atomic transaction.
//
//nolint:iface
type TxOps interface {
	NamespaceOps
}

// MetadataStore is the top-level store handle.
type MetadataStore interface {
	// Atomic runs fn in a serializable transaction, committing on success and rolling back on error or panic.
	// All writes and check-then-write sequences must go through Atomic.
	Atomic(ctx context.Context, fn func(ctx context.Context, tx TxOps) error) error

	// NamespaceExists reports whether name has been created.
	NamespaceExists(ctx context.Context, name string) (bool, error)
}
