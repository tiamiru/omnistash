package fs

import (
	"log/slog"

	"github.com/tiamiru/omnistash/internal/clock"
)

type Option func(*FilesystemBlobStore)

func WithLogger(logger *slog.Logger) Option {
	return func(s *FilesystemBlobStore) {
		if logger != nil {
			s.logger = logger
		}
	}
}

func WithClock(c clock.Clock) Option {
	return func(s *FilesystemBlobStore) {
		if c != nil {
			s.clock = c
		}
	}
}
