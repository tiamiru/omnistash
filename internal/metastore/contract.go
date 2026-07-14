package metastore

import (
	"context"
	"errors"
)

var (
	ErrMissingTables  = errors.New("missing tables")
	ErrMetastoreClose = errors.New("metastore close")
)

// NamespaceOps scopes blobs to a repository name.
//
//nolint:iface
type NamespaceOps interface {
	// CreateNamespace creates name if it does not exist. Returns (true, nil) when created,
	// (false, nil) when it already existed.
	CreateNamespace(ctx context.Context, name string) (created bool, err error)

	// DeleteNamespace removes name. Returns (true, nil) when deleted, (false, nil) when not found.
	DeleteNamespace(ctx context.Context, name string) (deleted bool, err error)
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
