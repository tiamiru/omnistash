package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/tiamiru/omnistash/internal/metastore"
)

//nolint:gochecknoglobals // read-only list of required tables in sqlite
var requiredTables = [1]string{
	"namespace",
}

const schema = `
CREATE TABLE IF NOT EXISTS namespace (
    name  TEXT PRIMARY KEY
);
`

func ApplyMigrations(ctx context.Context, s *SQLiteMetadataStore) error {
	_, err := s.writeDB.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("ApplyMigrations: exec schema: %w", err)
	}

	return nil
}

func CheckMigrations(ctx context.Context, s *SQLiteMetadataStore) error {
	rows, err := s.writeDB.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return fmt.Errorf("CheckMigrations: query tables: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	existing := make(map[string]bool)
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return fmt.Errorf("CheckMigrations: scan: %w", err)
		}
		existing[name] = true
	}
	err = rows.Err()
	if err != nil {
		return fmt.Errorf("CheckMigrations: rows: %w", err)
	}

	var missing []string
	for _, table := range requiredTables {
		if !existing[table] {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("CheckMigrations: %w: [%s]", metastore.ErrMissingTables, strings.Join(missing, ", "))
	}

	return nil
}
