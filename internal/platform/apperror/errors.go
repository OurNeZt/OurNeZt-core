package apperror

import "errors"

var (
	ErrQuotaExceeded          = errors.New("resource quota exceeded")
	ErrUnauthenticated        = errors.New("unauthenticated")
	ErrForbidden              = errors.New("forbidden")
	ErrNotFound               = errors.New("not found")
	ErrConflict               = errors.New("conflict")
	ErrInvalidArgument        = errors.New("invalid argument")
	ErrDisabledUser           = errors.New("disabled user")
	ErrPasswordChangeRequired = errors.New("password change required")
)
