package stub

import (
	"context"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/referrer"
)

const FixtureArtifactType = "application/vnd.example.test"

// ReferrersService is a stub implementation of rest.ReferrersService that always succeeds.
// Set FilterApplied to control optional fields in ListReferrers responses.
// Set Manifests to override the default fixture descriptor list (nil means use fixture default).
// Inspect Calls after each request to verify which methods were invoked.
type ReferrersService struct {
	FilterApplied bool
	Manifests     []referrer.Descriptor
	Calls         []string
}

func NewReferrersService() *ReferrersService {
	return &ReferrersService{}
}

func (s *ReferrersService) ListReferrers(
	_ context.Context,
	_ string,
	_ digest.Digest,
	_ string,
) (referrer.ListResult, error) {
	s.Calls = append(s.Calls, "ListReferrers")

	manifests := s.Manifests
	if manifests == nil {
		manifests = []referrer.Descriptor{
			{
				MediaType:    FixtureMediaType,
				Digest:       FixtureDigest,
				Size:         FixtureSizeBytes,
				ArtifactType: FixtureArtifactType,
			},
		}
	}

	return referrer.ListResult{
		Manifests:     manifests,
		FilterApplied: s.FilterApplied,
	}, nil
}
