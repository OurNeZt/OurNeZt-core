package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

func (r Repository) hasFamilyAccess(ctx context.Context, userID domain.ID, familyID domain.ID) (bool, error) {
	if strings.TrimSpace(string(userID)) == "" || strings.TrimSpace(string(familyID)) == "" {
		return false, apperror.ErrInvalidArgument
	}

	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM family_members
			WHERE family_id = $1::uuid AND user_id = $2::uuid
		)
	`, string(familyID), string(userID)).Scan(&exists)
	if err != nil {
		return false, normalizeError(err)
	}
	return exists, nil
}

func normalizeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperror.ErrConflict
		case "22P02":
			return apperror.ErrInvalidArgument
		}
	}

	return err
}

func optionalDateString(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

func parseOptionalDate(v sql.NullString) *time.Time {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", v.String)
	if err != nil {
		return nil
	}
	return &parsed
}

