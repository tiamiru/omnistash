package rest

import (
	"encoding/json"
	"net/http"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/logtag"
	"github.com/tiamiru/omnistash/internal/ocierror"
	"github.com/tiamiru/omnistash/internal/referrer"
)

const (
	mediaTypeOCIImageIndex     = "application/vnd.oci.image.index.v1+json"
	ociImageIndexSchemaVersion = 2
)

type ociImageIndex struct {
	SchemaVersion int                   `json:"schemaVersion"`
	MediaType     string                `json:"mediaType"`
	Manifests     []referrer.Descriptor `json:"manifests"`
}

// handleGetReferrers implements GET /v2/{name}/referrers/{digest}.
func (h *RegistryHandler) handleGetReferrers(w http.ResponseWriter, r *http.Request) {
	ns := repoName(r)
	rawDigest := r.PathValue("digest")
	artifactType := r.URL.Query().Get("artifactType")

	d, err := digest.Parse(rawDigest)
	if err != nil {
		h.registryErrToHTTP(w, "handleGetReferrers", ocierror.ErrDigestInvalid)

		return
	}

	result, err := h.referrers.ListReferrers(r.Context(), ns, d, artifactType)
	if err != nil {
		h.registryErrToHTTP(w, "handleGetReferrers", err)

		return
	}

	manifests := result.Manifests
	if manifests == nil {
		manifests = []referrer.Descriptor{}
	}

	body, err := json.Marshal(ociImageIndex{
		SchemaVersion: ociImageIndexSchemaVersion,
		MediaType:     mediaTypeOCIImageIndex,
		Manifests:     manifests,
	})
	if err != nil {
		h.logger.Warn("handleGetReferrers: marshal response", logtag.Err(err))
		h.writeOCIError(
			w,
			"handleGetReferrers",
			http.StatusInternalServerError,
			ociCodeInternalError,
			"internal server error",
		)

		return
	}

	w.Header().Set("Content-Type", mediaTypeOCIImageIndex)

	if result.FilterApplied {
		w.Header().Set(headerOCIFiltersApplied, "artifactType")
	}

	w.WriteHeader(http.StatusOK)

	_, writeErr := w.Write(body)
	if writeErr != nil {
		h.logger.Warn("handleGetReferrers: write response", logtag.Err(writeErr))
	}
}
