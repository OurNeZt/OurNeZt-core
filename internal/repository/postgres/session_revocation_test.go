package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func sessionPostgres(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run isolated-schema PostgreSQL session tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := fmt.Sprintf("session_test_%d", time.Now().UnixNano())
	schema := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	cfg.ConnConfig.RuntimeParams["application_name"] = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	files, err := filepath.Glob("../../../db/migrations/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("migrations unavailable: %v", err)
	}
	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("migration %s: %v", file, err)
		}
	}
	return pool, name
}

func sessionUser(t *testing.T, repo UserRepository, email string) domain.User {
	t.Helper()
	user, err := repo.CreateUser(context.Background(), domain.User{Email: email, Role: domain.UserRoleUser, PasswordHash: "verified-hash", MustChangePassword: true})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func createTestSession(t *testing.T, repo UserRepository, user domain.User, token string) {
	t.Helper()
	if err := repo.CreateSession(context.Background(), user.ID, user.PasswordHash, security.HashToken(token), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func requireSession(t *testing.T, repo UserRepository, token string, valid bool) {
	t.Helper()
	_, err := repo.GetUserBySessionTokenHash(context.Background(), security.HashToken(token), time.Now())
	if valid && err != nil {
		t.Fatalf("valid session rejected: %v", err)
	}
	if !valid && !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("invalid token remains reusable: %v", err)
	}
}

func TestSessionRevocationPostgresLifecycle(t *testing.T) {
	pool, _ := sessionPostgres(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()
	user, other := sessionUser(t, repo, "user@session.test"), sessionUser(t, repo, "other@session.test")
	createTestSession(t, repo, user, "first")
	createTestSession(t, repo, user, "second")
	createTestSession(t, repo, other, "other")
	if err := repo.CreateSession(ctx, user.ID, user.PasswordHash, security.HashToken("expired"), time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	requireSession(t, repo, "expired", false)
	for i := 0; i < 2; i++ {
		if err := repo.RevokeSessionByTokenHash(ctx, security.HashToken("first")); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.RevokeSessionByTokenHash(ctx, security.HashToken("unknown")); err != nil {
		t.Fatal(err)
	}
	requireSession(t, repo, "first", false)
	requireSession(t, repo, "second", true)
	requireSession(t, repo, "other", true)
	if err := repo.UpdateUserPassword(ctx, user.ID, user.PasswordHash, "new-hash", false); err != nil {
		t.Fatal(err)
	}
	requireSession(t, repo, "second", false)
	requireSession(t, repo, "other", true)
	if err := repo.CreateSession(ctx, user.ID, user.PasswordHash, security.HashToken("stale-login"), time.Now().Add(time.Hour)); !errors.Is(err, apperror.ErrUnauthenticated) {
		t.Fatalf("old verified credentials minted a session: %v", err)
	}
	updated, err := repo.GetUserByID(ctx, user.ID)
	if err != nil || updated.MustChangePassword || updated.PasswordHash != "new-hash" {
		t.Fatalf("password update failed: %v", err)
	}
	createTestSession(t, repo, updated, "fresh")
	if err := repo.UpdateUserPassword(ctx, user.ID, user.PasswordHash, "stale-change", false); !errors.Is(err, apperror.ErrUnauthenticated) {
		t.Fatalf("stale password change succeeded: %v", err)
	}
	requireSession(t, repo, "fresh", true)
	if _, err := repo.PromoteToBootstrapAdmin(ctx, user.ID, "Bootstrap", "bootstrap-hash"); err != nil {
		t.Fatal(err)
	}
	requireSession(t, repo, "fresh", false)
	requireSession(t, repo, "other", true)
	if err := repo.DisableUser(ctx, other.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(ctx, other.ID, other.PasswordHash, security.HashToken("disabled-login"), time.Now().Add(time.Hour)); !errors.Is(err, apperror.ErrUnauthenticated) {
		t.Fatalf("disabled user minted a session: %v", err)
	}
}

func TestSessionRevocationPostgresRollsBackPasswordOnFailure(t *testing.T) {
	pool, _ := sessionPostgres(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()
	user := sessionUser(t, repo, "rollback@session.test")
	createTestSession(t, repo, user, "still-valid")
	_, err := pool.Exec(ctx, `
 CREATE FUNCTION reject_session_revoke() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'injected revocation failure'; END $$;
 CREATE TRIGGER reject_session_revoke BEFORE UPDATE OF revoked_at ON sessions
 FOR EACH ROW EXECUTE FUNCTION reject_session_revoke();
 `)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateUserPassword(ctx, user.ID, user.PasswordHash, "must-not-commit", false); err == nil {
		t.Fatal("revocation failure was ignored")
	}
	got, err := repo.GetUserByID(ctx, user.ID)
	if err != nil || got.PasswordHash != user.PasswordHash || !got.MustChangePassword {
		t.Fatalf("partial password update committed: %v", err)
	}
	requireSession(t, repo, "still-valid", true)
	if _, err := repo.PromoteToBootstrapAdmin(ctx, user.ID, "Bootstrap", "must-not-commit"); err == nil {
		t.Fatal("bootstrap revocation failure was ignored")
	}
	got, err = repo.GetUserByID(ctx, user.ID)
	if err != nil || got.Role != domain.UserRoleUser || got.PasswordHash != user.PasswordHash {
		t.Fatal("partial bootstrap credential update committed")
	}
	requireSession(t, repo, "still-valid", true)
}

func waitForSessionLock(t *testing.T, pool *pgxpool.Pool, appName string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for {
		var waiting bool
		err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name = $1 AND wait_event_type = 'Lock' AND pid <> pg_backend_pid())", appName).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("concurrent operation did not wait for the user lock")
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestSessionRevocationPostgresConcurrentLoginAndPasswordChange(t *testing.T) {
	pool, appName := sessionPostgres(t)
	repo := NewUserRepository(pool)
	user := sessionUser(t, repo, "race@session.test")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// A login attempting to commit after the password change must be rejected.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func(tx pgx.Tx) { _ = tx.Rollback(ctx) }(tx)
	if _, err := tx.Exec(ctx, "UPDATE users SET password_hash = 'changed-hash' WHERE id=$1::uuid", user.ID); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- repo.CreateSession(ctx, user.ID, user.PasswordHash, security.HashToken("late-login"), time.Now().Add(time.Hour))
	}()
	waitForSessionLock(t, pool, appName)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, apperror.ErrUnauthenticated) {
		t.Fatalf("blocked stale login was accepted: %v", err)
	}
	requireSession(t, repo, "late-login", false)
	// A login committed first must be included in the subsequent revocation.
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func(tx pgx.Tx) { _ = tx.Rollback(ctx) }(tx)
	var hash string
	if err := tx.QueryRow(ctx, "SELECT password_hash FROM users WHERE id=$1::uuid FOR UPDATE", user.ID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO sessions(user_id,token_hash,expires_at) VALUES($1,$2,$3)", user.ID, security.HashToken("early-login"), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	go func() { done <- repo.UpdateUserPassword(ctx, user.ID, hash, "final-hash", false) }()
	waitForSessionLock(t, pool, appName)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	requireSession(t, repo, "early-login", false)
}
