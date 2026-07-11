package logtag

import "log/slog"

// Standardised slog attribute keys used throughout the codebase.
const (
	KeyDigest = "digest"
	KeyTag    = "tag"
	KeySize   = "size"
	KeyPath   = "path"
	KeyErr    = "error"
)

func Tag(name string) slog.Attr { return slog.String(KeyTag, name) }
func Path(p string) slog.Attr   { return slog.String(KeyPath, p) }
func Err(err error) slog.Attr   { return slog.Any(KeyErr, err) }
