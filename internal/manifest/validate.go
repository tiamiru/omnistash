package manifest

import (
	"context"
	"fmt"
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/ocierror"
)

func checkNamespaceExists(ctx context.Context, meta MetaOps, name string) error {
	exists, err := meta.NamespaceExists(ctx, name)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%w: %s", ocierror.ErrNameUnknown, name)
	}

	return nil
}

// validateDigestReference parses ref as a digest. Only digest references are
// accepted; tag references are not supported by this service.
func validateDigestReference(ref string) (digest.Digest, error) {
	if !strings.Contains(ref, ":") {
		return "", fmt.Errorf("%w: tag references are not supported", ocierror.ErrUnsupported)
	}

	d, err := digest.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ocierror.ErrDigestInvalid, ref)
	}

	return d, nil
}
