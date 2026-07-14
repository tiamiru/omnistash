package sqlite

import (
	"context"
	"fmt"
)

func (s *SQLiteMetadataStore) NamespaceExists(ctx context.Context, name string) (bool, error) {
	var found bool
	err := s.readDB.QueryRowContext(ctx, sqlNamespaceExists, name).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("%s.NamespaceExists: %w", storeTag, err)
	}

	return found, nil
}

func (tx *sqliteTx) CreateNamespace(ctx context.Context, name string) (bool, error) {
	result, err := tx.tx.ExecContext(ctx, sqlInsertNamespace, name)
	if err != nil {
		return false, fmt.Errorf("CreateNamespace: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("CreateNamespace: rows affected: %w", err)
	}

	return rows > 0, nil
}

func (tx *sqliteTx) DeleteNamespace(ctx context.Context, name string) (bool, error) {
	result, err := tx.tx.ExecContext(ctx, sqlDeleteNamespace, name)
	if err != nil {
		return false, fmt.Errorf("DeleteNamespace: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("DeleteNamespace: rows affected: %w", err)
	}

	return rows > 0, nil
}
