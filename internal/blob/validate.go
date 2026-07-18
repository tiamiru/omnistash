package blob

import (
	"context"
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/namespace"
	"github.com/tiamiru/omnistash/internal/ocierror"
)

func validateNamespace(ctx context.Context, meta metastore.MetadataStore, name string) error {
	err := namespace.ValidateName(name)
	if err != nil {
		return err
	}

	exists, err := meta.NamespaceExists(ctx, name)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%w: %s", ocierror.ErrNameUnknown, name)
	}

	return nil
}

func validateNamespaceDigest(ctx context.Context, meta metastore.MetadataStore, name string, d digest.Digest) error {
	err := namespace.ValidateName(name)
	if err != nil {
		return err
	}

	err = blobstore.ValidateDigest(d)
	if err != nil {
		return fmt.Errorf("%w: %w", ocierror.ErrDigestInvalid, err)
	}

	exists, err := meta.NamespaceExists(ctx, name)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%w: %s", ocierror.ErrNameUnknown, name)
	}

	return nil
}
