package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/jackc/pgx/v5/pgxpool"
)

type inviteFixture struct {
	pool    *pgxpool.Pool
	appName string
	repo    FamilyRepository
	owner   domain.User
	family  domain.Family
}

func familyInviteFixture(t *testing.T) inviteFixture {
	t.Helper()
	pool, name := isolatedPostgres(t, "family_invite")
	f := inviteFixture{pool: pool, appName: name, repo: NewFamilyRepository(pool)}
	f.owner = f.user(t, "owner")
	var err error
	f.family, err = f.repo.CreateFamily(context.Background(), domain.Family{Name: "Home", Type: domain.FamilyTypeFamily}, f.owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f inviteFixture) user(t *testing.T, name string) domain.User {
	t.Helper()
	user, err := NewUserRepository(f.pool).CreateUser(context.Background(), domain.User{Email: name + "@invites.test", Role: domain.UserRoleUser, PasswordHash: "test-hash"})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func (f inviteFixture) generate(t *testing.T, code string) {
	t.Helper()
	expires := time.Now().Add(time.Hour)
	if err := f.repo.GenerateFamilyJoinCode(context.Background(), f.owner.ID, f.family.ID, security.HashToken(code), &expires); err != nil {
		t.Fatal(err)
	}
}

func (f inviteFixture) requireJoin(t *testing.T, user domain.User, code string, valid bool) {
	t.Helper()
	family, err := f.repo.JoinFamilyByCode(context.Background(), user.ID, security.HashToken(code))
	if valid && (err != nil || family.ID != f.family.ID) {
		t.Fatalf("current invite rejected: family=%s error=%v", family.ID, err)
	}
	if !valid && !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("replaced/invalid invite was not rejected: %v", err)
	}
}

func (f inviteFixture) requireMembership(t *testing.T, user domain.User, want bool) {
	t.Helper()
	var exists bool
	err := f.pool.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM family_members WHERE family_id=$1 AND user_id=$2)`, f.family.ID, user.ID).Scan(&exists)
	if err != nil || exists != want {
		t.Fatalf("membership=%t, want %t: %v", exists, want, err)
	}
}

func (f inviteFixture) requireLatest(t *testing.T, code string) {
	t.Helper()
	var count int
	var latest string
	err := f.pool.QueryRow(context.Background(), `
		SELECT count(*), COALESCE(min(code_hash), '') FROM family_invites
		WHERE family_id=$1 AND revoked_at IS NULL
	`, f.family.ID).Scan(&count, &latest)
	if err != nil || count != 1 || latest != security.HashToken(code) {
		t.Fatalf("expected exactly the latest unrevoked invite, got %d: %v", count, err)
	}
}

func TestFamilyInvitePostgresRegenerationRevokesAllPreviousCodes(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	otherOwner := f.user(t, "other-owner")
	otherFamily, err := f.repo.CreateFamily(ctx, domain.Family{Name: "Other", Type: domain.FamilyTypeFamily}, otherOwner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.GenerateFamilyJoinCode(ctx, otherOwner.ID, otherFamily.ID, security.HashToken("other-family"), nil); err != nil {
		t.Fatal(err)
	}
	// Simulate legacy data with several still-active invites, including one
	// without expiry. Existing revocation timestamps must be preserved.
	_, err = f.pool.Exec(ctx, `
		INSERT INTO family_invites(family_id,code_hash,expires_at,revoked_at,created_by) VALUES
		($1,$2,NULL,NULL,$6),
		($1,$3,now()+interval '1 hour',NULL,$6),
		($1,$4,now()-interval '1 hour',NULL,$6),
		($1,$5,NULL,'2026-01-01 UTC',$6)
	`, f.family.ID, security.HashToken("legacy-no-expiry"), security.HashToken("legacy-active"), security.HashToken("expired"), security.HashToken("already-revoked"), f.owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	before := f.user(t, "joined-before")
	f.requireJoin(t, before, "legacy-active", true)
	f.generate(t, "latest")
	f.requireLatest(t, "latest")
	newUser := f.user(t, "new-member")
	for _, code := range []string{"legacy-no-expiry", "legacy-active", "expired", "already-revoked", "unknown"} {
		f.requireJoin(t, newUser, code, false)
		f.requireMembership(t, newUser, false)
	}
	var preserved bool
	if err := f.pool.QueryRow(ctx, `SELECT revoked_at='2026-01-01 UTC'::timestamptz FROM family_invites WHERE code_hash=$1`, security.HashToken("already-revoked")).Scan(&preserved); err != nil || !preserved {
		t.Fatalf("historical revocation timestamp changed: %v", err)
	}
	f.requireJoin(t, newUser, "latest", true)
	f.requireJoin(t, newUser, "latest", true) // Membership remains idempotent.
	f.requireJoin(t, f.user(t, "another-member"), "latest", true)
	f.requireMembership(t, before, true)
	joinedOther, err := f.repo.JoinFamilyByCode(ctx, f.user(t, "other-member").ID, security.HashToken("other-family"))
	if err != nil || joinedOther.ID != otherFamily.ID {
		t.Fatalf("regeneration affected another family's invite: %v", err)
	}
}

func TestFamilyInvitePostgresRegenerationRequiresManager(t *testing.T) {
	f := familyInviteFixture(t)
	f.generate(t, "original")
	ctx := context.Background()
	for _, role := range []string{"member", "viewer", "outsider", "admin"} {
		actor := f.user(t, role)
		if role != "outsider" {
			if _, err := f.pool.Exec(ctx, `INSERT INTO family_members(family_id,user_id,role) VALUES($1,$2,$3)`, f.family.ID, actor.ID, role); err != nil {
				t.Fatal(err)
			}
		}
		err := f.repo.GenerateFamilyJoinCode(ctx, actor.ID, f.family.ID, security.HashToken(role+"-code"), nil)
		if role == "admin" {
			if err != nil {
				t.Fatalf("family admin cannot regenerate: %v", err)
			}
			f.requireLatest(t, "admin-code")
			f.requireJoin(t, f.user(t, "admin-invited"), "admin-code", true)
		} else {
			if !errors.Is(err, apperror.ErrForbidden) {
				t.Fatalf("%s regenerated the invite: %v", role, err)
			}
			f.requireLatest(t, "original")
		}
	}
}

func TestFamilyInvitePostgresFailedReplacementRollsBack(t *testing.T) {
	for _, failure := range []string{"insert collision", "revocation failure"} {
		t.Run(failure, func(t *testing.T) {
			f := familyInviteFixture(t)
			f.generate(t, "original")
			ctx := context.Background()
			newCode := "original" // Unique violation after revocation has run.
			if failure == "revocation failure" {
				_, err := f.pool.Exec(ctx, `
					CREATE FUNCTION reject_invite_revocation() RETURNS trigger LANGUAGE plpgsql AS $$
					BEGIN RAISE EXCEPTION 'injected revocation failure'; END $$;
					CREATE TRIGGER reject_invite_revocation BEFORE UPDATE OF revoked_at ON family_invites
					FOR EACH ROW EXECUTE FUNCTION reject_invite_revocation();
				`)
				if err != nil {
					t.Fatal(err)
				}
				newCode = "replacement"
			}
			if err := f.repo.GenerateFamilyJoinCode(ctx, f.owner.ID, f.family.ID, security.HashToken(newCode), nil); err == nil {
				t.Fatal("injected replacement failure was ignored")
			}
			f.requireLatest(t, "original")
			f.requireJoin(t, f.user(t, "still-invited"), "original", true)
			var count int
			if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM family_invites WHERE family_id=$1`, f.family.ID).Scan(&count); err != nil || count != 1 {
				t.Fatalf("failed replacement inserted a new invite: count=%d error=%v", count, err)
			}
		})
	}
}

func TestFamilyInvitePostgresExpiredCodesCannotJoin(t *testing.T) {
	f := familyInviteFixture(t)
	past := time.Now().Add(-time.Hour)
	if err := f.repo.GenerateFamilyJoinCode(context.Background(), f.owner.ID, f.family.ID, security.HashToken("expired"), &past); err != nil {
		t.Fatal(err)
	}
	user := f.user(t, "expired-user")
	f.requireJoin(t, user, "expired", false)
	f.requireMembership(t, user, false)
}
