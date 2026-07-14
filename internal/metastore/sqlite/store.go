package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/tiamiru/omnistash/internal/metastore"
)

type SQLiteMetadataStore struct {
	writeDB *sql.DB
	readDB  *sql.DB
}

type sqliteTx struct {
	tx *sql.Tx
}

func NewSQLiteMetadataStore(ctx context.Context, dsn string) (*SQLiteMetadataStore, error) {
	writeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite.New: open write db: %w", err)
	}
	writeDB.SetMaxOpenConns(1)

	_, err = writeDB.ExecContext(ctx, "PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("sqlite.New: enable WAL: %w", err)
	}

	_, err = writeDB.ExecContext(ctx, "PRAGMA foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("sqlite.New: enable foreign keys: %w", err)
	}

	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite.New: open read db: %w", err)
	}

	return &SQLiteMetadataStore{writeDB: writeDB, readDB: readDB}, nil
}

func (s *SQLiteMetadataStore) Close() error {
	writeErr := s.writeDB.Close()
	readErr := s.readDB.Close()

	if writeErr != nil || readErr != nil {
		return fmt.Errorf("%w: write=%w read=%w", metastore.ErrMetastoreClose, writeErr, readErr)
	}

	return nil
}

func (s *SQLiteMetadataStore) Atomic(
	ctx context.Context,
	fn func(ctx context.Context, tx metastore.TxOps) error,
) (err error) {
	sqlTx, err := s.writeDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("sqlite.Atomic: begin: %w", err)
	}

	committed := false
	defer func() {
		if p := recover(); p != nil {
			rollbackErr := sqlTx.Rollback()
			if rollbackErr != nil {
				panic(fmt.Errorf("sqlite.Atomic: rollback after panic: %w; original: %v", rollbackErr, p))
			}
			panic(p)
		}
		if !committed {
			rollbackErr := sqlTx.Rollback()
			if rollbackErr != nil {
				err = errors.Join(err, fmt.Errorf("sqlite.Atomic: rollback: %w", rollbackErr))
			}
		}
	}()

	tx := &sqliteTx{tx: sqlTx}
	err = fn(ctx, tx)
	if err != nil {
		return err
	}

	err = sqlTx.Commit()
	if err != nil {
		return fmt.Errorf("sqlite.Atomic: commit: %w", err)
	}

	committed = true

	return nil
}
