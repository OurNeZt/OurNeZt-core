package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	Repository
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return UserRepository{Repository: New(pool)}
}

func (r UserRepository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, display_name, role, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, email, display_name, role, password_hash, disabled_at, created_at, updated_at
	`, user.Email, user.DisplayName, string(user.Role), user.PasswordHash)

	created, err := scanUserRow(row)
	if err != nil {
		return domain.User{}, normalizeError(err)
	}
	return created, nil
}

func (r UserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, email, display_name, role, password_hash, disabled_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`, strings.ToLower(strings.TrimSpace(email)))

	user, err := scanUserRow(row)
	if err != nil {
		return domain.User{}, normalizeError(err)
	}
	return user, nil
}

func (r UserRepository) DisableUser(ctx context.Context, userID domain.ID) error {
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET disabled_at = now(), updated_at = now()
		WHERE id = $1::uuid
	`, string(userID))
	if err != nil {
		return normalizeError(err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r UserRepository) CreateSession(ctx context.Context, userID domain.ID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1::uuid, $2, $3)
	`, string(userID), tokenHash, expiresAt)
	return normalizeError(err)
}

func (r UserRepository) GetUserBySessionTokenHash(ctx context.Context, tokenHash string, now time.Time) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.display_name, u.role, u.password_hash, u.disabled_at, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > $2
		ORDER BY s.created_at DESC
		LIMIT 1
	`, tokenHash, now)

	user, err := scanUserRow(row)
	if err != nil {
		return domain.User{}, normalizeError(err)
	}
	return user, nil
}

func (r UserRepository) RevokeSessionsByUserID(ctx context.Context, userID domain.ID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = now()
		WHERE user_id = $1::uuid AND revoked_at IS NULL
	`, string(userID))
	return normalizeError(err)
}

func scanUserRow(scanner interface{ Scan(dest ...any) error }) (domain.User, error) {
	var (
		user       domain.User
		id         string
		role       string
		disabledAt sql.NullTime
	)

	err := scanner.Scan(
		&id,
		&user.Email,
		&user.DisplayName,
		&role,
		&user.PasswordHash,
		&disabledAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}

	user.ID = domain.ID(id)
	user.Role = domain.UserRole(role)
	if disabledAt.Valid {
		t := disabledAt.Time
		user.DisabledAt = &t
	}
	return user, nil
}
