package migration_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/migration"
)

func TestApplySQLiteMigrations(t *testing.T) {
	t.Parallel()

	t.Run("happy path: all tables present after applying migrations", func(t *testing.T) {
		t.Parallel()

		dsn := filepath.Join(t.TempDir(), "meta.db")

		err := migration.ApplySQLiteMigrations(context.Background(), dsn)
		require.NoError(t, err)

		err = migration.CheckSQLiteSetup(context.Background(), dsn)
		assert.NoError(t, err)
	})

	t.Run("error path: CheckSQLiteSetup returns error for every missing table", func(t *testing.T) {
		t.Parallel()

		dsn := filepath.Join(t.TempDir(), "meta.db")

		err := migration.CheckSQLiteSetup(context.Background(), dsn)
		require.Error(t, err)
		require.ErrorIs(t, err, migration.ErrMissingTables)

		for _, table := range []string{"namespace"} {
			assert.ErrorContains(t, err, table)
		}
	})
}
