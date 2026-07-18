package namespace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tiamiru/omnistash/internal/metastore"
)

// Namespace is the domain representation of a namespace.
type Namespace struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Service struct {
	meta metastore.MetadataStore
}

func NewService(meta metastore.MetadataStore) *Service {
	return &Service{meta: meta}
}

func (s *Service) CreateNamespace(ctx context.Context, name string) (Namespace, error) {
	err := ValidateName(name)
	if err != nil {
		return Namespace{}, fmt.Errorf("CreateNamespace: name=%s: %w", name, err)
	}

	var ns Namespace
	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		row, createErr := tx.CreateNamespace(ctx, name)
		if createErr != nil {
			if errors.Is(createErr, metastore.ErrNameExists) {
				return fmt.Errorf("%w: name=%s", ErrNameExists, name)
			}

			return createErr
		}

		ns = Namespace{Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}

		return nil
	})

	if err != nil {
		return Namespace{}, fmt.Errorf("CreateNamespace: name=%s: %w", name, err)
	}

	return ns, nil
}

func (s *Service) DeleteNamespace(ctx context.Context, name string) (Namespace, error) {
	err := ValidateName(name)
	if err != nil {
		return Namespace{}, fmt.Errorf("DeleteNamespace: name=%s: %w", name, err)
	}

	var ns Namespace
	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		row, deleteErr := tx.DeleteNamespace(ctx, name)
		if deleteErr != nil {
			if errors.Is(deleteErr, metastore.ErrNameUnknown) {
				return fmt.Errorf("%w: name=%s", ErrNameUnknown, name)
			}

			return deleteErr
		}

		ns = Namespace{Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}

		return nil
	})
	if err != nil {
		return Namespace{}, fmt.Errorf("DeleteNamespace: name=%s: %w", name, err)
	}

	return ns, nil
}
