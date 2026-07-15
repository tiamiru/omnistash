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
    name        TEXT    PRIMARY KEY,
    created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at  INTEGER NOT NULL DEFAULT (unixepoch())
);
`

func ApplyMigrations(ctx context.Context, s *SQLiteMetadataStore) error {
	_, err := s.writeDB.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("%s.ApplyMigrations: exec schema: %w", packageTag, err)
	}

	return nil
}

func CheckMigrations(ctx context.Context, s *SQLiteMetadataStore) error {
	rows, err := s.writeDB.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return fmt.Errorf("%s.CheckMigrations: query tables: %w", packageTag, err)
	}
	defer rows.Close() //nolint:errcheck

	existing := make(map[string]bool)
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return fmt.Errorf("%s.CheckMigrations: scan: %w", packageTag, err)
		}
		existing[name] = true
	}
	err = rows.Err()
	if err != nil {
		return fmt.Errorf("%s.CheckMigrations: rows: %w", packageTag, err)
	}

	var missing []string
	for _, table := range requiredTables {
		if !existing[table] {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"%s.CheckMigrations: %w: missing=[%s]",
			packageTag,
			metastore.ErrMissingTables,
			strings.Join(missing, ", "),
		)
	}

	return nil
}
