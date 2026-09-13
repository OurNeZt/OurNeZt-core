package postgres

import (
	"context"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/jackc/pgx/v5"
)

func (r HousingRepository) ListHousingCriteria(ctx context.Context, familyID, actorID domain.ID) ([]domain.HousingCriterion, error) {
	allowed, err := r.hasFamilyAccess(ctx, actorID, familyID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperror.ErrForbidden
	}
	rows, err := r.pool.Query(ctx, `SELECT id::text, family_id::text, name, description, display_order, weight FROM housing_checklist_criteria WHERE family_id=$1::uuid AND deleted_at IS NULL ORDER BY display_order, id`, string(familyID))
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()
	criteria := []domain.HousingCriterion{}
	for rows.Next() {
		var c domain.HousingCriterion
		if err := rows.Scan(&c.ID, &c.FamilyID, &c.Name, &c.Description, &c.DisplayOrder, &c.Weight); err != nil {
			return nil, normalizeError(err)
		}
		criteria = append(criteria, c)
	}
	return criteria, normalizeError(rows.Err())
}

// Serialize template mutations per family so positions stay contiguous even when
// two family members add, move, or retire criteria concurrently.
func (r HousingRepository) checklistTransaction(ctx context.Context, familyID, actorID domain.ID) (pgx.Tx, error) {
	if err := r.assertFamilyWriter(ctx, actorID, familyID); err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, normalizeError(err)
	}
	var id string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM families WHERE id=$1::uuid FOR UPDATE`, string(familyID)).Scan(&id); err != nil {
		_ = tx.Rollback(ctx)
		return nil, normalizeError(err)
	}
	return tx, nil
}

func checklistIDs(ctx context.Context, tx pgx.Tx, familyID domain.ID) ([]domain.ID, error) {
	rows, err := tx.Query(ctx, `SELECT id::text FROM housing_checklist_criteria WHERE family_id=$1::uuid AND deleted_at IS NULL ORDER BY display_order, id`, string(familyID))
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()
	ids := []domain.ID{}
	for rows.Next() {
		var id domain.ID
		if err := rows.Scan(&id); err != nil {
			return nil, normalizeError(err)
		}
		ids = append(ids, id)
	}
	return ids, normalizeError(rows.Err())
}

func writeChecklistOrder(ctx context.Context, tx pgx.Tx, ids []domain.ID) error {
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE housing_checklist_criteria SET display_order=$2, updated_at=now() WHERE id=$1::uuid`, string(id), i); err != nil {
			return normalizeError(err)
		}
	}
	return nil
}

