package stub

import (
	"bytes"
	"context"
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/manifest"
)

const (
	FixtureMediaType               = "application/vnd.oci.image.manifest.v1+json"
	FixtureSizeBytes               = int64(512)
	FixtureDigest    digest.Digest = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	FixtureLocation                = "/v2/myrepo/manifests/" + string(FixtureDigest)
)

func FixtureBody() []byte {
	return []byte(`{"schemaVersion":2}`)
}

// ManifestService is a stub implementation of rest.ManifestService that always succeeds.
// Set Subject to control optional fields in PutManifest responses.
// Inspect Calls after each request to verify which methods were invoked.
type ManifestService struct {
	Subject *digest.Digest
	Calls   []string
}

func NewManifestService() *ManifestService {
	return &ManifestService{}
}

func (s *ManifestService) GetManifest(_ context.Context, _, _ string) (manifest.ManifestInfo, io.ReadCloser, error) {
	s.Calls = append(s.Calls, "GetManifest")

	return manifest.ManifestInfo{
		Digest:    FixtureDigest,
		MediaType: FixtureMediaType,
		Size:      FixtureSizeBytes,
	}, io.NopCloser(bytes.NewReader(FixtureBody())), nil
}

func (s *ManifestService) HeadManifest(_ context.Context, _, _ string) (manifest.ManifestInfo, error) {
	s.Calls = append(s.Calls, "HeadManifest")

	return manifest.ManifestInfo{
		Digest:    FixtureDigest,
		MediaType: FixtureMediaType,
		Size:      FixtureSizeBytes,
	}, nil
}

func (s *ManifestService) PutManifest(_ context.Context, _, _, _ string, _ []byte) (manifest.PutResult, error) {
	s.Calls = append(s.Calls, "PutManifest")

	return manifest.PutResult{
		Digest:   FixtureDigest,
		Location: FixtureLocation,
		Subject:  s.Subject,
	}, nil
}

func (s *ManifestService) DeleteManifest(_ context.Context, _, _ string) error {
	s.Calls = append(s.Calls, "DeleteManifest")

	return nil
}
