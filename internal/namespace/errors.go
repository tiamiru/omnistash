package namespace

import "errors"

var (
	ErrNameInvalid = errors.New("name invalid")
	ErrNameExists  = errors.New("name already exists")
	ErrNameUnknown = errors.New("name unknown")
)
