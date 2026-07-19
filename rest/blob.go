package rest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/logtag"
	"github.com/tiamiru/omnistash/internal/ocierror"
)

// handleHeadBlob implements HEAD /v2/{name}/blobs/{digest}.
func (h *RegistryHandler) handleHeadBlob(w http.ResponseWriter, r *http.Request) {
	name := repoName(r)
	digestStr := r.PathValue("digest")
	d := digest.Digest(digestStr)

	size, err := h.blob.StatBlob(r.Context(), name, d)
	if err != nil {
		h.registryErrToHTTPNoBody(w, "handleHeadBlob", err)

		return
	}

	w.Header().Set("Docker-Content-Digest", digestStr)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusOK)
}

// handleGetBlob implements GET /v2/{name}/blobs/{digest}.
func (h *RegistryHandler) handleGetBlob(w http.ResponseWriter, r *http.Request) {
	name := repoName(r)
	digestStr := r.PathValue("digest")
	d := digest.Digest(digestStr)

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		h.serveFullBlob(w, r, name, d, digestStr)

		return
	}

	first, last, err := parseRangeHeader(rangeHeader)
	if err != nil {
		h.serveFullBlob(w, r, name, d, digestStr)

		return
	}

	h.serveRangeBlob(w, r, name, d, digestStr, first, last)
}

func (h *RegistryHandler) serveFullBlob(
	w http.ResponseWriter,
	r *http.Request,
	name string,
	d digest.Digest,
	digestStr string,
) {
	rc, size, err := h.blob.GetBlob(r.Context(), name, d)
	if err != nil {
		h.registryErrToHTTP(w, "serveFullBlob", err)

		return
	}

	defer func() {
		closeErr := rc.Close()
		if closeErr != nil {
			h.logger.Warn("serveFullBlob: close reader", logtag.Err(closeErr))
		}
	}()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Docker-Content-Digest", digestStr)
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, rc)
	if err != nil {
		h.logger.Warn("serveFullBlob: write response", logtag.Err(err))
	}
}

func (h *RegistryHandler) serveRangeBlob(
	w http.ResponseWriter,
	r *http.Request,
	name string,
	d digest.Digest,
	digestStr string,
	first, last int64,
) {
	rc, totalSize, err := h.blob.GetBlobRange(r.Context(), name, d, first, last)
	if err != nil {
		if errors.Is(err, ocierror.ErrRangeNotSatisfiable) {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)

			return
		}

		h.registryErrToHTTP(w, "serveRangeBlob", err)

		return
	}

	defer func() {
		closeErr := rc.Close()
		if closeErr != nil {
			h.logger.Warn("serveRangeBlob: close reader", logtag.Err(closeErr))
		}
	}()

	partialSize := last - first + 1

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", first, last, totalSize))
	w.Header().Set("Content-Length", strconv.FormatInt(partialSize, 10))
	w.Header().Set("Docker-Content-Digest", digestStr)
	w.WriteHeader(http.StatusPartialContent)

	_, err = io.Copy(w, rc)
	if err != nil {
		h.logger.Warn("serveRangeBlob: write response", logtag.Err(err))
	}
}

func blobURL(name, digestStr string) string {
	return fmt.Sprintf("/v2/%s/blobs/%s", name, digestStr)
}
