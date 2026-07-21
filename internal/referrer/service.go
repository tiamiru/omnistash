package referrer

import (
	"context"
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/namespace"
)

// MetaOps is the subset of metastore.MetadataStore used by this service.
type MetaOps interface {
	NamespaceExists(ctx context.Context, ns string) (bool, error)
	Atomic(ctx context.Context, fn func(ctx context.Context, tx metastore.TxOps) error) error
}

type Service struct {
	meta MetaOps
}

func NewService(meta MetaOps) *Service {
	return &Service{meta: meta}
}

func (s *Service) ListReferrers(
	ctx context.Context,
	ns string,
	subject digest.Digest,
	artifactType string,
) (ListResult, error) {
	err := namespace.ValidateName(ns)
	if err != nil {
		return ListResult{}, fmt.Errorf("ListReferrers: %w", err)
	}

	err = checkNamespaceExists(ctx, s.meta, ns)
	if err != nil {
		return ListResult{}, fmt.Errorf("ListReferrers: %w", err)
	}

	var rows []metastore.ReferrerRow

	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		var listErr error
		rows, listErr = tx.ListReferrers(ctx, ns, subject)

		return listErr
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("ListReferrers: %w", err)
	}

	descriptors := make([]Descriptor, 0, len(rows))

	for _, row := range rows {
		descriptors = append(descriptors, Descriptor{
			MediaType:    row.MediaType,
			Digest:       row.Digest,
			Size:         row.Size,
			ArtifactType: row.ArtifactType,
			Annotations:  row.Annotations,
		})
	}

	if artifactType != "" {
		filtered := descriptors[:0]

		for _, d := range descriptors {
			if d.ArtifactType == artifactType {
				filtered = append(filtered, d)
			}
		}

		return ListResult{Manifests: filtered, FilterApplied: true}, nil
	}

	return ListResult{Manifests: descriptors}, nil
}
