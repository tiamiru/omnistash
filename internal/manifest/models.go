package manifest

import "github.com/opencontainers/go-digest"

type ManifestInfo struct {
	Digest    digest.Digest
	MediaType string
	Size      int64
}

type PutResult struct {
	Digest   digest.Digest
	Location string
	Subject  *digest.Digest
}

type partialManifest struct {
	MediaType string          `json:"mediaType"`
	Subject   *partialSubject `json:"subject,omitempty"`
}

type partialSubject struct {
	Digest digest.Digest `json:"digest"`
}
