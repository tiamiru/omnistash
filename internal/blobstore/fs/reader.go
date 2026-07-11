package fs

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"

	"github.com/opencontainers/go-digest"

	"github.com/tiamiru/omnistash/internal/blobstore"
)

func (s *FilesystemBlobStore) GetBlob(d digest.Digest) (io.ReadCloser, int64, error) {
	err := blobstore.ValidateDigest(d)
	if err != nil {
		return nil, 0, fmt.Errorf("get blob %s: %w", d, err)
	}

	p := buildBlobPath(s.prefix, string(s.partition), d)
	f, err := os.Open(p) //nolint:gosec
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return nil, 0, fmt.Errorf("%w: digest=%s", blobstore.ErrBlobUnknown, d)
		}

		return nil, 0, fmt.Errorf("get blob %s: open: %w", d, err)
	}

	fi, err := f.Stat()
	if err != nil {
		statErr := fmt.Errorf("get blob %s: stat: %w", d, err)
		closeErr := f.Close()
		if closeErr != nil {
			return nil, 0, errors.Join(statErr, fmt.Errorf("get blob %s: close: %w", d, closeErr))
		}

		return nil, 0, statErr
	}

	return f, fi.Size(), nil
}

func (s *FilesystemBlobStore) GetBlobRange(d digest.Digest, first, last int64, w io.Writer) (err error) {
	err = blobstore.ValidateDigest(d)
	if err != nil {
		return fmt.Errorf("get blob range %s: %w", d, err)
	}

	if first < 0 || last < first {
		return fmt.Errorf("%w: first=%d last=%d", blobstore.ErrInvalidRange, first, last)
	}

	p := buildBlobPath(s.prefix, string(s.partition), d)
	f, err := os.Open(p) //nolint:gosec
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return fmt.Errorf("%w: digest=%s", blobstore.ErrBlobUnknown, d)
		}

		return fmt.Errorf("get blob range %s: open: %w", d, err)
	}

	defer func() {
		closeErr := f.Close()
		if closeErr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("get blob range %s: close: %w", d, closeErr))
		}
	}()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("get blob range %s: stat: %w", d, err)
	}

	if last >= fi.Size() {
		return fmt.Errorf("%w: last=%d size=%d", blobstore.ErrRangeNotSatisfiable, last, fi.Size())
	}

	_, err = f.Seek(first, io.SeekStart)
	if err != nil {
		return fmt.Errorf("get blob range %s: seek: %w", d, err)
	}

	_, err = io.CopyN(w, f, last-first+1)
	if err != nil {
		return fmt.Errorf("get blob range %s: copy: %w", d, err)
	}

	return nil
}
