package rest

import (
	"net/http"
	"strings"
)

// splitNamespacedPath splits the url path by finding the rightmost occurrence of any
// known OCI suffix separator.
func splitNamespacedPath(urlPath string) (string, string, bool) {
	const prefix = "/v2/"
	if !strings.HasPrefix(urlPath, prefix) {
		return "", "", false
	}

	ociPathSeparators := []string{
		"/blobs/",
		"/manifests/",
		"/tags/",
		"/referrers/",
	}

	rest := urlPath[len(prefix):]

	bestIdx := -1

	for _, sep := range ociPathSeparators {
		if idx := strings.LastIndex(rest, sep); idx > bestIdx {
			bestIdx = idx
		}
	}

	if bestIdx < 0 {
		return "", "", false
	}

	name := rest[:bestIdx]
	tail := rest[bestIdx:]

	return name, tail, name != ""
}

// withNamespace parses the /v2/{name} prefix from the URL, validates the
// repository name, stores it in the request context.
func withNamespace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, tail, ok := splitNamespacedPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)

			return
		}

		r2 := withRepoName(r, name)
		r2.URL.Path = tail
		r2.URL.RawPath = ""

		next.ServeHTTP(w, r2)
	})
}
