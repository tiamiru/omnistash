package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/metastore"
)

func (tx *sqliteTx) UpsertReferrer(ctx context.Context, namespace string, row metastore.ReferrerRow) error {
	annotations := row.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	b, err := json.Marshal(annotations)
	if err != nil {
		return fmt.Errorf("UpsertReferrer: marshal annotations: %w", err)
	}

	s := string(b)
	annotationsJSON := &s

	_, err = tx.tx.ExecContext(
		ctx,
		sqlUpsertReferrer,
		namespace,
		string(row.SubjectDigest),
		string(row.Digest),
		row.MediaType,
		row.ArtifactType,
		row.Size,
		annotationsJSON,
	)
	if err != nil {
		return fmt.Errorf(
			"UpsertReferrer: namespace=%s subject=%s referrer=%s: %w",
			namespace, row.SubjectDigest, row.Digest, err,
		)
	}

	return nil
}

func (tx *sqliteTx) ListReferrers(
	ctx context.Context,
	namespace string,
	subjectDigest digest.Digest,
) ([]metastore.ReferrerRow, error) {
	rows, err := tx.tx.QueryContext(ctx, sqlListReferrers, namespace, string(subjectDigest))
	if err != nil {
		return nil, fmt.Errorf("ListReferrers: namespace=%s subject=%s: %w", namespace, subjectDigest, err)
	}

	defer rows.Close() //nolint:errcheck

	var result []metastore.ReferrerRow

	for rows.Next() {
		var (
			row             metastore.ReferrerRow
			referrerDigest  string
			annotationsJSON sql.NullString
		)

		scanErr := rows.Scan(&referrerDigest, &row.MediaType, &row.ArtifactType, &row.Size, &annotationsJSON)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"ListReferrers: scan: namespace=%s subject=%s: %w",
				namespace,
				subjectDigest,
				scanErr,
			)
		}

		row.SubjectDigest = subjectDigest
		row.Digest = digest.Digest(referrerDigest)

		if annotationsJSON.Valid && annotationsJSON.String != "" {
			unmarshalErr := json.Unmarshal([]byte(annotationsJSON.String), &row.Annotations)
			if unmarshalErr != nil {
				return nil, fmt.Errorf(
					"ListReferrers: unmarshal annotations: namespace=%s subject=%s: %w",
					namespace, subjectDigest, unmarshalErr,
				)
			}
		} else {
			row.Annotations = map[string]string{}
		}

		result = append(result, row)
	}

	rowsErr := rows.Err()
	if rowsErr != nil {
		return nil, fmt.Errorf("ListReferrers: rows: namespace=%s subject=%s: %w", namespace, subjectDigest, rowsErr)
	}

	if result == nil {
		result = []metastore.ReferrerRow{}
	}

	return result, nil
}

func (tx *sqliteTx) DeleteReferrer(ctx context.Context, namespace string, referrerDigest digest.Digest) error {
	_, err := tx.tx.ExecContext(ctx, sqlDeleteReferrer, namespace, string(referrerDigest))
	if err != nil {
		return fmt.Errorf("DeleteReferrer: namespace=%s referrer=%s: %w", namespace, referrerDigest, err)
	}

	return nil
}
