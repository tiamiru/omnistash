package rest

import (
	"io"
	"net/http"
	"strconv"

	"github.com/tiamiru/omnistash/internal/logtag"
)

const maxManifestSize = 4 * 1024 * 1024 // 4 MiB

// handleGetManifest implements GET /v2/{name}/manifests/{reference}.
func (h *RegistryHandler) handleGetManifest(w http.ResponseWriter, r *http.Request) {
	ns := repoName(r)
	reference := r.PathValue("reference")

	info, body, err := h.manifest.GetManifest(r.Context(), ns, reference)
	if err != nil {
		h.registryErrToHTTP(w, "handleGetManifest", err)

		return
	}

	defer func() {
		closeErr := body.Close()
		if closeErr != nil {
			h.logger.Warn("handleGetManifest: close body", logtag.Err(closeErr))
		}
	}()

	w.Header().Set("Content-Type", info.MediaType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.Header().Set("Docker-Content-Digest", info.Digest.String())
	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, body)
	if err != nil {
		h.logger.Warn("handleGetManifest: write response", logtag.Err(err))
	}
}

// handleHeadManifest implements HEAD /v2/{name}/manifests/{reference}.
func (h *RegistryHandler) handleHeadManifest(w http.ResponseWriter, r *http.Request) {
	ns := repoName(r)
	reference := r.PathValue("reference")

	info, err := h.manifest.HeadManifest(r.Context(), ns, reference)
	if err != nil {
		h.registryErrToHTTPNoBody(w, "handleHeadManifest", err)

		return
	}

	w.Header().Set("Content-Type", info.MediaType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.Header().Set("Docker-Content-Digest", info.Digest.String())
	w.WriteHeader(http.StatusOK)
}

// handlePutManifest implements PUT /v2/{name}/manifests/{reference}.
func (h *RegistryHandler) handlePutManifest(w http.ResponseWriter, r *http.Request) {
	ns := repoName(r)
	reference := r.PathValue("reference")
	contentType := r.Header.Get("Content-Type")

	body, err := io.ReadAll(io.LimitReader(r.Body, maxManifestSize+1))
	if err != nil {
		h.registryErrToHTTP(w, "handlePutManifest", err)

		return
	}

	if int64(len(body)) > maxManifestSize {
		w.WriteHeader(http.StatusRequestEntityTooLarge)

		return
	}

	result, err := h.manifest.PutManifest(r.Context(), ns, reference, contentType, body)
	if err != nil {
		h.registryErrToHTTP(w, "handlePutManifest", err)

		return
	}

	w.Header().Set("Location", result.Location)
	w.Header().Set("Docker-Content-Digest", result.Digest.String())

	if result.Subject != nil {
		w.Header().Set(headerOCISubject, result.Subject.String())
	}

	w.WriteHeader(http.StatusCreated)
}

// handleDeleteManifest implements DELETE /v2/{name}/manifests/{reference}.
func (h *RegistryHandler) handleDeleteManifest(w http.ResponseWriter, r *http.Request) {
	ns := repoName(r)
	reference := r.PathValue("reference")

	err := h.manifest.DeleteManifest(r.Context(), ns, reference)
	if err != nil {
		h.registryErrToHTTP(w, "handleDeleteManifest", err)

		return
	}

	w.WriteHeader(http.StatusAccepted)
}
