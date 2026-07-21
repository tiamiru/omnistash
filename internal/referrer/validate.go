package referrer

import (
	"context"
	"fmt"

	"github.com/tiamiru/omnistash/internal/ocierror"
)

func checkNamespaceExists(ctx context.Context, meta MetaOps, ns string) error {
	exists, err := meta.NamespaceExists(ctx, ns)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("%w: %s", ocierror.ErrNameUnknown, ns)
	}

	return nil
}
