package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HousingRepository struct {
	Repository
}

func NewHousingRepository(pool *pgxpool.Pool) HousingRepository {
	return HousingRepository{Repository: New(pool)}
}

func (r HousingRepository) CreateHousingOption(ctx context.Context, option domain.HousingOption, actorID domain.ID) (domain.HousingOption, error) {
	if strings.TrimSpace(string(option.FamilyID)) == "" || strings.TrimSpace(option.Name) == "" {
		return domain.HousingOption{}, apperror.ErrInvalidArgument
	}

	if err := r.assertFamilyWriter(ctx, actorID, option.FamilyID); err != nil {
		return domain.HousingOption{}, err
	}
	if err := r.validateHousingGroupAssignment(ctx, option.FamilyID, option.GroupID); err != nil {
		return domain.HousingOption{}, err
	}

	diaOverridesJSON, marshalErr := marshalDIAIncomeOverrides(option.DIAIncomeOverrides)
	if marshalErr != nil {
		return domain.HousingOption{}, apperror.ErrInvalidArgument
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO housing_options (
			family_id, name, housing_type, location, unit_type, purchase_price_cents, grant_amount_cents,
			loan_type, loan_amount_cents, interest_rate_bps, loan_tenure_months, downpayment_percent_bps,
			renovation_budget_cents, furniture_budget_cents, legal_fees_cents, buyer_stamp_duty_cents,
			monthly_maintenance_cents, dia_income_overrides, expected_key_collection_date, housing_group_id,
			visible_on_dashboard
		)
		VALUES (
			$1::uuid, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18::jsonb, $19::date, NULLIF($20, '')::uuid, $21
		)
		RETURNING
			id::text, family_id::text, name, housing_type, location, unit_type, purchase_price_cents,
			grant_amount_cents, loan_type, loan_amount_cents, interest_rate_bps, loan_tenure_months,
			downpayment_percent_bps, renovation_budget_cents, furniture_budget_cents, legal_fees_cents,
			buyer_stamp_duty_cents, monthly_maintenance_cents, dia_income_overrides, expected_key_collection_date::text,
			COALESCE(housing_group_id::text, ''), visible_on_dashboard, created_at, updated_at
	`, string(option.FamilyID), option.Name, string(option.Type), option.Location, option.UnitType, option.PurchasePriceCents,
		option.GrantAmountCents, string(option.LoanType), option.LoanAmountCents, option.InterestRateBps, option.LoanTenureMonths,
		option.DownpaymentPercentBps, option.RenovationBudgetCents, option.FurnitureBudgetCents, option.LegalFeesCents,
		option.BuyerStampDutyCents, option.MonthlyMaintenanceCents, diaOverridesJSON, optionalDateString(option.ExpectedKeyCollectionDate),
		string(option.GroupID), option.VisibleOnDashboard)

	created, err := scanHousingRow(row)
	if err != nil {
		return domain.HousingOption{}, normalizeError(err)
	}
	return created, nil
}

func (r HousingRepository) GetHousingOption(ctx context.Context, housingID domain.ID, viewerID domain.ID) (domain.HousingOption, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT
			h.id::text, h.family_id::text, h.name, h.housing_type, h.location, h.unit_type, h.purchase_price_cents,
			h.grant_amount_cents, h.loan_type, h.loan_amount_cents, h.interest_rate_bps, h.loan_tenure_months,
			h.downpayment_percent_bps, h.renovation_budget_cents, h.furniture_budget_cents, h.legal_fees_cents,
			h.buyer_stamp_duty_cents, h.monthly_maintenance_cents, h.dia_income_overrides, h.expected_key_collection_date::text,
			COALESCE(h.housing_group_id::text, ''), h.visible_on_dashboard, h.created_at, h.updated_at
		FROM housing_options h
		JOIN family_members fm ON fm.family_id = h.family_id
		WHERE h.id = $1::uuid AND fm.user_id = $2::uuid
	`, string(housingID), string(viewerID))

	option, err := scanHousingRow(row)
	if err != nil {
		return domain.HousingOption{}, normalizeError(err)
	}
	return option, nil
}

