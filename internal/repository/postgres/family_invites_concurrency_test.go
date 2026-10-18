package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/jackc/pgx/v5"
)

// Hold a repository operation inside its transaction, using a test-only trigger
// and advisory lock. pg_stat_activity confirms ordering instead of sleeps.
func blockInviteInsert(t *testing.T, f inviteFixture, table string) func() {
	t.Helper()
	key := time.Now().UnixNano()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sql := fmt.Sprintf(`
		CREATE FUNCTION block_invite_insert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN PERFORM pg_advisory_xact_lock(%d); RETURN NEW; END $$;
		CREATE TRIGGER block_invite_insert BEFORE INSERT ON %s
		FOR EACH ROW EXECUTE FUNCTION block_invite_insert();
	`, key, pgx.Identifier{table}.Sanitize())
	if _, err := f.pool.Exec(ctx, sql); err != nil {
		t.Fatal(err)
	}
	conn, err := f.pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, key); err != nil {
		conn.Release()
		t.Fatal(err)
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, key); err != nil {
				t.Error(err)
			}
			conn.Release()
		})
	}
	t.Cleanup(release)
	return release
}

func waitForInviteLock(t *testing.T, ctx context.Context, f inviteFixture, queryPattern string) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		err := f.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE application_name=$1 AND wait_event_type='Lock'
				  AND query LIKE $2 AND pid <> pg_backend_pid()
			)
		`, f.appName, queryPattern).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("operation did not wait for the expected database lock (%s): %v", queryPattern, ctx.Err())
		case <-ticker.C:
		}
	}
}

func TestFamilyInvitePostgresConcurrentRegenerations(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing-invite=%t", existing), func(t *testing.T) {
			f := familyInviteFixture(t)
			if existing {
				f.generate(t, "original")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			release := blockInviteInsert(t, f, "family_invites")
			first, second := make(chan error, 1), make(chan error, 1)
			go func() {
				first <- f.repo.GenerateFamilyJoinCode(ctx, f.owner.ID, f.family.ID, security.HashToken("first"), nil)
			}()
			waitForInviteLock(t, ctx, f, "%INSERT INTO family_invites%")
			go func() {
				second <- f.repo.GenerateFamilyJoinCode(ctx, f.owner.ID, f.family.ID, security.HashToken("second"), nil)
			}()
			waitForInviteLock(t, ctx, f, "%FOR UPDATE%")
			release()
			if err := <-first; err != nil {
				t.Fatal(err)
			}
			if err := <-second; err != nil {
				t.Fatal(err)
			}
			f.requireLatest(t, "second")
			user := f.user(t, "concurrent-member")
			f.requireJoin(t, user, "first", false)
			if existing {
				f.requireJoin(t, user, "original", false)
			}
			f.requireJoin(t, user, "second", true)
		})
	}
}

func TestFamilyInvitePostgresJoinWaitsForRegeneration(t *testing.T) {
	f := familyInviteFixture(t)
	f.generate(t, "original")
	user := f.user(t, "late-member")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	release := blockInviteInsert(t, f, "family_invites")
	generated, joined := make(chan error, 1), make(chan error, 1)
	go func() {
		generated <- f.repo.GenerateFamilyJoinCode(ctx, f.owner.ID, f.family.ID, security.HashToken("replacement"), nil)
	}()
	waitForInviteLock(t, ctx, f, "%INSERT INTO family_invites%")
	go func() { _, err := f.repo.JoinFamilyByCode(ctx, user.ID, security.HashToken("original")); joined <- err }()
	waitForInviteLock(t, ctx, f, "%FOR SHARE OF f%")
	release()
	if err := <-generated; err != nil {
		t.Fatal(err)
	}
	if err := <-joined; !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("join accepted a code replaced while it was waiting: %v", err)
	}
	f.requireMembership(t, user, false)
	f.requireJoin(t, user, "replacement", true)
}

func TestFamilyInvitePostgresRegenerationWaitsForJoin(t *testing.T) {
	f := familyInviteFixture(t)
	f.generate(t, "original")
	user := f.user(t, "early-member")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	release := blockInviteInsert(t, f, "family_members")
	joined, generated := make(chan error, 1), make(chan error, 1)
	go func() { _, err := f.repo.JoinFamilyByCode(ctx, user.ID, security.HashToken("original")); joined <- err }()
	waitForInviteLock(t, ctx, f, "%INSERT INTO family_members%")
	go func() {
		generated <- f.repo.GenerateFamilyJoinCode(ctx, f.owner.ID, f.family.ID, security.HashToken("replacement"), nil)
	}()
	waitForInviteLock(t, ctx, f, "%FOR UPDATE%")
	release()
	if err := <-joined; err != nil {
		t.Fatal(err)
	}
	if err := <-generated; err != nil {
		t.Fatal(err)
	}
	f.requireMembership(t, user, true)
	late := f.user(t, "next-member")
	f.requireJoin(t, late, "original", false)
	f.requireMembership(t, late, false)
	f.requireJoin(t, late, "replacement", true)
}

func TestFamilyInvitePostgresJoinRechecksExpiryAfterWaiting(t *testing.T) {
	f := familyInviteFixture(t)
	f.generate(t, "expiring")
	user := f.user(t, "expired-while-waiting")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT id FROM families WHERE id=$1 FOR UPDATE`, f.family.ID); err != nil {
		t.Fatal(err)
	}
	joined := make(chan error, 1)
	go func() { _, err := f.repo.JoinFamilyByCode(ctx, user.ID, security.HashToken("expiring")); joined <- err }()
	waitForInviteLock(t, ctx, f, "%FOR SHARE OF f%")
	// This expiry is after the join transaction's start, but before it can
	// acquire the family lock. Transaction-start now() would accept the code.
	if _, err := tx.Exec(ctx, `UPDATE family_invites SET expires_at=clock_timestamp() WHERE family_id=$1`, f.family.ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-joined; !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("join used an invite that expired while waiting: %v", err)
	}
	f.requireMembership(t, user, false)
}
