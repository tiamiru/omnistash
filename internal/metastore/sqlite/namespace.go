package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/tiamiru/omnistash/internal/metastore"
)

func (s *SQLiteMetadataStore) NamespaceExists(ctx context.Context, name string) (bool, error) {
	var found bool
	err := s.readDB.QueryRowContext(ctx, sqlNamespaceExists, name).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("NamespaceExists: name=%s: %w", name, err)
	}

	return found, nil
}

func (tx *sqliteTx) CreateNamespace(ctx context.Context, name string) (metastore.NamespaceRow, error) {
	result, err := tx.tx.ExecContext(ctx, sqlInsertNamespace, name)
	if err != nil {
		return metastore.NamespaceRow{}, fmt.Errorf("CreateNamespace: name=%s: %w", name, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return metastore.NamespaceRow{}, fmt.Errorf("CreateNamespace: name=%s: rows affected: %w", name, err)
	}

	if rows == 0 {
		return metastore.NamespaceRow{}, fmt.Errorf("CreateNamespace: %w: name=%s", metastore.ErrNameExists, name)
	}

	var ns metastore.NamespaceRow
	var createdAt, updatedAt int64
	err = tx.tx.QueryRowContext(ctx, sqlSelectNamespace, name).Scan(&ns.Name, &createdAt, &updatedAt)
	if err != nil {
		return metastore.NamespaceRow{}, fmt.Errorf("CreateNamespace: name=%s: select: %w", name, err)
	}

	ns.CreatedAt = metastore.UnixToTime(createdAt)
	ns.UpdatedAt = metastore.UnixToTime(updatedAt)

	return ns, nil
}

func (tx *sqliteTx) DeleteNamespace(ctx context.Context, name string) (metastore.NamespaceRow, error) {
	var ns metastore.NamespaceRow
	var createdAt, updatedAt int64
	err := tx.tx.QueryRowContext(ctx, sqlSelectNamespace, name).Scan(&ns.Name, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return metastore.NamespaceRow{}, fmt.Errorf("%w: name=%s", metastore.ErrNameUnknown, name)
		}

		return metastore.NamespaceRow{}, fmt.Errorf("DeleteNamespace: name=%s: query: %w", name, err)
	}

	ns.CreatedAt = metastore.UnixToTime(createdAt)
	ns.UpdatedAt = metastore.UnixToTime(updatedAt)

	_, err = tx.tx.ExecContext(ctx, sqlDeleteNamespace, name)
	if err != nil {
		return metastore.NamespaceRow{}, fmt.Errorf("DeleteNamespace: name=%s: %w", name, err)
	}

	return ns, nil
}
