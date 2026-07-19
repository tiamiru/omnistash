package manifest

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/tiamiru/omnistash/internal/ocierror"
)

type Service struct {
	logger *slog.Logger
}

func NewService(logger *slog.Logger) *Service {
	return &Service{logger: logger}
}

func (s *Service) PutManifest(
	_ context.Context,
	_, _, _ string,
	_ []byte,
) (PutResult, error) {
	return PutResult{}, fmt.Errorf("PutManifest: %w", ocierror.ErrUnsupported)
}

func (s *Service) PutManifestWithTags(
	_ context.Context,
	_, _ string,
	_ []string,
	_ string,
	_ []byte,
) (PutResult, error) {
	return PutResult{}, fmt.Errorf("PutManifestWithTags: %w", ocierror.ErrUnsupported)
}

func (s *Service) GetManifest(
	_ context.Context,
	_, _ string,
) (ManifestInfo, io.ReadCloser, error) {
	return ManifestInfo{}, nil, fmt.Errorf("GetManifest: %w", ocierror.ErrUnsupported)
}

func (s *Service) HeadManifest(
	_ context.Context,
	_, _ string,
) (ManifestInfo, error) {
	return ManifestInfo{}, fmt.Errorf("HeadManifest: %w", ocierror.ErrUnsupported)
}

func (s *Service) DeleteManifest(
	_ context.Context,
	_, _ string,
) error {
	return fmt.Errorf("DeleteManifest: %w", ocierror.ErrUnsupported)
}
