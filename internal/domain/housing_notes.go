package domain

import (
	"unicode/utf8"

	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

const HousingNotesMaxLength = 10000

func ValidateHousingNotes(notes string) error {
	if !utf8.ValidString(notes) || utf8.RuneCountInString(notes) > HousingNotesMaxLength {
		return apperror.ErrInvalidArgument
	}
	return nil
}
