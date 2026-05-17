package postgres

import (
	"context"
	"database/sql"
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

	row := r.pool.QueryRow(ctx, `
		INSERT INTO housing_options (
			family_id, name, housing_type, location, unit_type, purchase_price_cents, grant_amount_cents,
			loan_type, loan_amount_cents, interest_rate_bps, loan_tenure_months, downpayment_percent_bps,
			renovation_budget_cents, furniture_budget_cents, legal_fees_cents, buyer_stamp_duty_cents,
			monthly_maintenance_cents, expected_key_collection_date
		)
		VALUES (
			$1::uuid, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18::date
		)
		RETURNING
			id::text, family_id::text, name, housing_type, location, unit_type, purchase_price_cents,
			grant_amount_cents, loan_type, loan_amount_cents, interest_rate_bps, loan_tenure_months,
			downpayment_percent_bps, renovation_budget_cents, furniture_budget_cents, legal_fees_cents,
			buyer_stamp_duty_cents, monthly_maintenance_cents, expected_key_collection_date::text, created_at, updated_at
	`, string(option.FamilyID), option.Name, string(option.Type), option.Location, option.UnitType, option.PurchasePriceCents,
		option.GrantAmountCents, string(option.LoanType), option.LoanAmountCents, option.InterestRateBps, option.LoanTenureMonths,
		option.DownpaymentPercentBps, option.RenovationBudgetCents, option.FurnitureBudgetCents, option.LegalFeesCents,
		option.BuyerStampDutyCents, option.MonthlyMaintenanceCents, optionalDateString(option.ExpectedKeyCollectionDate))

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
			h.buyer_stamp_duty_cents, h.monthly_maintenance_cents, h.expected_key_collection_date::text, h.created_at, h.updated_at
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
			buyer_stamp_duty_cents, monthly_maintenance_cents, expected_key_collection_date::text, created_at, updated_at
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
			expected_key_collection_date = $18::date,
			updated_at = now()
		FROM family_members fm
		WHERE h.id = $1::uuid
			AND fm.family_id = h.family_id
			AND fm.user_id = $19::uuid
			AND fm.role IN ('owner', 'admin', 'member')
		RETURNING
			h.id::text, h.family_id::text, h.name, h.housing_type, h.location, h.unit_type, h.purchase_price_cents,
			h.grant_amount_cents, h.loan_type, h.loan_amount_cents, h.interest_rate_bps, h.loan_tenure_months,
			h.downpayment_percent_bps, h.renovation_budget_cents, h.furniture_budget_cents, h.legal_fees_cents,
			h.buyer_stamp_duty_cents, h.monthly_maintenance_cents, h.expected_key_collection_date::text, h.created_at, h.updated_at
	`, string(option.ID), option.Name, string(option.Type), option.Location, option.UnitType, option.PurchasePriceCents,
		option.GrantAmountCents, string(option.LoanType), option.LoanAmountCents, option.InterestRateBps,
		option.LoanTenureMonths, option.DownpaymentPercentBps, option.RenovationBudgetCents, option.FurnitureBudgetCents,
		option.LegalFeesCents, option.BuyerStampDutyCents, option.MonthlyMaintenanceCents, optionalDateString(option.ExpectedKeyCollectionDate),
		string(actorID))

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

func scanHousingRow(scanner interface{ Scan(dest ...any) error }) (domain.HousingOption, error) {
	var (
		option           domain.HousingOption
		id               string
		familyID         string
		housingType      string
		loanType         string
		expectedKeyDate  sql.NullString
		interestRateBps  int32
		loanTenureMonths int32
		downpaymentBps   int32
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
		&expectedKeyDate,
		&option.CreatedAt,
		&option.UpdatedAt,
	)
	if err != nil {
		return domain.HousingOption{}, err
	}

	option.ID = domain.ID(id)
	option.FamilyID = domain.ID(familyID)
	option.Type = domain.HousingType(housingType)
	option.LoanType = domain.LoanType(loanType)
	option.InterestRateBps = int64(interestRateBps)
	option.LoanTenureMonths = int(loanTenureMonths)
	option.DownpaymentPercentBps = int64(downpaymentBps)
	option.ExpectedKeyCollectionDate = parseOptionalDate(expectedKeyDate)
	return option, nil
}
