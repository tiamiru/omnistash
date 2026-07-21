package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/tiamiru/omnistash/internal/logtag"
	"github.com/tiamiru/omnistash/internal/ocierror"
	"github.com/tiamiru/omnistash/internal/referrer"
)

const ociImageIndexSchemaVersion = 2

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
	if errors.Is(err, ocierror.ErrNameUnknown) {
		result = referrer.ListResult{Manifests: []ocispec.Descriptor{}}
	} else if err != nil {
		h.registryErrToHTTP(w, "handleGetReferrers", err)

		return
	}

	manifests := result.Manifests
	if manifests == nil {
		manifests = []ocispec.Descriptor{}
	}

	body, err := json.Marshal(ocispec.Index{
		Versioned: specs.Versioned{SchemaVersion: ociImageIndexSchemaVersion},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: manifests,
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

	w.Header().Set("Content-Type", ocispec.MediaTypeImageIndex)

	if result.FilterApplied {
		w.Header().Set(headerOCIFiltersApplied, "artifactType")
	}

	w.WriteHeader(http.StatusOK)

	_, writeErr := w.Write(body)
	if writeErr != nil {
		h.logger.Warn("handleGetReferrers: write response", logtag.Err(writeErr))
	}
}
