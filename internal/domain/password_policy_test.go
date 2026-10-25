package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

func TestPasswordPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, password, reason string
	}{
		{"empty", "", "blank or whitespace-only"},
		{"unicode whitespace", strings.Repeat(" \t\u2003", 15), "blank or whitespace-only"},
		{"one character", "x", "at least 15 characters"},
		{"below minimum", "quiet-otter-ba", "at least 15 characters"},
		{"unicode counts characters", strings.Repeat("界", 14), "at least 15 characters"},
		{"above byte maximum", strings.Repeat("a", 1025), "must not exceed 1024 bytes"},
		{"unicode above byte maximum", strings.Repeat("界", 342), "must not exceed 1024 bytes"},
		{"invalid UTF-8", "long-enough-password\xff", "valid UTF-8"},
		{"listed long password", "123456789987654321", "too common"},
		{"listed case variant", "PASSWORDPASSWORD", "too common"},
		{"padded common password", "    PASSWORD    ", "must not contain whitespace"},
		{"leading space", " quiet-otter-bay", "must not contain whitespace"},
		{"trailing space", "quiet-otter-bay ", "must not contain whitespace"},
		{"internal space", "quiet otter bay", "must not contain whitespace"},
		{"tab", "quiet\totter-bay", "must not contain whitespace"},
		{"line break", "quiet-otter\nbay", "must not contain whitespace"},
		{"nonbreaking space", "quiet\u00a0otter-bay", "must not contain whitespace"},
		{"unicode space", "quiet\u2003otter-bay", "must not contain whitespace"},
		{"minimum", "quiet-otter-bay", ""},
		{"unicode minimum", strings.Repeat("界", 15), ""},
		{"maximum bytes", strings.Repeat("a", 1024), ""},
		{"unicode maximum bytes", strings.Repeat("界", 341) + "a", ""},
		{"no required character mix", "apeacefulgardennearhome", ""},
		{"no substring ban", "the-passwordpassword-story", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateNewPassword(tc.password)
			if tc.reason == "" {
				if err != nil {
					t.Fatalf("valid password rejected: %v", err)
				}
				return
			}
			if !errors.Is(err, apperror.ErrInvalidArgument) || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("error = %v, want InvalidArgument with %q", err, tc.reason)
			}
			if tc.password != "" && strings.Contains(err.Error(), tc.password) {
				t.Fatal("validation error exposed the password")
			}
		})
	}
}
