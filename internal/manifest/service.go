package manifest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/logtag"
	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/namespace"
	"github.com/tiamiru/omnistash/internal/ocierror"
)

// MetaOps is the subset of metastore.MetadataStore used by this service.
type MetaOps interface {
	NamespaceExists(ctx context.Context, namespace string) (bool, error)
	Atomic(ctx context.Context, fn func(ctx context.Context, tx metastore.TxOps) error) error
}

// BlobStore is the subset of blobstore.BlobStore used by this service.
type BlobStore interface {
	PutBlob(namespace string, d digest.Digest, size int64, r io.Reader) (int64, error)
	GetBlob(namespace string, d digest.Digest) (io.ReadCloser, int64, error)
	DeleteBlob(ctx context.Context, namespace string, d digest.Digest) error
}

type Service struct {
	meta   MetaOps
	blobs  BlobStore
	logger *slog.Logger
}

func NewService(meta MetaOps, blobs BlobStore, logger *slog.Logger) *Service {
	return &Service{meta: meta, blobs: blobs, logger: logger}
}

func (s *Service) PutManifest(
	ctx context.Context,
	ns, reference, contentType string,
	body []byte,
) (PutResult, error) {
	err := namespace.ValidateName(ns)
	if err != nil {
		return PutResult{}, fmt.Errorf("PutManifest: %w", err)
	}

	refDigest, err := validateDigestReference(reference)
	if err != nil {
		return PutResult{}, fmt.Errorf("PutManifest: %w", err)
	}

	computedDigest := digest.FromBytes(body)
	if computedDigest != refDigest {
		return PutResult{}, fmt.Errorf(
			"PutManifest: %w: reference=%s body=%s",
			ocierror.ErrDigestInvalid, refDigest, computedDigest,
		)
	}

	err = checkNamespaceExists(ctx, s.meta, ns)
	if err != nil {
		return PutResult{}, fmt.Errorf("PutManifest: %w", err)
	}

	partial, parseErr := parseManifestBody(contentType, body)
	if parseErr != nil {
		return PutResult{}, fmt.Errorf("PutManifest: %w", parseErr)
	}

	size := int64(len(body))
	_, putErr := s.blobs.PutBlob(ns, computedDigest, size, bytes.NewReader(body))
	if putErr != nil && !errors.Is(putErr, blobstore.ErrBlobCommitted) {
		return PutResult{}, fmt.Errorf("PutManifest: %w", putErr)
	}

	referrerRow := deriveReferrer(partial, computedDigest, size)
	err = s.storeManifest(ctx, ns, partial.MediaType, computedDigest, size, referrerRow)
	if err != nil {
		return PutResult{}, fmt.Errorf("PutManifest: %w", err)
	}

	var subject *digest.Digest
	if partial.Subject != nil && partial.Subject.Digest != "" {
		d := partial.Subject.Digest
		subject = &d
	}

	return PutResult{
		Digest:   computedDigest,
		Location: fmt.Sprintf("/v2/%s/manifests/%s", ns, computedDigest),
		Subject:  subject,
	}, nil
}

func (s *Service) GetManifest(
	ctx context.Context,
	namespace, reference string,
) (ManifestInfo, io.ReadCloser, error) {
	row, err := s.resolveManifest(ctx, namespace, reference)
	if err != nil {
		return ManifestInfo{}, nil, fmt.Errorf("GetManifest: %w", err)
	}

	rc, _, err := s.blobs.GetBlob(namespace, row.Digest)
	if err != nil {
		if errors.Is(err, blobstore.ErrBlobUnknown) {
			return ManifestInfo{}, nil, fmt.Errorf("GetManifest: %w", ocierror.ErrManifestUnknown)
		}

		return ManifestInfo{}, nil, fmt.Errorf("GetManifest: %w", err)
	}

	return convertRowToManifestInfo(row), rc, nil
}

func (s *Service) HeadManifest(
	ctx context.Context,
	namespace, reference string,
) (ManifestInfo, error) {
	row, err := s.resolveManifest(ctx, namespace, reference)
	if err != nil {
		return ManifestInfo{}, fmt.Errorf("HeadManifest: %w", err)
	}

	return convertRowToManifestInfo(row), nil
}

func (s *Service) DeleteManifest(ctx context.Context, namespace, reference string) error {
	refDigest, err := s.validateContext(ctx, namespace, reference)
	if err != nil {
		return fmt.Errorf("DeleteManifest: %w", err)
	}

	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		deleteErr := tx.DeleteManifestByDigest(ctx, namespace, refDigest)
		if deleteErr != nil {
			return deleteErr
		}

		return tx.DeleteReferrer(ctx, namespace, refDigest)
	})
	if err != nil {
		return fmt.Errorf("DeleteManifest: %w", err)
	}

	delErr := s.blobs.DeleteBlob(ctx, namespace, refDigest)
	if delErr != nil {
		s.logger.Warn("DeleteManifest: delete blob bytes", logtag.Err(delErr))
	}

	return nil
}

func (s *Service) validateContext(ctx context.Context, ns, reference string) (digest.Digest, error) {
	err := namespace.ValidateName(ns)
	if err != nil {
		return "", err
	}

	d, err := validateDigestReference(reference)
	if err != nil {
		return "", err
	}

	err = checkNamespaceExists(ctx, s.meta, ns)
	if err != nil {
		return "", err
	}

	return d, nil
}

func (s *Service) storeManifest(
	ctx context.Context,
	ns string,
	mediaType string,
	d digest.Digest,
	size int64,
	ref *metastore.ReferrerRow,
) error {
	return s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		err := tx.InsertManifest(ctx, ns, d, mediaType, size)
		if err != nil {
			return err
		}

		if ref != nil {
			return tx.UpsertReferrer(ctx, ns, *ref)
		}

		return nil
	})
}

func (s *Service) resolveManifest(
	ctx context.Context,
	namespace, reference string,
) (metastore.ManifestRow, error) {
	refDigest, err := s.validateContext(ctx, namespace, reference)
	if err != nil {
		return metastore.ManifestRow{}, err
	}

	var row metastore.ManifestRow

	err = s.meta.Atomic(ctx, func(ctx context.Context, tx metastore.TxOps) error {
		var txErr error
		row, txErr = tx.GetManifestByDigest(ctx, namespace, refDigest)

		return txErr
	})
	if err != nil {
		return metastore.ManifestRow{}, err
	}

	return row, nil
}
