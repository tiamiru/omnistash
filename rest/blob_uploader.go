package rest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blob"
	"github.com/tiamiru/omnistash/internal/namespace"
)

// handlePostBlobUploads implements POST /v2/{name}/blobs/uploads/.
// Supports cross-repo mount, monolithic upload, and multi-part uploads.
func (h *RegistryHandler) handlePostBlobUploads(w http.ResponseWriter, r *http.Request) {
	name := repoName(r)
	q := r.URL.Query()

	alg := q.Get("digest-algorithm")
	if alg != "" {
		if !digest.Algorithm(alg).Available() {
			h.writeOCIError(
				w,
				"handlePostBlobUploads",
				http.StatusBadRequest,
				ociCodeUnsupported,
				"unsupported digest algorithm",
			)

			return
		}
	}

	mountDigest := q.Get("mount")
	if mountDigest != "" {
		sourceNs := q.Get("from")
		h.handleMountBlob(w, r, name, sourceNs, mountDigest)

		return
	}

	digestParam := q.Get("digest")
	if digestParam != "" {
		h.handleMonolithicUpload(w, r, name, digestParam)

		return
	}

	h.handleStartUpload(w, r, name)
}

func (h *RegistryHandler) handleStartUpload(w http.ResponseWriter, r *http.Request, name string) {
	uploadID, err := h.blob.InitiateUpload(r.Context(), name)
	if err != nil {
		h.registryErrToHTTP(w, "handleStartUpload", err)

		return
	}

	w.Header().Set("Location", uploadSessionURL(name, uploadID))
	w.WriteHeader(http.StatusAccepted)
}

func (h *RegistryHandler) handleMonolithicUpload(
	w http.ResponseWriter,
	r *http.Request,
	name, digestStr string,
) {
	err := namespace.ValidateName(name)
	if err != nil {
		h.registryErrToHTTP(w, "handleMonolithicUpload", err)

		return
	}

	d := digest.Digest(digestStr)

	err = blob.ValidateDigest(d)
	if err != nil {
		h.registryErrToHTTP(w, "handleMonolithicUpload", err)

		return
	}

	clStr := r.Header.Get("Content-Length")
	size, err := strconv.ParseInt(clStr, 10, 64)
	if err != nil || size < 0 {
		h.writeOCIError(
			w,
			"handleMonolithicUpload",
			http.StatusBadRequest,
			ociCodeSizeInvalid,
			"invalid Content-Length",
		)

		return
	}

	err = h.blob.MonolithicUpload(r.Context(), name, d, size, r.Body)
	if err != nil {
		h.registryErrToHTTP(w, "handleMonolithicUpload", err)

		return
	}

	w.Header().Set("Location", blobURL(name, digestStr))
	w.Header().Set("Docker-Content-Digest", digestStr)
	w.WriteHeader(http.StatusCreated)
}

func (h *RegistryHandler) handleMountBlob(
	w http.ResponseWriter,
	r *http.Request,
	targetName, sourceName, digestStr string,
) {
	d := digest.Digest(digestStr)

	err := h.blob.MountBlob(r.Context(), sourceName, targetName, d)
	if errors.Is(err, blob.ErrMountFailed) {
		h.handleStartUpload(w, r, targetName)

		return
	}

	if err != nil {
		h.registryErrToHTTP(w, "handleMountBlob", err)

		return
	}

	w.Header().Set("Location", blobURL(targetName, digestStr))
	w.Header().Set("Docker-Content-Digest", digestStr)
	w.WriteHeader(http.StatusCreated)
}

// handleGetBlobUpload implements GET /v2/{name}/blobs/uploads/{uuid}.
func (h *RegistryHandler) handleGetBlobUpload(w http.ResponseWriter, r *http.Request) {
	name := repoName(r)
	uploadID := r.PathValue("uuid")

	offset, err := h.blob.GetUploadStatus(r.Context(), name, uploadID)
	if err != nil {
		h.registryErrToHTTP(w, "handleGetBlobUpload", err)

		return
	}

	w.Header().Set("Range", fmt.Sprintf("0-%d", max(0, offset-1)))
	w.Header().Set("Location", uploadSessionURL(name, uploadID))
	w.WriteHeader(http.StatusNoContent)
}

// handlePatchBlobUpload implements PATCH /v2/{name}/blobs/uploads/{uuid}.
func (h *RegistryHandler) handlePatchBlobUpload(w http.ResponseWriter, r *http.Request) {
	name := repoName(r)
	uploadID := r.PathValue("uuid")

	err := namespace.ValidateName(name)
	if err != nil {
		h.registryErrToHTTP(w, "handlePatchBlobUpload", err)

		return
	}

	var offset int64

	contentRangeHeader := r.Header.Get("Content-Range")
	if contentRangeHeader != "" {
		first, _, err := parseContentRangeHeader(contentRangeHeader)
		if err != nil {
			h.registryErrToHTTP(w, "handlePatchBlobUpload", err)

			return
		}

		offset = first
	}

	newOffset, err := h.blob.AppendChunk(r.Context(), name, uploadID, offset, r.Body)
	if err != nil {
		h.registryErrToHTTP(w, "handlePatchBlobUpload", err)

		return
	}

	w.Header().Set("Location", uploadSessionURL(name, uploadID))
	w.Header().Set("Range", fmt.Sprintf("0-%d", newOffset-1))
	w.WriteHeader(http.StatusAccepted)
}

// handlePutBlobUpload implements PUT /v2/{name}/blobs/uploads/{uuid}?digest={digest}.
func (h *RegistryHandler) handlePutBlobUpload(w http.ResponseWriter, r *http.Request) {
	name := repoName(r)
	uploadID := r.PathValue("uuid")

	err := namespace.ValidateName(name)
	if err != nil {
		h.registryErrToHTTP(w, "handlePutBlobUpload", err)

		return
	}

	digestStr := r.URL.Query().Get("digest")

	if digestStr == "" {
		h.writeOCIError(
			w,
			"handlePutBlobUpload",
			http.StatusBadRequest,
			ociCodeDigestInvalid,
			"digest query parameter required",
		)

		return
	}

	d := digest.Digest(digestStr)

	err = blob.ValidateDigest(d)
	if err != nil {
		h.registryErrToHTTP(w, "handlePutBlobUpload", err)

		return
	}

	// If Content-Length > 0 the PUT body is a final chunk.
	var finalChunk io.Reader
	contentLengthStr := r.Header.Get("Content-Length")
	if contentLengthStr != "" {
		contentLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
		if err == nil && contentLength > 0 {
			finalChunk = r.Body
		}
	}

	err = h.blob.CommitUpload(r.Context(), name, uploadID, d, finalChunk)
	if err != nil {
		h.registryErrToHTTP(w, "handlePutBlobUpload", err)

		return
	}

	w.Header().Set("Location", blobURL(name, digestStr))
	w.Header().Set("Docker-Content-Digest", digestStr)
	w.WriteHeader(http.StatusCreated)
}

// handleDeleteBlobUpload implements DELETE /v2/{name}/blobs/uploads/{uuid}.
func (h *RegistryHandler) handleDeleteBlobUpload(w http.ResponseWriter, r *http.Request) {
	name := repoName(r)
	uploadID := r.PathValue("uuid")

	err := h.blob.CancelUpload(r.Context(), name, uploadID)
	if err != nil {
		h.registryErrToHTTP(w, "handleDeleteBlobUpload", err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func uploadSessionURL(name, uploadID string) string {
	return fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uploadID)
}
