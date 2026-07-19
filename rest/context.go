package rest

import (
	"context"
	"net/http"
)

type contextKey int

const repoNameKey contextKey = iota

func withRepoName(r *http.Request, name string) *http.Request {
	return r.Clone(context.WithValue(r.Context(), repoNameKey, name))
}

func repoName(r *http.Request) string {
	name, _ := r.Context().Value(repoNameKey).(string)

	return name
}
