package ocierror

import "errors"

var (
	ErrNameInvalid         = errors.New("name invalid")
	ErrNameExists          = errors.New("name already exists")
	ErrNameUnknown         = errors.New("name unknown")
	ErrDigestInvalid       = errors.New("digest invalid")
	ErrBlobUnknown         = errors.New("blob unknown")
	ErrBlobUploadUnknown   = errors.New("blob upload unknown")
	ErrSizeInvalid         = errors.New("size invalid")
	ErrRangeNotSatisfiable = errors.New("range not satisfiable")
	ErrUnsupported         = errors.New("unsupported")

	ErrManifestUnknown     = errors.New("manifest unknown")
	ErrManifestInvalid     = errors.New("manifest invalid")
	ErrManifestBlobUnknown = errors.New("manifest blob unknown")
)
