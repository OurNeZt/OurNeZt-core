package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PersonRepository struct {
	Repository
}

func NewPersonRepository(pool *pgxpool.Pool) PersonRepository {
	return PersonRepository{Repository: New(pool)}
}

func (r PersonRepository) CreatePersonProfile(ctx context.Context, profile domain.PersonProfile, actorID domain.ID) (domain.PersonProfile, error) {
	if strings.TrimSpace(string(profile.FamilyID)) == "" || strings.TrimSpace(profile.Name) == "" {
		return domain.PersonProfile{}, apperror.ErrInvalidArgument
	}

	if err := r.assertFamilyWriter(ctx, actorID, profile.FamilyID); err != nil {
		return domain.PersonProfile{}, err
	}

	row := r.pool.QueryRow(ctx, `
		WITH created AS (
			INSERT INTO person_profiles (
				family_id, name, age, relationship_label, employment_status, gross_monthly_income_cents,
				expected_future_income_cents, expected_income_start_date, graduation_date, ord_date,
				cash_savings_cents, cpf_oa_cents, cpf_sa_cents, cpf_ma_cents, monthly_expenses_cents
			)
			VALUES (
				$1::uuid, $2, $3, $4, $5, $6,
				$7, $8::date, $9::date, $10::date,
				$11, $12, $13, $14, $15
			)
			RETURNING
				id, family_id, name, age, relationship_label, employment_status,
				gross_monthly_income_cents, expected_future_income_cents, expected_income_start_date,
				graduation_date, ord_date, cash_savings_cents, cpf_oa_cents, cpf_sa_cents,
				cpf_ma_cents, monthly_expenses_cents, created_at, updated_at
		), history_insert AS (
			INSERT INTO person_income_history (
				person_profile_id, family_id, person_name, gross_monthly_income_cents,
				expected_future_income_cents, recorded_at
			)
			SELECT
				id, family_id, name, gross_monthly_income_cents, expected_future_income_cents, updated_at
			FROM created
		)
		SELECT
			id::text, family_id::text, name, age, relationship_label, employment_status,
			gross_monthly_income_cents, expected_future_income_cents, expected_income_start_date::text,
			graduation_date::text, ord_date::text, cash_savings_cents, cpf_oa_cents, cpf_sa_cents,
			cpf_ma_cents, monthly_expenses_cents, created_at, updated_at
		FROM created
	`, string(profile.FamilyID), profile.Name, profile.Age, profile.RelationshipLabel, string(profile.EmploymentStatus),
		profile.GrossMonthlyIncomeCents, profile.ExpectedFutureIncomeCents,
		optionalDateString(profile.ExpectedIncomeStartDate), optionalDateString(profile.GraduationDate), optionalDateString(profile.ORDDate),
		profile.CashSavingsCents, profile.CPFOACents, profile.CPFSACents, profile.CPFMACents, profile.MonthlyExpensesCents)

	created, err := scanPersonRow(row)
	if err != nil {
		if isMissingIncomeHistoryRelationError(err) {
			legacyRow := r.pool.QueryRow(ctx, `
				INSERT INTO person_profiles (
					family_id, name, age, relationship_label, employment_status, gross_monthly_income_cents,
					expected_future_income_cents, expected_income_start_date, graduation_date, ord_date,
					cash_savings_cents, cpf_oa_cents, cpf_sa_cents, cpf_ma_cents, monthly_expenses_cents
				)
				VALUES (
					$1::uuid, $2, $3, $4, $5, $6,
					$7, $8::date, $9::date, $10::date,
					$11, $12, $13, $14, $15
				)
				RETURNING
					id::text, family_id::text, name, age, relationship_label, employment_status,
					gross_monthly_income_cents, expected_future_income_cents, expected_income_start_date::text,
					graduation_date::text, ord_date::text, cash_savings_cents, cpf_oa_cents, cpf_sa_cents,
					cpf_ma_cents, monthly_expenses_cents, created_at, updated_at
			`, string(profile.FamilyID), profile.Name, profile.Age, profile.RelationshipLabel, string(profile.EmploymentStatus),
				profile.GrossMonthlyIncomeCents, profile.ExpectedFutureIncomeCents,
				optionalDateString(profile.ExpectedIncomeStartDate), optionalDateString(profile.GraduationDate), optionalDateString(profile.ORDDate),
				profile.CashSavingsCents, profile.CPFOACents, profile.CPFSACents, profile.CPFMACents, profile.MonthlyExpensesCents)
			created, err = scanPersonRow(legacyRow)
			if err == nil {
				return created, nil
			}
		}
		return domain.PersonProfile{}, normalizeError(err)
	}
	return created, nil
}

