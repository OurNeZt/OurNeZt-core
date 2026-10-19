package domain

import (
	"fmt"
	"unicode/utf8"

	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

func ValidateAuthInputLengths(email, displayName, password string) error {
	if !utf8.ValidString(email) || utf8.RuneCountInString(email) > 254 {
		return fmt.Errorf("%w: email must not exceed 254 characters", apperror.ErrInvalidArgument)
	}
	if !utf8.ValidString(displayName) || utf8.RuneCountInString(displayName) > 120 {
		return fmt.Errorf("%w: display_name must not exceed 120 characters", apperror.ErrInvalidArgument)
	}
	if !utf8.ValidString(password) || len(password) > 1024 {
		return fmt.Errorf("%w: password must not exceed 1024 bytes", apperror.ErrInvalidArgument)
	}
	return nil
}
