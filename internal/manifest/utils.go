package manifest

import (
	"encoding/json"
	"fmt"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/ocierror"
)

func convertRowToManifestInfo(row metastore.ManifestRow) ManifestInfo {
	return ManifestInfo{
		Digest:    row.Digest,
		MediaType: row.MediaType,
		Size:      row.Size,
	}
}

func deriveReferrer(manifest ocispec.Manifest, d digest.Digest, size int64) *metastore.ReferrerRow {
	if manifest.Subject == nil || manifest.Subject.Digest == "" {
		return nil
	}

	return &metastore.ReferrerRow{
		SubjectDigest: manifest.Subject.Digest,
		Digest:        d,
		MediaType:     manifest.MediaType,
		ArtifactType:  deriveReferrerArtifactType(manifest),
		Size:          size,
		Annotations:   manifest.Annotations,
	}
}

// deriveReferrerArtifactType computes the artifactType for a referrer descriptor.
func deriveReferrerArtifactType(m ocispec.Manifest) string {
	if m.ArtifactType != "" {
		return m.ArtifactType
	}

	if m.MediaType == ocispec.MediaTypeImageManifest && m.Config.MediaType != "" {
		return m.Config.MediaType
	}

	return ""
}

func parseManifestBody(contentType string, body []byte) (ocispec.Manifest, error) {
	var manifest ocispec.Manifest
	unmarshalErr := json.Unmarshal(body, &manifest)
	if unmarshalErr != nil {
		return ocispec.Manifest{}, fmt.Errorf("%w: %w", ocierror.ErrManifestInvalid, unmarshalErr)
	}

	if contentType != "" {
		manifest.MediaType = contentType
	}

	if manifest.MediaType == "" {
		return ocispec.Manifest{}, fmt.Errorf("%w: missing media type", ocierror.ErrManifestInvalid)
	}

	return manifest, nil
}
