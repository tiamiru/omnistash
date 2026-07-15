package rest

import (
	"context"

	"github.com/tiamiru/omnistash/internal/namespace"
)

var _ NamespaceService = &namespace.Service{}

type NamespaceService interface {
	CreateNamespace(ctx context.Context, name string) (namespace.Namespace, error)
	DeleteNamespace(ctx context.Context, name string) (namespace.Namespace, error)
}