func (r HousingRepository) ListHousingOptions(ctx context.Context, familyID domain.ID, viewerID domain.ID) ([]domain.HousingOption, error) {
	allowed, err := r.hasFamilyAccess(ctx, viewerID, familyID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperror.ErrForbidden
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text, family_id::text, name, housing_type, location, unit_type, purchase_price_cents,
			grant_amount_cents, loan_type, loan_amount_cents, interest_rate_bps, loan_tenure_months,
			downpayment_percent_bps, renovation_budget_cents, furniture_budget_cents, legal_fees_cents,
			buyer_stamp_duty_cents, monthly_maintenance_cents, dia_income_overrides, expected_key_collection_date::text,
			COALESCE(housing_group_id::text, ''), visible_on_dashboard, created_at, updated_at
		FROM housing_options
		WHERE family_id = $1::uuid
		ORDER BY created_at DESC
	`, string(familyID))
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()

	options := make([]domain.HousingOption, 0)
	for rows.Next() {
		option, scanErr := scanHousingRow(rows)
		if scanErr != nil {
			return nil, normalizeError(scanErr)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeError(err)
	}
	return options, nil
}

func (r HousingRepository) UpdateHousingOption(ctx context.Context, option domain.HousingOption, actorID domain.ID) (domain.HousingOption, error) {
	if strings.TrimSpace(string(option.ID)) == "" || strings.TrimSpace(option.Name) == "" {
		return domain.HousingOption{}, apperror.ErrInvalidArgument
	}
	if strings.TrimSpace(string(option.FamilyID)) == "" {
		return domain.HousingOption{}, apperror.ErrInvalidArgument
	}
	if err := r.validateHousingGroupAssignment(ctx, option.FamilyID, option.GroupID); err != nil {
		return domain.HousingOption{}, err
	}

	diaOverridesJSON, marshalErr := marshalDIAIncomeOverrides(option.DIAIncomeOverrides)
	if marshalErr != nil {
		return domain.HousingOption{}, apperror.ErrInvalidArgument
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE housing_options h
		SET
			name = $2,
			housing_type = $3,
			location = $4,
			unit_type = $5,
			purchase_price_cents = $6,
			grant_amount_cents = $7,
			loan_type = $8,
			loan_amount_cents = $9,
			interest_rate_bps = $10,
			loan_tenure_months = $11,
			downpayment_percent_bps = $12,
			renovation_budget_cents = $13,
			furniture_budget_cents = $14,
			legal_fees_cents = $15,
			buyer_stamp_duty_cents = $16,
			monthly_maintenance_cents = $17,
			dia_income_overrides = $18::jsonb,
			expected_key_collection_date = $19::date,
			housing_group_id = NULLIF($20, '')::uuid,
			visible_on_dashboard = $21,
			updated_at = now()
		FROM family_members fm
		WHERE h.id = $1::uuid
			AND fm.family_id = h.family_id
			AND fm.user_id = $22::uuid
			AND fm.role IN ('owner', 'admin', 'member')
		RETURNING
			h.id::text, h.family_id::text, h.name, h.housing_type, h.location, h.unit_type, h.purchase_price_cents,
			h.grant_amount_cents, h.loan_type, h.loan_amount_cents, h.interest_rate_bps, h.loan_tenure_months,
			h.downpayment_percent_bps, h.renovation_budget_cents, h.furniture_budget_cents, h.legal_fees_cents,
			h.buyer_stamp_duty_cents, h.monthly_maintenance_cents, h.dia_income_overrides, h.expected_key_collection_date::text,
			COALESCE(h.housing_group_id::text, ''), h.visible_on_dashboard, h.created_at, h.updated_at
	`, string(option.ID), option.Name, string(option.Type), option.Location, option.UnitType, option.PurchasePriceCents,
		option.GrantAmountCents, string(option.LoanType), option.LoanAmountCents, option.InterestRateBps,
		option.LoanTenureMonths, option.DownpaymentPercentBps, option.RenovationBudgetCents, option.FurnitureBudgetCents,
		option.LegalFeesCents, option.BuyerStampDutyCents, option.MonthlyMaintenanceCents, diaOverridesJSON, optionalDateString(option.ExpectedKeyCollectionDate),
		string(option.GroupID), option.VisibleOnDashboard, string(actorID))

	updated, err := scanHousingRow(row)
	if err != nil {
		return domain.HousingOption{}, normalizeError(err)
	}
	return updated, nil
}

