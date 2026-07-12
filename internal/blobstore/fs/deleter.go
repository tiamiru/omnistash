package fs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/opencontainers/go-digest"

	blobs "github.com/tiamiru/omnistash/internal/blobstore"
)

func (s *FilesystemBlobStore) StartVacuumProcess() {
	stop := s.vacuumManager.Start()
	if stop != nil {
		s.stopVacuum = stop
	}
}

func (s *FilesystemBlobStore) StopVacuumProcess() error {
	return s.stopVacuum()
}

func (s *FilesystemBlobStore) DeleteBlob(ctx context.Context, d digest.Digest) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("delete blob: %w", err)
	}

	err = blobs.ValidateDigest(d)
	if err != nil {
		return fmt.Errorf("delete blob %s: %w", d, err)
	}

	dir := filepath.Join(s.prefix, string(s.partition))
	p := buildBlobPath(s.prefix, string(s.partition), d)

	removed, pathErrs, removeErr := s.vacuumManager.removeBatch(ctx, dir, []string{p})
	if removeErr != nil {
		return fmt.Errorf("delete blob %s: %w", d, removeErr)
	}
	pathErr := errors.Join(pathErrs...)
	if pathErr != nil {
		return fmt.Errorf("delete blob %s: %w", d, pathErr)
	}
	if removed == 0 {
		return fmt.Errorf("%w: digest=%s", blobs.ErrBlobUnknown, d)
	}

	return nil
}

func (s *FilesystemBlobStore) BatchDeleteBlobs(ctx context.Context, digests []digest.Digest) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("batch delete blobs: %w", err)
	}

	dir := filepath.Join(s.prefix, string(s.partition))

	var (
		paths []string
		errs  []error
	)

	for _, d := range digests {
		valErr := blobs.ValidateDigest(d)
		if valErr != nil {
			errs = append(errs, valErr)

			continue
		}

		paths = append(paths, buildBlobPath(s.prefix, string(s.partition), d))
	}

	removed, pathErrs, removeErr := s.vacuumManager.removeBatch(ctx, dir, paths)
	if removeErr != nil {
		errs = append(errs, removeErr)
	}
	errs = append(errs, pathErrs...)

	joined := errors.Join(errs...)
	if removed > 0 && joined != nil {
		return fmt.Errorf("%w: %w", blobs.ErrPartialDeletion, joined)
	}

	return joined
}