func (r PersonRepository) GetPersonProfile(ctx context.Context, personID domain.ID, viewerID domain.ID) (domain.PersonProfile, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT
			p.id::text, p.family_id::text, p.name, p.age, p.relationship_label, p.employment_status,
			p.gross_monthly_income_cents, p.expected_future_income_cents, p.expected_income_start_date::text,
			p.graduation_date::text, p.ord_date::text, p.cash_savings_cents, p.cpf_oa_cents, p.cpf_sa_cents,
			p.cpf_ma_cents, p.monthly_expenses_cents, p.created_at, p.updated_at
		FROM person_profiles p
		JOIN family_members fm ON fm.family_id = p.family_id
		WHERE p.id = $1::uuid AND fm.user_id = $2::uuid
	`, string(personID), string(viewerID))

	profile, err := scanPersonRow(row)
	if err != nil {
		return domain.PersonProfile{}, normalizeError(err)
	}
	return profile, nil
}

func (r PersonRepository) ListPersonProfilesByFamily(ctx context.Context, familyID domain.ID, viewerID domain.ID) ([]domain.PersonProfile, error) {
	allowed, err := r.hasFamilyAccess(ctx, viewerID, familyID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperror.ErrForbidden
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text, family_id::text, name, age, relationship_label, employment_status,
			gross_monthly_income_cents, expected_future_income_cents, expected_income_start_date::text,
			graduation_date::text, ord_date::text, cash_savings_cents, cpf_oa_cents, cpf_sa_cents,
			cpf_ma_cents, monthly_expenses_cents, created_at, updated_at
		FROM person_profiles
		WHERE family_id = $1::uuid
		ORDER BY created_at DESC
	`, string(familyID))
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()

	profiles := make([]domain.PersonProfile, 0)
	for rows.Next() {
		profile, scanErr := scanPersonRow(rows)
		if scanErr != nil {
			return nil, normalizeError(scanErr)
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeError(err)
	}
	return profiles, nil
}

func (r PersonRepository) ListPersonIncomeHistoryByFamily(ctx context.Context, familyID domain.ID, viewerID domain.ID) ([]domain.PersonIncomeHistoryEntry, error) {
	allowed, err := r.hasFamilyAccess(ctx, viewerID, familyID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperror.ErrForbidden
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			h.person_profile_id::text,
			COALESCE(p.name, h.person_name),
			h.gross_monthly_income_cents,
			h.expected_future_income_cents,
			h.recorded_at
		FROM person_income_history h
		LEFT JOIN person_profiles p ON p.id = h.person_profile_id
		WHERE h.family_id = $1::uuid
		ORDER BY h.recorded_at ASC, h.person_profile_id ASC
	`, string(familyID))
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()

	entries := make([]domain.PersonIncomeHistoryEntry, 0)
	for rows.Next() {
		var entry domain.PersonIncomeHistoryEntry
		var personID string
		if scanErr := rows.Scan(
			&personID,
			&entry.PersonName,
			&entry.GrossMonthlyIncomeCents,
			&entry.ExpectedFutureIncomeCents,
			&entry.RecordedAt,
		); scanErr != nil {
			return nil, normalizeError(scanErr)
		}
		entry.PersonID = domain.ID(personID)
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeError(err)
	}
	return entries, nil
}

func (r PersonRepository) UpdatePersonProfile(ctx context.Context, profile domain.PersonProfile, actorID domain.ID) (domain.PersonProfile, error) {
	if strings.TrimSpace(string(profile.ID)) == "" || strings.TrimSpace(profile.Name) == "" {
		return domain.PersonProfile{}, apperror.ErrInvalidArgument
	}

	row := r.pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE person_profiles p
			SET
				name = $2,
				age = $3,
				relationship_label = $4,
				employment_status = $5,
				gross_monthly_income_cents = $6,
				expected_future_income_cents = $7,
				expected_income_start_date = $8::date,
				graduation_date = $9::date,
				ord_date = $10::date,
				cash_savings_cents = $11,
				cpf_oa_cents = $12,
				cpf_sa_cents = $13,
				cpf_ma_cents = $14,
				monthly_expenses_cents = $15,
				updated_at = now()
			FROM family_members fm
			WHERE p.id = $1::uuid
				AND fm.family_id = p.family_id
				AND fm.user_id = $16::uuid
				AND fm.role IN ('owner', 'admin', 'member')
			RETURNING
				p.id, p.family_id, p.name, p.age, p.relationship_label, p.employment_status,
				p.gross_monthly_income_cents, p.expected_future_income_cents, p.expected_income_start_date,
				p.graduation_date, p.ord_date, p.cash_savings_cents, p.cpf_oa_cents, p.cpf_sa_cents,
				p.cpf_ma_cents, p.monthly_expenses_cents, p.created_at, p.updated_at
		), history_insert AS (
			INSERT INTO person_income_history (
				person_profile_id, family_id, person_name, gross_monthly_income_cents,
				expected_future_income_cents, recorded_at
			)
			SELECT
				id, family_id, name, gross_monthly_income_cents, expected_future_income_cents, updated_at
			FROM updated
		)
		SELECT
			id::text, family_id::text, name, age, relationship_label, employment_status,
			gross_monthly_income_cents, expected_future_income_cents, expected_income_start_date::text,
			graduation_date::text, ord_date::text, cash_savings_cents, cpf_oa_cents, cpf_sa_cents,
			cpf_ma_cents, monthly_expenses_cents, created_at, updated_at
		FROM updated
	`, string(profile.ID), profile.Name, profile.Age, profile.RelationshipLabel, string(profile.EmploymentStatus),
		profile.GrossMonthlyIncomeCents, profile.ExpectedFutureIncomeCents,
		optionalDateString(profile.ExpectedIncomeStartDate), optionalDateString(profile.GraduationDate), optionalDateString(profile.ORDDate),
		profile.CashSavingsCents, profile.CPFOACents, profile.CPFSACents, profile.CPFMACents, profile.MonthlyExpensesCents,
		string(actorID))

	updated, err := scanPersonRow(row)
	if err != nil {
		if isMissingIncomeHistoryRelationError(err) {
			legacyRow := r.pool.QueryRow(ctx, `
				UPDATE person_profiles p
				SET
					name = $2,
					age = $3,
					relationship_label = $4,
					employment_status = $5,
					gross_monthly_income_cents = $6,
					expected_future_income_cents = $7,
					expected_income_start_date = $8::date,
					graduation_date = $9::date,
					ord_date = $10::date,
					cash_savings_cents = $11,
					cpf_oa_cents = $12,
					cpf_sa_cents = $13,
					cpf_ma_cents = $14,
					monthly_expenses_cents = $15,
					updated_at = now()
				FROM family_members fm
				WHERE p.id = $1::uuid
					AND fm.family_id = p.family_id
					AND fm.user_id = $16::uuid
					AND fm.role IN ('owner', 'admin', 'member')
				RETURNING
					p.id::text, p.family_id::text, p.name, p.age, p.relationship_label, p.employment_status,
					p.gross_monthly_income_cents, p.expected_future_income_cents, p.expected_income_start_date::text,
					p.graduation_date::text, p.ord_date::text, p.cash_savings_cents, p.cpf_oa_cents, p.cpf_sa_cents,
					p.cpf_ma_cents, p.monthly_expenses_cents, p.created_at, p.updated_at
			`, string(profile.ID), profile.Name, profile.Age, profile.RelationshipLabel, string(profile.EmploymentStatus),
				profile.GrossMonthlyIncomeCents, profile.ExpectedFutureIncomeCents,
				optionalDateString(profile.ExpectedIncomeStartDate), optionalDateString(profile.GraduationDate), optionalDateString(profile.ORDDate),
				profile.CashSavingsCents, profile.CPFOACents, profile.CPFSACents, profile.CPFMACents, profile.MonthlyExpensesCents,
				string(actorID))
			updated, err = scanPersonRow(legacyRow)
			if err == nil {
				return updated, nil
			}
		}
		return domain.PersonProfile{}, normalizeError(err)
	}
	return updated, nil
}

