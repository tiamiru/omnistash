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
	Tags     []string
}
