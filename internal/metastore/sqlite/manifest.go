package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/ocierror"
)

func (tx *sqliteTx) InsertManifest(
	ctx context.Context,
	namespace string,
	d digest.Digest,
	mediaType string,
	size int64,
) error {
	_, err := tx.tx.ExecContext(ctx, sqlInsertManifest, namespace, string(d), mediaType, size)
	if err != nil {
		return fmt.Errorf("InsertManifest: namespace=%s digest=%s: %w", namespace, d, err)
	}

	return nil
}

func (tx *sqliteTx) GetManifestByDigest(
	ctx context.Context,
	namespace string,
	d digest.Digest,
) (metastore.ManifestRow, error) {
	var row metastore.ManifestRow
	var digestStr string

	err := tx.tx.QueryRowContext(ctx, sqlGetManifestByDigest, namespace, string(d)).
		Scan(&row.Namespace, &digestStr, &row.MediaType, &row.Size)
	if errors.Is(err, sql.ErrNoRows) {
		return metastore.ManifestRow{}, fmt.Errorf(
			"%w: namespace=%s digest=%s",
			ocierror.ErrManifestUnknown,
			namespace,
			d,
		)
	}

	if err != nil {
		return metastore.ManifestRow{}, fmt.Errorf("GetManifestByDigest: namespace=%s digest=%s: %w", namespace, d, err)
	}

	row.Digest = digest.Digest(digestStr)

	return row, nil
}

func (tx *sqliteTx) DeleteManifestByDigest(ctx context.Context, namespace string, d digest.Digest) error {
	result, err := tx.tx.ExecContext(ctx, sqlDeleteManifestByDigest, namespace, string(d))
	if err != nil {
		return fmt.Errorf("DeleteManifestByDigest: namespace=%s digest=%s: %w", namespace, d, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteManifestByDigest: rows affected: namespace=%s digest=%s: %w", namespace, d, err)
	}

	if rows == 0 {
		return fmt.Errorf("%w: namespace=%s digest=%s", ocierror.ErrManifestUnknown, namespace, d)
	}

	return nil
}
