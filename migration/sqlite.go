// Package migration provides helpers for initialising the omnistash metadata store.
package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

var ErrMissingTables = errors.New("missing tables")

//nolint:gochecknoglobals // read-only list of tables that must exist after migration
var requiredTables = [5]string{
	"namespace",
	"namespace_blobs",
	"manifests",
	"manifest_tags",
	"manifest_referrers",
}

const schema = `
CREATE TABLE IF NOT EXISTS namespace (
    name        TEXT    PRIMARY KEY,
    created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at  INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS namespace_blobs (
    name    TEXT    NOT NULL,
    digest  TEXT    NOT NULL,
    size    INTEGER NOT NULL,
    lifecycle      TEXT    NOT NULL DEFAULT 'active',
    created_at     INTEGER NOT NULL DEFAULT (unixepoch()),
    deleted_at     TIMESTAMP,
    PRIMARY KEY (name, digest),
    FOREIGN KEY (name) REFERENCES namespace(name)
);

CREATE TABLE IF NOT EXISTS manifests (
    namespace   TEXT    NOT NULL,
    digest      TEXT    NOT NULL,
    media_type  TEXT    NOT NULL,
    size        INTEGER NOT NULL,
    lifecycle   TEXT    NOT NULL DEFAULT 'active',
    created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
    deleted_at  INTEGER,
    PRIMARY KEY (namespace, digest),
    FOREIGN KEY (namespace) REFERENCES namespace(name)
);

CREATE TABLE IF NOT EXISTS manifest_tags (
    namespace   TEXT    NOT NULL,
    tag         TEXT    NOT NULL,
    digest      TEXT    NOT NULL,
    created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at  INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (namespace, tag),
    FOREIGN KEY (namespace) REFERENCES namespace(name)
);

CREATE TABLE IF NOT EXISTS manifest_referrers (
    namespace       TEXT    NOT NULL,
    subject_digest  TEXT    NOT NULL,
    referrer_digest TEXT    NOT NULL,
    media_type      TEXT    NOT NULL,
    artifact_type   TEXT    NOT NULL DEFAULT '',
    size            INTEGER NOT NULL,
    annotations     TEXT,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (namespace, referrer_digest),
    FOREIGN KEY (namespace) REFERENCES namespace(name)
);
`

// ApplySQLiteMigrations creates the omnistash schema in the SQLite database at dsn.
// Call this once before starting the server for the first time.
func ApplySQLiteMigrations(ctx context.Context, dsn string) error {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("migration.ApplySQLiteMigrations: open: %w", err)
	}

	defer db.Close() //nolint:errcheck

	_, err = db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("migration.ApplySQLiteMigrations: exec schema: %w", err)
	}

	return nil
}

// CheckSQLiteSetup verifies that all required omnistash tables exist in the SQLite database at dsn.
// It returns a joined error for every missing or unreadable table.
func CheckSQLiteSetup(ctx context.Context, dsn string) error {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("migration.CheckSQLiteSetup: open: %w", err)
	}

	defer db.Close() //nolint:errcheck

	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return fmt.Errorf("migration.CheckSQLiteSetup: query tables: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	existing := make(map[string]bool)
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return fmt.Errorf("migration.CheckSQLiteSetup: scan: %w", err)
		}
		existing[name] = true
	}

	err = rows.Err()
	if err != nil {
		return fmt.Errorf("migration.CheckSQLiteSetup: rows: %w", err)
	}

	var missing []string
	for _, table := range requiredTables {
		if !existing[table] {
			missing = append(missing, table)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"migration.CheckSQLiteSetup: %w: missing=[%s]",
			ErrMissingTables,
			strings.Join(missing, ", "),
		)
	}

	return nil
}
