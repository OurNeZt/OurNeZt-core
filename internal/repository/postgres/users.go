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
		INSERT INTO users (email, display_name, role, password_hash, must_change_password)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, email, display_name, role, password_hash, must_change_password, disabled_at, created_at, updated_at
	`, user.Email, user.DisplayName, string(user.Role), user.PasswordHash, user.MustChangePassword)

	created, err := scanUserRow(row)
	if err != nil {
		return domain.User{}, normalizeError(err)
	}
	return created, nil
}

func (r UserRepository) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, email, display_name, role, password_hash, must_change_password, disabled_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC, email ASC
	`)
	if err != nil {
		return nil, normalizeError(err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		user, scanErr := scanUserRow(rows)
		if scanErr != nil {
			return nil, normalizeError(scanErr)
		}
		users = append(users, user)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, normalizeError(rowsErr)
	}

	return users, nil
}

func (r UserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, email, display_name, role, password_hash, must_change_password, disabled_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`, strings.ToLower(strings.TrimSpace(email)))

	user, err := scanUserRow(row)
	if err != nil {
		return domain.User{}, normalizeError(err)
	}
	return user, nil
}

func (r UserRepository) GetUserByID(ctx context.Context, userID domain.ID) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, email, display_name, role, password_hash, must_change_password, disabled_at, created_at, updated_at
		FROM users
		WHERE id = $1::uuid
	`, string(userID))

	user, err := scanUserRow(row)
	if err != nil {
		return domain.User{}, normalizeError(err)
	}
	return user, nil
}

func (r UserRepository) HasActiveAdmin(ctx context.Context) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users
			WHERE role = 'admin' AND disabled_at IS NULL
		)
	`).Scan(&exists)
	if err != nil {
		return false, normalizeError(err)
	}
	return exists, nil
}

func (r UserRepository) PromoteToBootstrapAdmin(ctx context.Context, userID domain.ID, displayName string, passwordHash string) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE users
		SET
			role = 'admin',
			display_name = COALESCE(NULLIF($2, ''), display_name),
			password_hash = $3,
			must_change_password = true,
			disabled_at = NULL,
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text, email, display_name, role, password_hash, must_change_password, disabled_at, created_at, updated_at
	`, string(userID), strings.TrimSpace(displayName), passwordHash)

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

func (r UserRepository) UpdateUserPassword(ctx context.Context, userID domain.ID, passwordHash string, mustChangePassword bool) error {
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, must_change_password = $3, updated_at = now()
		WHERE id = $1::uuid
	`, string(userID), passwordHash, mustChangePassword)
	if err != nil {
		return normalizeError(err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r UserRepository) UpdateUserAccount(ctx context.Context, userID domain.ID, email string, displayName string) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE users
		SET
			email = $2,
			display_name = $3,
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text, email, display_name, role, password_hash, must_change_password, disabled_at, created_at, updated_at
	`, string(userID), strings.ToLower(strings.TrimSpace(email)), strings.TrimSpace(displayName))

	user, err := scanUserRow(row)
	if err != nil {
		return domain.User{}, normalizeError(err)
	}
	return user, nil
}

func (r UserRepository) CreateAdminAccessKey(ctx context.Context, userID domain.ID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO admin_access_keys (user_id, token_hash, expires_at)
		VALUES ($1::uuid, $2, $3)
	`, string(userID), tokenHash, expiresAt)
	return normalizeError(err)
}

func (r UserRepository) ConsumeAdminAccessKey(ctx context.Context, userID domain.ID, tokenHash string, now time.Time) (bool, error) {
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE admin_access_keys
		SET used_at = $4
		WHERE user_id = $1::uuid
		  AND token_hash = $2
		  AND used_at IS NULL
		  AND expires_at > $3
	`, string(userID), tokenHash, now, now)
	if err != nil {
		return false, normalizeError(err)
	}
	return commandTag.RowsAffected() > 0, nil
}

func (r UserRepository) GetUserBySessionTokenHash(ctx context.Context, tokenHash string, now time.Time) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.display_name, u.role, u.password_hash, u.must_change_password, u.disabled_at, u.created_at, u.updated_at
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
		user               domain.User
		id                 string
		role               string
		mustChangePassword bool
		disabledAt         sql.NullTime
	)

	err := scanner.Scan(
		&id,
		&user.Email,
		&user.DisplayName,
		&role,
		&user.PasswordHash,
		&mustChangePassword,
		&disabledAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}

	user.ID = domain.ID(id)
	user.Role = domain.UserRole(role)
	user.MustChangePassword = mustChangePassword
	if disabledAt.Valid {
		t := disabledAt.Time
		user.DisabledAt = &t
	}
	return user, nil
}