func (r HousingRepository) DeleteHousingOption(ctx context.Context, housingID domain.ID, actorID domain.ID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM housing_options h
		USING family_members fm
		WHERE h.id = $1::uuid
		  AND fm.family_id = h.family_id
		  AND fm.user_id = $2::uuid
		  AND fm.role IN ('owner', 'admin', 'member')
	`, string(housingID), string(actorID))
	if err != nil {
		return normalizeError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r HousingRepository) CreateHousingGroup(ctx context.Context, group domain.HousingGroup, actorID domain.ID) (domain.HousingGroup, error) {
	if strings.TrimSpace(string(group.FamilyID)) == "" || strings.TrimSpace(group.Name) == "" {
		return domain.HousingGroup{}, apperror.ErrInvalidArgument
	}

	if err := r.assertFamilyWriter(ctx, actorID, group.FamilyID); err != nil {
		return domain.HousingGroup{}, err
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO housing_groups (family_id, name)
		VALUES ($1::uuid, $2)
		RETURNING id::text, family_id::text, name, created_at, updated_at
	`, string(group.FamilyID), group.Name)

	created, err := scanHousingGroupRow(row)
	if err != nil {
		return domain.HousingGroup{}, normalizeError(err)
	}
	return created, nil
}

func (r HousingRepository) ListHousingGroups(ctx context.Context, familyID domain.ID, viewerID domain.ID) ([]domain.HousingGroup, error) {
	allowed, err := r.hasFamilyAccess(ctx, viewerID, familyID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperror.ErrForbidden
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id::text, family_id::text, name, created_at, updated_at
		FROM housing_groups
		WHERE family_id = $1::uuid
		ORDER BY created_at DESC
	`, string(familyID))
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()

	groups := make([]domain.HousingGroup, 0)
	for rows.Next() {
		group, scanErr := scanHousingGroupRow(rows)
		if scanErr != nil {
			return nil, normalizeError(scanErr)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeError(err)
	}
	return groups, nil
}

func (r HousingRepository) UpdateHousingGroup(ctx context.Context, group domain.HousingGroup, actorID domain.ID) (domain.HousingGroup, error) {
	if strings.TrimSpace(string(group.ID)) == "" || strings.TrimSpace(group.Name) == "" {
		return domain.HousingGroup{}, apperror.ErrInvalidArgument
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE housing_groups hg
		SET
			name = $2,
			updated_at = now()
		FROM family_members fm
		WHERE hg.id = $1::uuid
			AND fm.family_id = hg.family_id
			AND fm.user_id = $3::uuid
			AND fm.role IN ('owner', 'admin', 'member')
		RETURNING hg.id::text, hg.family_id::text, hg.name, hg.created_at, hg.updated_at
	`, string(group.ID), group.Name, string(actorID))

	updated, err := scanHousingGroupRow(row)
	if err != nil {
		return domain.HousingGroup{}, normalizeError(err)
	}
	return updated, nil
}

