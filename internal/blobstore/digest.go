package blobstore

import (
	"fmt"
	"hash"

	"github.com/opencontainers/go-digest"
)

func ValidateDigest(d digest.Digest) error {
	err := d.Validate()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidDigest, d)
	}

	return nil
}

func HasherFor(d digest.Digest) (hash.Hash, error) {
	algo := d.Algorithm()
	if !algo.Available() {
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrInvalidDigest, algo)
	}

	return algo.Hash(), nil
}

func MatchDigest(expected, actual digest.Digest) error {
	if expected != actual {
		return fmt.Errorf("%w: expected=%s actual=%s", ErrDigestMismatch, expected, actual)
	}

	return nil
}
