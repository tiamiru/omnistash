package referrer

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

// ListResult is the output of a ListReferrers call.
type ListResult struct {
	Manifests     []ocispec.Descriptor
	FilterApplied bool
}