func (r HousingRepository) DeleteHousingGroup(ctx context.Context, groupID domain.ID, actorID domain.ID) error {
	familyID, err := r.getHousingGroupFamilyID(ctx, groupID)
	if err != nil {
		return err
	}
	if err := r.assertFamilyWriter(ctx, actorID, familyID); err != nil {
		return err
	}

	tag, err := r.pool.Exec(ctx, `
		DELETE FROM housing_groups
		WHERE id = $1::uuid
	`, string(groupID))
	if err != nil {
		return normalizeError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r HousingRepository) AssignHousingOptionGroup(ctx context.Context, housingID domain.ID, groupID domain.ID, actorID domain.ID) (domain.HousingOption, error) {
	groupIDText := strings.TrimSpace(string(groupID))
	var groupFamilyID interface{}
	if groupIDText != "" {
		familyID, err := r.getHousingGroupFamilyID(ctx, domain.ID(groupIDText))
		if err != nil {
			return domain.HousingOption{}, err
		}
		groupFamilyID = string(familyID)
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE housing_options h
		SET
			housing_group_id = NULLIF($2, '')::uuid,
			updated_at = now()
		FROM family_members fm
		WHERE h.id = $1::uuid
			AND fm.family_id = h.family_id
			AND fm.user_id = $3::uuid
			AND fm.role IN ('owner', 'admin', 'member')
			AND ($4::uuid IS NULL OR h.family_id = $4::uuid)
		RETURNING
			h.id::text, h.family_id::text, h.name, h.housing_type, h.location, h.unit_type, h.purchase_price_cents,
			h.grant_amount_cents, h.loan_type, h.loan_amount_cents, h.interest_rate_bps, h.loan_tenure_months,
			h.downpayment_percent_bps, h.renovation_budget_cents, h.furniture_budget_cents, h.legal_fees_cents,
			h.buyer_stamp_duty_cents, h.monthly_maintenance_cents, h.dia_income_overrides, h.expected_key_collection_date::text,
			COALESCE(h.housing_group_id::text, ''), h.visible_on_dashboard, h.created_at, h.updated_at
	`, string(housingID), groupIDText, string(actorID), groupFamilyID)

	updated, err := scanHousingRow(row)
	if err != nil {
		return domain.HousingOption{}, normalizeError(err)
	}
	return updated, nil
}

func (r HousingRepository) UpdateHousingOptionVisibility(ctx context.Context, housingID domain.ID, visible bool, actorID domain.ID) (domain.HousingOption, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE housing_options h
		SET
			visible_on_dashboard = $2,
			updated_at = now()
		FROM family_members fm
		WHERE h.id = $1::uuid
			AND fm.family_id = h.family_id
			AND fm.user_id = $3::uuid
			AND fm.role IN ('owner', 'admin', 'member')
		RETURNING
			h.id::text, h.family_id::text, h.name, h.housing_type, h.location, h.unit_type, h.purchase_price_cents,
			h.grant_amount_cents, h.loan_type, h.loan_amount_cents, h.interest_rate_bps, h.loan_tenure_months,
			h.downpayment_percent_bps, h.renovation_budget_cents, h.furniture_budget_cents, h.legal_fees_cents,
			h.buyer_stamp_duty_cents, h.monthly_maintenance_cents, h.dia_income_overrides, h.expected_key_collection_date::text,
			COALESCE(h.housing_group_id::text, ''), h.visible_on_dashboard, h.created_at, h.updated_at
	`, string(housingID), visible, string(actorID))

	updated, err := scanHousingRow(row)
	if err != nil {
		return domain.HousingOption{}, normalizeError(err)
	}
	return updated, nil
}

func (r HousingRepository) BulkUpdateHousingGroupVisibility(ctx context.Context, groupID domain.ID, visible bool, actorID domain.ID) ([]domain.HousingOption, error) {
	familyID, err := r.getHousingGroupFamilyID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if err := r.assertFamilyWriter(ctx, actorID, familyID); err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		UPDATE housing_options
		SET
			visible_on_dashboard = $2,
			updated_at = now()
		WHERE housing_group_id = $1::uuid
		RETURNING
			id::text, family_id::text, name, housing_type, location, unit_type, purchase_price_cents,
			grant_amount_cents, loan_type, loan_amount_cents, interest_rate_bps, loan_tenure_months,
			downpayment_percent_bps, renovation_budget_cents, furniture_budget_cents, legal_fees_cents,
			buyer_stamp_duty_cents, monthly_maintenance_cents, dia_income_overrides, expected_key_collection_date::text,
			COALESCE(housing_group_id::text, ''), visible_on_dashboard, created_at, updated_at
	`, string(groupID), visible)
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()

	options := make([]domain.HousingOption, 0)
	for rows.Next() {
		option, scanErr := scanHousingRow(rows)
		if scanErr != nil {
			return nil, normalizeError(scanErr)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeError(err)
	}
	return options, nil
}

func scanHousingRow(scanner interface{ Scan(dest ...any) error }) (domain.HousingOption, error) {
	var (
		option             domain.HousingOption
		id                 string
		familyID           string
		groupID            string
		housingType        string
		loanType           string
		expectedKeyDate    sql.NullString
		interestRateBps    int32
		loanTenureMonths   int32
		downpaymentBps     int32
		diaOverridesRaw    []byte
		visibleOnDashboard bool
	)

	err := scanner.Scan(
		&id,
		&familyID,
		&option.Name,
		&housingType,
		&option.Location,
		&option.UnitType,
		&option.PurchasePriceCents,
		&option.GrantAmountCents,
		&loanType,
		&option.LoanAmountCents,
		&interestRateBps,
		&loanTenureMonths,
		&downpaymentBps,
		&option.RenovationBudgetCents,
		&option.FurnitureBudgetCents,
		&option.LegalFeesCents,
		&option.BuyerStampDutyCents,
		&option.MonthlyMaintenanceCents,
		&diaOverridesRaw,
		&expectedKeyDate,
		&groupID,
		&visibleOnDashboard,
		&option.CreatedAt,
		&option.UpdatedAt,
	)
	if err != nil {
		return domain.HousingOption{}, err
	}

	option.ID = domain.ID(id)
	option.FamilyID = domain.ID(familyID)
	option.GroupID = domain.ID(strings.TrimSpace(groupID))
	option.Type = domain.HousingType(housingType)
	option.LoanType = domain.LoanType(loanType)
	option.InterestRateBps = int64(interestRateBps)
	option.LoanTenureMonths = int(loanTenureMonths)
	option.DownpaymentPercentBps = int64(downpaymentBps)
	option.VisibleOnDashboard = visibleOnDashboard
	option.ExpectedKeyCollectionDate = parseOptionalDate(expectedKeyDate)
	option.DIAIncomeOverrides = unmarshalDIAIncomeOverrides(diaOverridesRaw)
	return option, nil
}

func scanHousingGroupRow(scanner interface{ Scan(dest ...any) error }) (domain.HousingGroup, error) {
	var group domain.HousingGroup
	var id string
	var familyID string

	err := scanner.Scan(
		&id,
		&familyID,
		&group.Name,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if err != nil {
		return domain.HousingGroup{}, err
	}

	group.ID = domain.ID(id)
	group.FamilyID = domain.ID(familyID)
	return group, nil
}

func (r HousingRepository) validateHousingGroupAssignment(ctx context.Context, familyID domain.ID, groupID domain.ID) error {
	groupIDText := strings.TrimSpace(string(groupID))
	if groupIDText == "" {
		return nil
	}

	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM housing_groups
			WHERE id = $1::uuid AND family_id = $2::uuid
		)
	`, groupIDText, string(familyID)).Scan(&exists)
	if err != nil {
		return normalizeError(err)
	}
	if !exists {
		return apperror.ErrInvalidArgument
	}
	return nil
}

