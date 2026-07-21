package rest

import (
	"net/http"
)

func setupRoutes(mux *http.ServeMux, h *RegistryHandler) {
	mux.HandleFunc("GET /health", h.HandleHealth)

	mux.HandleFunc("POST /v2/namespaces", h.HandleCreateNamespace)
	mux.HandleFunc("DELETE /v2/namespaces/{namespace...}", h.HandleDeleteNamespace)

	mux.HandleFunc("GET /v2/{$}", h.HandleBaseCheck)

	ociRouter := buildOCIRouter(h)
	v2Handler := withNamespace(ociRouter)

	mux.Handle("/v2/{rest...}", v2Handler)
}

// buildOCIRouter returns a mux that matches OCI endpoint suffixes.
// It expects the request URL to already have the /v2/{name} prefix stripped
// and the repository name stored in context via withNamespace.
func buildOCIRouter(h *RegistryHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /blobs/{digest}", h.handleGetBlob)
	mux.HandleFunc("HEAD /blobs/{digest}", h.handleHeadBlob)

	mux.HandleFunc("POST /blobs/uploads/", h.handlePostBlobUploads)
	mux.HandleFunc("GET /blobs/uploads/{uuid}", h.handleGetBlobUpload)
	mux.HandleFunc("PATCH /blobs/uploads/{uuid}", h.handlePatchBlobUpload)
	mux.HandleFunc("PUT /blobs/uploads/{uuid}", h.handlePutBlobUpload)
	mux.HandleFunc("DELETE /blobs/uploads/{uuid}", h.handleDeleteBlobUpload)

	mux.HandleFunc("GET /manifests/{reference}", h.handleGetManifest)
	mux.HandleFunc("HEAD /manifests/{reference}", h.handleHeadManifest)
	mux.HandleFunc("PUT /manifests/{reference}", h.handlePutManifest)
	mux.HandleFunc("DELETE /manifests/{reference}", h.handleDeleteManifest)

	mux.HandleFunc("GET /referrers/{digest}", h.handleGetReferrers)

	return mux
}