func (r HousingRepository) SaveHousingCriterion(ctx context.Context, c domain.HousingCriterion, actorID domain.ID) (domain.HousingCriterion, error) {
	c.Name = strings.TrimSpace(c.Name)
	if err := domain.ValidateHousingCriterion(c); err != nil {
		return domain.HousingCriterion{}, err
	}
	tx, err := r.checklistTransaction(ctx, c.FamilyID, actorID)
	if err != nil {
		return domain.HousingCriterion{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	ids, err := checklistIDs(ctx, tx, c.FamilyID)
	if err != nil {
		return domain.HousingCriterion{}, err
	}
	if c.ID != "" {
		found := false
		for i, id := range ids {
			if id == c.ID {
				ids = append(ids[:i], ids[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return domain.HousingCriterion{}, apperror.ErrNotFound
		}
	}
	if int(c.DisplayOrder) > len(ids) {
		return domain.HousingCriterion{}, apperror.ErrInvalidArgument
	}
	if c.ID == "" {
		err = tx.QueryRow(ctx, `INSERT INTO housing_checklist_criteria(family_id,name,description,display_order,weight) VALUES($1::uuid,$2,$3,$4,$5) RETURNING id::text`, string(c.FamilyID), c.Name, c.Description, c.DisplayOrder, c.Weight).Scan(&c.ID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE housing_checklist_criteria SET name=$2,description=$3,weight=$4,updated_at=now() WHERE id=$1::uuid`, string(c.ID), c.Name, c.Description, c.Weight)
	}
	if err != nil {
		return domain.HousingCriterion{}, normalizeError(err)
	}
	pos := int(c.DisplayOrder)
	ids = append(ids, "")
	copy(ids[pos+1:], ids[pos:])
	ids[pos] = c.ID
	if err := writeChecklistOrder(ctx, tx, ids); err != nil {
		return domain.HousingCriterion{}, err
	}
	return c, normalizeError(tx.Commit(ctx))
}

func (r HousingRepository) DeleteHousingCriterion(ctx context.Context, familyID, criterionID, actorID domain.ID) error {
	tx, err := r.checklistTransaction(ctx, familyID, actorID)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	tag, err := tx.Exec(ctx, `UPDATE housing_checklist_criteria SET deleted_at=now(),updated_at=now() WHERE id=$1::uuid AND family_id=$2::uuid AND deleted_at IS NULL`, string(criterionID), string(familyID))
	if err != nil {
		return normalizeError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	ids, err := checklistIDs(ctx, tx, familyID)
	if err != nil {
		return err
	}
	if err := writeChecklistOrder(ctx, tx, ids); err != nil {
		return err
	}
	return normalizeError(tx.Commit(ctx))
}

func (r HousingRepository) ReorderHousingCriteria(ctx context.Context, familyID domain.ID, ids []domain.ID, actorID domain.ID) error {
	tx, err := r.checklistTransaction(ctx, familyID, actorID)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	current, err := checklistIDs(ctx, tx, familyID)
	if err != nil {
		return err
	}
	if len(ids) != len(current) {
		return apperror.ErrInvalidArgument
	}
	remaining := make(map[domain.ID]bool, len(current))
	for _, id := range current {
		remaining[id] = true
	}
	for _, id := range ids {
		if !remaining[id] {
			return apperror.ErrInvalidArgument
		}
		delete(remaining, id)
	}
	if err := writeChecklistOrder(ctx, tx, ids); err != nil {
		return err
	}
	return normalizeError(tx.Commit(ctx))
}

func (r HousingRepository) ListHousingAnswers(ctx context.Context, familyID, actorID domain.ID) ([]domain.HousingAnswer, error) {
	allowed, err := r.hasFamilyAccess(ctx, actorID, familyID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperror.ErrForbidden
	}
	rows, err := r.pool.Query(ctx, `SELECT a.housing_id::text,a.criterion_id::text,a.state,a.rating,a.notes FROM housing_checklist_answers a JOIN housing_checklist_criteria c ON c.id=a.criterion_id WHERE a.family_id=$1::uuid AND c.deleted_at IS NULL`, string(familyID))
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()
	answers := []domain.HousingAnswer{}
	for rows.Next() {
		var a domain.HousingAnswer
		if err := rows.Scan(&a.HousingID, &a.CriterionID, &a.State, &a.Rating, &a.Notes); err != nil {
			return nil, normalizeError(err)
		}
		answers = append(answers, a)
	}
	return answers, normalizeError(rows.Err())
}

func (r HousingRepository) SaveHousingAnswer(ctx context.Context, a domain.HousingAnswer, actorID domain.ID) (domain.HousingAnswer, error) {
	if err := domain.ValidateHousingAnswer(a); err != nil {
		return domain.HousingAnswer{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.HousingAnswer{}, normalizeError(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var familyID string
	// Lock the active criterion until the answer is saved; retirement cannot race it.
	err = tx.QueryRow(ctx, `SELECT h.family_id::text FROM housing_options h JOIN housing_checklist_criteria c ON c.family_id=h.family_id JOIN family_members fm ON fm.family_id=h.family_id WHERE h.id=$1::uuid AND c.id=$2::uuid AND c.deleted_at IS NULL AND fm.user_id=$3::uuid AND fm.role IN ('owner','admin','member') FOR SHARE OF c, h, fm`, string(a.HousingID), string(a.CriterionID), string(actorID)).Scan(&familyID)
	if err != nil {
		return domain.HousingAnswer{}, normalizeError(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO housing_checklist_answers(housing_id,criterion_id,family_id,state,rating,notes) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6) ON CONFLICT(housing_id,criterion_id) DO UPDATE SET state=EXCLUDED.state,rating=EXCLUDED.rating,notes=EXCLUDED.notes,updated_at=now()`, string(a.HousingID), string(a.CriterionID), familyID, a.State, a.Rating, a.Notes)
	if err != nil {
		return domain.HousingAnswer{}, normalizeError(err)
	}
	return a, normalizeError(tx.Commit(ctx))
}

func (r HousingRepository) attachHousingEvaluations(ctx context.Context, options []domain.HousingOption, familyID, actorID domain.ID) error {
	if len(options) == 0 {
		return nil
	}
	criteria, err := r.ListHousingCriteria(ctx, familyID, actorID)
	if err != nil {
		return err
	}
	answers, err := r.ListHousingAnswers(ctx, familyID, actorID)
	if err != nil {
		return err
	}
	byHousing := make(map[domain.ID][]domain.HousingAnswer)
	for _, a := range answers {
		byHousing[a.HousingID] = append(byHousing[a.HousingID], a)
	}
	for i := range options {
		summary := domain.SummarizeHousingEvaluation(criteria, byHousing[options[i].ID])
		options[i].Evaluation = &summary
	}
	return nil
}
