package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/tiamiru/omnistash/internal/metastore"
)

const (
	storeTag   = "SQLiteMetadataStore"
	packageTag = "sqlite"
)

var (
	_ metastore.MetadataStore = &SQLiteMetadataStore{}
	_ metastore.TxOps         = &sqliteTx{}
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
		return nil, fmt.Errorf("%s.New: open write db: %w", packageTag, err)
	}
	writeDB.SetMaxOpenConns(1)

	_, err = writeDB.ExecContext(ctx, "PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("%s.New: enable WAL: %w", packageTag, err)
	}

	_, err = writeDB.ExecContext(ctx, "PRAGMA foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("%s.New: enable foreign keys: %w", packageTag, err)
	}

	readDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s.New: open read db: %w", packageTag, err)
	}

	return &SQLiteMetadataStore{writeDB: writeDB, readDB: readDB}, nil
}

func (s *SQLiteMetadataStore) Close() error {
	writeErr := s.writeDB.Close()
	readErr := s.readDB.Close()

	if writeErr != nil || readErr != nil {
		return fmt.Errorf("%w: %w", metastore.ErrMetastoreClose, errors.Join(writeErr, readErr))
	}

	return nil
}

func (s *SQLiteMetadataStore) Atomic(
	ctx context.Context,
	fn func(ctx context.Context, tx metastore.TxOps) error,
) (err error) {
	sqlTx, err := s.writeDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("%s.Atomic: begin: %w", storeTag, err)
	}

	committed := false
	defer func() {
		p := recover()
		if p != nil {
			rollbackErr := sqlTx.Rollback()
			if rollbackErr != nil {
				panic(fmt.Errorf("%s.Atomic: rollback after panic: original=%v: %w", storeTag, p, rollbackErr))
			}
			panic(p)
		}
		if !committed {
			rollbackErr := sqlTx.Rollback()
			if rollbackErr != nil {
				err = errors.Join(err, fmt.Errorf("%s.Atomic: rollback: %w", storeTag, rollbackErr))
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
		return fmt.Errorf("%s.Atomic: commit: %w", storeTag, err)
	}

	committed = true

	return nil
}
