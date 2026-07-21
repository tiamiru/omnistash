package referrer

import "github.com/opencontainers/go-digest"

// Descriptor is a single referrer entry in the OCI image index response.
type Descriptor struct {
	MediaType    string            `json:"mediaType"`
	Digest       digest.Digest     `json:"digest"`
	Size         int64             `json:"size"`
	ArtifactType string            `json:"artifactType,omitempty"`
	Annotations  map[string]string `json:"annotations"`
}

// ListResult is the output of a ListReferrers call.
type ListResult struct {
	Manifests     []Descriptor
	FilterApplied bool
}
