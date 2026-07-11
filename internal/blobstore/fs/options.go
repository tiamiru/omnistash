package fs

import "log/slog"

type Option func(*FilesystemBlobStore)

func WithLogger(logger *slog.Logger) Option {
	return func(s *FilesystemBlobStore) {
		if logger != nil {
			s.logger = logger
		}
	}
}
