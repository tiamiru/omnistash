package manifest

import (
	"encoding/json"
	"fmt"

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

func parseManifestBody(contentType string, body []byte) (partialManifest, error) {
	var partial partialManifest
	unmarshalErr := json.Unmarshal(body, &partial)
	if unmarshalErr != nil {
		return partialManifest{}, fmt.Errorf("%w: %w", ocierror.ErrManifestInvalid, unmarshalErr)
	}

	if contentType != "" {
		partial.MediaType = contentType
	}

	if partial.MediaType == "" {
		return partialManifest{}, fmt.Errorf("%w: missing media type", ocierror.ErrManifestInvalid)
	}

	return partial, nil
}
