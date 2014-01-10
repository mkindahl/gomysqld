package stable

import "errors"

var (
	ErrInvalidDist     = errors.New("invalid distribution")
	ErrUnpackFailure   = errors.New("unable to unpack distribution")
	ErrVersionNotFound = errors.New("version not found")
	ErrStableExists    = errors.New("stable exists")
)
