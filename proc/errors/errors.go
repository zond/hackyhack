package errors

import "errors"

var (
	ErrUnknownBlobType     = errors.New("Unknown blob type.")
	ErrOnlyRegisterOnce    = errors.New("Can only Register once.")
	ErrNoSuchResource      = errors.New("No such resource.")
	ErrUnavailableResource = errors.New("Unavailable resource.")
)
