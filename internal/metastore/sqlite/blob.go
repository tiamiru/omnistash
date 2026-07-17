package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/metastore"
)

func (tx *sqliteTx) GetNamespaceBlob(ctx context.Context, name string, d digest.Digest) (int64, error) {
	var size int64
	err := tx.tx.QueryRowContext(ctx, sqlGetNamespaceBlobSize, name, string(d)).Scan(&size)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%w: name=%s digest=%s", metastore.ErrBlobUnknown, name, d)
	}
	if err != nil {
		return 0, fmt.Errorf("GetNamespaceBlob: %w", err)
	}

	return size, nil
}

func (tx *sqliteTx) StatNamespaceBlob(
	ctx context.Context,
	name string,
	d digest.Digest,
) (int64, error) {
	var size int64
	err := tx.tx.QueryRowContext(ctx, sqlGetNamespaceBlobSize, name, string(d)).Scan(&size)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%w: name=%s digest=%s", metastore.ErrBlobUnknown, name, d)
	}
	if err != nil {
		return 0, fmt.Errorf("StatNamespaceBlob: %w", err)
	}

	return size, nil
}

func (tx *sqliteTx) InsertNamespaceBlob(
	ctx context.Context,
	name string,
	d digest.Digest,
	size int64,
) error {
	_, err := tx.tx.ExecContext(ctx, sqlResurrectNamespaceBlob, name, string(d))
	if err != nil {
		return fmt.Errorf("InsertNamespaceBlob: resurrect: name=%s digest=%s: %w", name, d, err)
	}

	_, err = tx.tx.ExecContext(ctx, sqlInsertNamespaceBlob, name, string(d), size)
	if err != nil {
		return fmt.Errorf("InsertNamespaceBlob: insert: name=%s digest=%s: %w", name, d, err)
	}

	return nil
}