func (r PersonRepository) DeletePersonProfile(ctx context.Context, personID domain.ID, actorID domain.ID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM person_profiles p
		USING family_members fm
		WHERE p.id = $1::uuid
		  AND fm.family_id = p.family_id
		  AND fm.user_id = $2::uuid
		  AND fm.role IN ('owner', 'admin', 'member')
	`, string(personID), string(actorID))
	if err != nil {
		return normalizeError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func scanPersonRow(scanner interface{ Scan(dest ...any) error }) (domain.PersonProfile, error) {
	var (
		profile           domain.PersonProfile
		id                string
		familyID          string
		employmentStatus  string
		age               int32
		expectedStartDate sql.NullString
		graduationDate    sql.NullString
		ordDate           sql.NullString
	)

	err := scanner.Scan(
		&id,
		&familyID,
		&profile.Name,
		&age,
		&profile.RelationshipLabel,
		&employmentStatus,
		&profile.GrossMonthlyIncomeCents,
		&profile.ExpectedFutureIncomeCents,
		&expectedStartDate,
		&graduationDate,
		&ordDate,
		&profile.CashSavingsCents,
		&profile.CPFOACents,
		&profile.CPFSACents,
		&profile.CPFMACents,
		&profile.MonthlyExpensesCents,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return domain.PersonProfile{}, err
	}

	profile.ID = domain.ID(id)
	profile.FamilyID = domain.ID(familyID)
	profile.Age = int(age)
	profile.EmploymentStatus = domain.EmploymentStatus(employmentStatus)
	profile.ExpectedIncomeStartDate = parseOptionalDate(expectedStartDate)
	profile.GraduationDate = parseOptionalDate(graduationDate)
	profile.ORDDate = parseOptionalDate(ordDate)
	return profile, nil
}

func isMissingIncomeHistoryRelationError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42P01" && strings.Contains(strings.ToLower(pgErr.Message), "person_income_history")
	}
	lowerErr := strings.ToLower(err.Error())
	return strings.Contains(lowerErr, "person_income_history") && strings.Contains(lowerErr, "does not exist")
}
