package blob

import "errors"

var (
	ErrDigestInvalid       = errors.New("digest invalid")
	ErrBlobUnknown         = errors.New("blob unknown")
	ErrBlobUploadUnknown   = errors.New("blob upload unknown")
	ErrSizeInvalid         = errors.New("size invalid")
	ErrRangeNotSatisfiable = errors.New("range not satisfiable")
)
