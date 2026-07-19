package blob

import (
	"context"
	"errors"
	"fmt"

	"github.com/opencontainers/go-digest"

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

func ValidateDigest(d digest.Digest) error {
	valErr := d.Validate()
	if valErr != nil {
		if errors.Is(valErr, digest.ErrDigestUnsupported) {
			return fmt.Errorf("%w: %s", ocierror.ErrUnsupported, d)
		}

		return fmt.Errorf("%w: %s", ocierror.ErrDigestInvalid, d)
	}

	return nil
}

func validateNamespaceDigest(ctx context.Context, meta metastore.MetadataStore, name string, d digest.Digest) error {
	err := namespace.ValidateName(name)
	if err != nil {
		return err
	}
	err = ValidateDigest(d)
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
