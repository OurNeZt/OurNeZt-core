package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FamilyRepository struct {
	Repository
}

func NewFamilyRepository(pool *pgxpool.Pool) FamilyRepository {
	return FamilyRepository{Repository: New(pool)}
}

func (r FamilyRepository) CreateFamily(ctx context.Context, family domain.Family, ownerID domain.ID) (domain.Family, error) {
	if strings.TrimSpace(family.Name) == "" || strings.TrimSpace(string(family.Type)) == "" || strings.TrimSpace(string(ownerID)) == "" {
		return domain.Family{}, apperror.ErrInvalidArgument
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Family{}, normalizeError(err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO families (name, family_type)
		VALUES ($1, $2)
		RETURNING id::text, name, family_type, created_at, updated_at
	`, family.Name, string(family.Type))

	created, err := scanFamilyRow(row)
	if err != nil {
		return domain.Family{}, normalizeError(err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO family_members (family_id, user_id, role)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (family_id, user_id)
		DO UPDATE SET role = EXCLUDED.role
	`, string(created.ID), string(ownerID), string(domain.FamilyRoleOwner))
	if err != nil {
		return domain.Family{}, normalizeError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Family{}, normalizeError(err)
	}
	return created, nil
}

func (r FamilyRepository) GetFamily(ctx context.Context, familyID domain.ID, viewerID domain.ID) (domain.Family, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT f.id::text, f.name, f.family_type, f.created_at, f.updated_at
		FROM families f
		JOIN family_members fm ON fm.family_id = f.id
		WHERE f.id = $1::uuid AND fm.user_id = $2::uuid
	`, string(familyID), string(viewerID))

	family, err := scanFamilyRow(row)
	if err != nil {
		return domain.Family{}, normalizeError(err)
	}
	return family, nil
}

func (r FamilyRepository) ListUserFamilies(ctx context.Context, userID domain.ID) ([]domain.Family, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT f.id::text, f.name, f.family_type, f.created_at, f.updated_at
		FROM families f
		JOIN family_members fm ON fm.family_id = f.id
		WHERE fm.user_id = $1::uuid
		ORDER BY f.created_at DESC
	`, string(userID))
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()

	families := make([]domain.Family, 0)
	for rows.Next() {
		family, scanErr := scanFamilyRow(rows)
		if scanErr != nil {
			return nil, normalizeError(scanErr)
		}
		families = append(families, family)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeError(err)
	}
	return families, nil
}

func (r FamilyRepository) JoinFamilyByCode(ctx context.Context, userID domain.ID, codeHash string) (domain.Family, error) {
	if strings.TrimSpace(string(userID)) == "" || strings.TrimSpace(codeHash) == "" {
		return domain.Family{}, apperror.ErrInvalidArgument
	}

	row := r.pool.QueryRow(ctx, `
		SELECT f.id::text, f.name, f.family_type, f.created_at, f.updated_at
		FROM family_invites fi
		JOIN families f ON f.id = fi.family_id
		WHERE fi.code_hash = $1
		  AND fi.revoked_at IS NULL
		  AND (fi.expires_at IS NULL OR fi.expires_at > now())
		ORDER BY fi.created_at DESC
		LIMIT 1
	`, codeHash)

	family, err := scanFamilyRow(row)
	if err != nil {
		return domain.Family{}, normalizeError(err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO family_members (family_id, user_id, role)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (family_id, user_id)
		DO NOTHING
	`, string(family.ID), string(userID), string(domain.FamilyRoleMember))
	if err != nil {
		return domain.Family{}, normalizeError(err)
	}

	return family, nil
}

func (r FamilyRepository) GenerateFamilyJoinCode(ctx context.Context, actorID domain.ID, familyID domain.ID, codeHash string, expiresAt *time.Time) error {
	if strings.TrimSpace(codeHash) == "" {
		return apperror.ErrInvalidArgument
	}
	if err := r.assertFamilyManager(ctx, actorID, familyID); err != nil {
		return err
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO family_invites (family_id, code_hash, expires_at, created_by)
		VALUES ($1::uuid, $2, $3, $4::uuid)
	`, string(familyID), codeHash, expiresAt, string(actorID))
	return normalizeError(err)
}

func (r FamilyRepository) ListFamilyMembers(ctx context.Context, actorID domain.ID, familyID domain.ID) ([]repository.FamilyMember, error) {
	allowed, err := r.hasFamilyAccess(ctx, actorID, familyID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperror.ErrForbidden
	}

	rows, err := r.pool.Query(ctx, `
		SELECT fm.family_id::text, fm.user_id::text, COALESCE(u.display_name, ''), fm.role
		FROM family_members fm
		JOIN users u ON u.id = fm.user_id
		WHERE fm.family_id = $1::uuid
		ORDER BY fm.joined_at ASC
	`, string(familyID))
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()

	members := make([]repository.FamilyMember, 0)
	for rows.Next() {
		var (
			member   repository.FamilyMember
			familyID string
			userID   string
			role     string
		)
		if err := rows.Scan(&familyID, &userID, &member.DisplayName, &role); err != nil {
			return nil, normalizeError(err)
		}
		member.FamilyID = domain.ID(familyID)
		member.UserID = domain.ID(userID)
		member.Role = domain.FamilyRole(role)
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeError(err)
	}
	return members, nil
}

func (r FamilyRepository) assertFamilyManager(ctx context.Context, actorID domain.ID, familyID domain.ID) error {
	var hasManagerRole bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM family_members
			WHERE family_id = $1::uuid
			  AND user_id = $2::uuid
			  AND role IN ('owner', 'admin')
		)
	`, string(familyID), string(actorID)).Scan(&hasManagerRole)
	if err != nil {
		return normalizeError(err)
	}
	if !hasManagerRole {
		return apperror.ErrForbidden
	}
	return nil
}

func scanFamilyRow(scanner interface{ Scan(dest ...any) error }) (domain.Family, error) {
	var (
		family domain.Family
		id     string
		kind   string
	)
	err := scanner.Scan(&id, &family.Name, &kind, &family.CreatedAt, &family.UpdatedAt)
	if err != nil {
		return domain.Family{}, err
	}
	family.ID = domain.ID(id)
	family.Type = domain.FamilyType(kind)
	return family, nil
}

