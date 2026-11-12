package domain

import (
	_ "embed"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

const (
	MinPasswordCharacters = 15
	MaxPasswordBytes      = 1024
)

//go:embed passwords/common.txt
var commonPasswordData string

var commonPasswords = func() map[string]struct{} {
	passwords := make(map[string]struct{})
	for _, password := range strings.Split(commonPasswordData, "\n") {
		password = strings.ToLower(strings.TrimSpace(password))
		if password != "" {
			passwords[password] = struct{}{}
		}
	}
	return passwords
}()

// ValidateNewPassword applies only when setting credentials. Login and current
// password verification must remain compatible with existing passwords.
func ValidateNewPassword(password string) error {
	if err := ValidateAuthInputLengths("", "", password); err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("%w: password must not be blank or whitespace-only", apperror.ErrInvalidArgument)
	}
	if strings.IndexFunc(password, unicode.IsSpace) >= 0 {
		return fmt.Errorf("%w: password must not contain whitespace", apperror.ErrInvalidArgument)
	}
	if utf8.RuneCountInString(password) < MinPasswordCharacters {
		return fmt.Errorf("%w: password must contain at least %d characters", apperror.ErrInvalidArgument, MinPasswordCharacters)
	}
	// Only the blocklist comparison is folded. Hash the exact original bytes.
	if _, blocked := commonPasswords[strings.ToLower(password)]; blocked {
		return fmt.Errorf("%w: password is too common; choose a less predictable password", apperror.ErrInvalidArgument)
	}
	return nil
}