func (r HousingRepository) getHousingGroupFamilyID(ctx context.Context, groupID domain.ID) (domain.ID, error) {
	if strings.TrimSpace(string(groupID)) == "" {
		return "", apperror.ErrInvalidArgument
	}

	var familyID string
	err := r.pool.QueryRow(ctx, `
		SELECT family_id::text
		FROM housing_groups
		WHERE id = $1::uuid
	`, string(groupID)).Scan(&familyID)
	if err != nil {
		return "", normalizeError(err)
	}
	return domain.ID(familyID), nil
}

func marshalDIAIncomeOverrides(overrides []domain.HousingDIAIncomeOverride) ([]byte, error) {
	clean := make([]domain.HousingDIAIncomeOverride, 0, len(overrides))
	for _, override := range overrides {
		personID := strings.TrimSpace(string(override.PersonID))
		if personID == "" {
			continue
		}
		projected := override.ProjectedIncomeCents
		if projected < 0 {
			projected = 0
		}
		clean = append(clean, domain.HousingDIAIncomeOverride{
			PersonID:             domain.ID(personID),
			ProjectedIncomeCents: projected,
		})
	}
	if len(clean) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(clean)
}

func unmarshalDIAIncomeOverrides(raw []byte) []domain.HousingDIAIncomeOverride {
	if len(raw) == 0 {
		return nil
	}
	var parsed []domain.HousingDIAIncomeOverride
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil
	}
	clean := make([]domain.HousingDIAIncomeOverride, 0, len(parsed))
	for _, override := range parsed {
		personID := strings.TrimSpace(string(override.PersonID))
		if personID == "" {
			continue
		}
		projected := override.ProjectedIncomeCents
		if projected < 0 {
			projected = 0
		}
		clean = append(clean, domain.HousingDIAIncomeOverride{
			PersonID:             domain.ID(personID),
			ProjectedIncomeCents: projected,
		})
	}
	return clean
}
