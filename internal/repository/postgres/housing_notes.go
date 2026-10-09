package postgres

import (
	"context"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

func (r HousingRepository) UpdateHousingNotes(ctx context.Context, housingID domain.ID, notes string, actorID domain.ID) (string, error) {
	if strings.TrimSpace(string(housingID)) == "" {
		return "", apperror.ErrInvalidArgument
	}
	if err := domain.ValidateHousingNotes(notes); err != nil {
		return "", err
	}
	// Update only the notepad, with membership checked in the same statement.
	var saved string
	err := r.pool.QueryRow(ctx, `
  UPDATE housing_options h SET notes = $2, updated_at = now()
  FROM family_members fm
  WHERE h.id = $1::uuid AND fm.family_id = h.family_id
    AND fm.user_id = $3::uuid AND fm.role IN ('owner', 'admin', 'member')
  RETURNING h.notes
 `, string(housingID), notes, string(actorID)).Scan(&saved)
	if err != nil {
		return "", normalizeError(err)
	}
	return saved, nil
}
