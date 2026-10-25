package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
)

func resourceExec(t *testing.T, f inviteFixture, query string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func TestResourceLimitsPostgresConcurrentQuotas(t *testing.T) {
	for _, tc := range []struct {
		table        string
		cap          int
		seed, insert string
		userScope    bool
	}{
		{"sessions", 100, `INSERT INTO sessions(user_id,token_hash,expires_at) SELECT $1,'seed-'||i,now()+interval '1 hour' FROM generate_series(1,$2::int) i`, `INSERT INTO sessions(user_id,token_hash,expires_at) VALUES($1,$2,now()+interval '1 hour')`, true},
		{"admin_access_keys", 100, `INSERT INTO admin_access_keys(user_id,token_hash,expires_at) SELECT $1,'seed-'||i,now()+interval '1 hour' FROM generate_series(1,$2::int) i`, `INSERT INTO admin_access_keys(user_id,token_hash,expires_at) VALUES($1,$2,now()+interval '1 hour')`, true},
		{"person_profiles", 20, `INSERT INTO person_profiles(family_id,name,age,employment_status) SELECT $1,'Person '||i,30,'full_time_employee' FROM generate_series(1,$2::int) i`, `INSERT INTO person_profiles(family_id,name,age,employment_status) VALUES($1,$2,30,'full_time_employee')`, false},
		{"housing_options", 100, `INSERT INTO housing_options(family_id,name,housing_type,purchase_price_cents,loan_type) SELECT $1,'Home '||i,'bto',1,'cash' FROM generate_series(1,$2::int) i`, `INSERT INTO housing_options(family_id,name,housing_type,purchase_price_cents,loan_type) VALUES($1,$2,'bto',1,'cash')`, false},
		{"housing_groups", 50, `INSERT INTO housing_groups(family_id,name) SELECT $1,'Group '||i FROM generate_series(1,$2::int) i`, `INSERT INTO housing_groups(family_id,name) VALUES($1,$2)`, false},
		{"housing_checklist_criteria", 50, `INSERT INTO housing_checklist_criteria(family_id,name,display_order,deleted_at) SELECT $1,'Criterion '||i,i,CASE WHEN i%2=0 THEN now() END FROM generate_series(1,$2::int) i`, `INSERT INTO housing_checklist_criteria(family_id,name,display_order) VALUES($1,$2,50)`, false},
		{"family_invites", 100, `INSERT INTO family_invites(family_id,code_hash,expires_at,revoked_at) SELECT $1,'seed-'||i,now()+interval '1 hour',CASE WHEN i%2=0 THEN now() END FROM generate_series(1,$2::int) i`, `INSERT INTO family_invites(family_id,code_hash,expires_at) VALUES($1,$2,now()+interval '1 hour')`, false},
	} {
		t.Run(tc.table, func(t *testing.T) {
			f := familyInviteFixture(t)
			owner := f.family.ID
			if tc.userScope {
				owner = f.owner.ID
			}
			resourceExec(t, f, tc.seed, owner, tc.cap-1)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			start := make(chan struct{})
			results := make(chan error, 2)
			for i := 0; i < 2; i++ {
				go func(i int) {
					<-start
					_, err := f.pool.Exec(ctx, tc.insert, owner, fmt.Sprintf("concurrent-%d", i))
					results <- normalizeError(err)
				}(i)
			}
			close(start)
			success, limited := 0, 0
			for i := 0; i < 2; i++ {
				err := <-results
				if err == nil {
					success++
				} else if errors.Is(err, apperror.ErrQuotaExceeded) {
					limited++
				} else {
					t.Fatal(err)
				}
			}
			if success != 1 || limited != 1 {
				t.Fatalf("quota race: success=%d limited=%d", success, limited)
			}
			var count int
			if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM "+tc.table).Scan(&count); err != nil || count != tc.cap {
				t.Fatalf("count=%d error=%v", count, err)
			}
			// Another owner/family has an independent budget.
			other := f.user(t, "other")
			if !tc.userScope {
				family, err := f.repo.CreateFamily(ctx, domain.Family{Name: "Other", Type: domain.FamilyTypeFamily}, other.ID)
				if err != nil {
					t.Fatal(err)
				}
				owner = family.ID
			} else {
				owner = other.ID
			}
			resourceExec(t, f, tc.insert, owner, "independent")
		})
	}
}

func TestResourceLimitsPostgresFamilyMemberships(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	for i := 1; i < 10; i++ {
		if _, err := f.repo.CreateFamily(ctx, domain.Family{Name: fmt.Sprintf("Home %d", i), Type: domain.FamilyTypeFamily}, f.owner.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.repo.CreateFamily(ctx, domain.Family{Name: "Over quota", Type: domain.FamilyTypeFamily}, f.owner.ID); !errors.Is(err, apperror.ErrQuotaExceeded) {
		t.Fatalf("family cap bypassed: %v", err)
	}
	var count int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM families").Scan(&count); err != nil || count != 10 {
		t.Fatalf("failed create left orphan: count=%d err=%v", count, err)
	}
	other := f.user(t, "other-owner")
	family, err := f.repo.CreateFamily(ctx, domain.Family{Name: "Other", Type: domain.FamilyTypeFamily}, other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.GenerateFamilyJoinCode(ctx, other.ID, family.ID, "invite", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.JoinFamilyByCode(ctx, f.owner.ID, "invite"); !errors.Is(err, apperror.ErrQuotaExceeded) {
		t.Fatalf("joining bypassed family cap: %v", err)
	}
}

func TestResourceLimitsPostgresConcurrentFamilyJoins(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	resourceExec(t, f, `WITH created AS (INSERT INTO users(email,role,password_hash) SELECT 'member-'||i||'@test','user','hash' FROM generate_series(1,18) i RETURNING id) INSERT INTO family_members(family_id,user_id,role) SELECT $1,id,'member' FROM created`, f.family.ID)
	f.generate(t, "join-code")
	first, second := f.user(t, "first"), f.user(t, "second")
	start, results := make(chan struct{}), make(chan error, 2)
	for _, user := range []domain.User{first, second} {
		go func(user domain.User) {
			<-start
			_, err := f.repo.JoinFamilyByCode(ctx, user.ID, security.HashToken("join-code"))
			results <- err
		}(user)
	}
	close(start)
	success, limited := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, apperror.ErrQuotaExceeded) {
			limited++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || limited != 1 {
		t.Fatalf("member quota race: success=%d limited=%d", success, limited)
	}
	var members int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM family_members WHERE family_id=$1", f.family.ID).Scan(&members); err != nil || members != 20 {
		t.Fatalf("member quota count=%d, err=%v", members, err)
	}
	for i := 1; i < 10; i++ {
		if _, err := f.repo.CreateFamily(ctx, domain.Family{Name: "Other home", Type: domain.FamilyTypeFamily}, f.owner.ID); err != nil {
			t.Fatal(err)
		}
	}
	// Existing membership can be joined again even when both quotas are full.
	f.requireJoin(t, f.owner, "join-code", true)
}

func TestResourceLimitsPostgresMixedFamilyMemberships(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	member := f.user(t, "member")
	if _, err := f.repo.CreateFamily(ctx, domain.Family{Name: "Owned home", Type: domain.FamilyTypeFamily}, member.ID); err != nil {
		t.Fatal(err)
	}
	// Nine joined families plus one owned family use the same ten slots.
	for i := 0; i < 9; i++ {
		family, err := f.repo.CreateFamily(ctx, domain.Family{Name: fmt.Sprintf("Joined %d", i), Type: domain.FamilyTypeFamily}, f.owner.ID)
		if err != nil {
			t.Fatal(err)
		}
		code := fmt.Sprintf("mixed-%d", i)
		if err := f.repo.GenerateFamilyJoinCode(ctx, f.owner.ID, family.ID, security.HashToken(code), nil); err != nil {
			t.Fatal(err)
		}
		if _, err := f.repo.JoinFamilyByCode(ctx, member.ID, security.HashToken(code)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.repo.CreateFamily(ctx, domain.Family{Name: "Over quota", Type: domain.FamilyTypeFamily}, member.ID); !errors.Is(err, apperror.ErrQuotaExceeded) || !strings.Contains(err.Error(), "maximum 10 families per user") {
		t.Fatalf("mixed memberships bypassed creation quota: %v", err)
	}
	f.generate(t, "extra-family")
	if _, err := f.repo.JoinFamilyByCode(ctx, member.ID, security.HashToken("extra-family")); !errors.Is(err, apperror.ErrQuotaExceeded) {
		t.Fatalf("mixed memberships bypassed joining quota: %v", err)
	}
	var families, memberships int
	if err := f.pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM families),(SELECT count(*) FROM family_members WHERE user_id=$1)", member.ID).Scan(&families, &memberships); err != nil || families != 11 || memberships != 10 {
		t.Fatalf("failed operation altered families/memberships: families=%d memberships=%d err=%v", families, memberships, err)
	}
	f.requireMembership(t, member, false)
	f.requireJoin(t, f.user(t, "independent-member"), "extra-family", true)
}

func TestResourceLimitsPostgresInputConstraints(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	for _, tc := range []struct{ name, query, value string }{
		{"email", `UPDATE users SET email=$1 WHERE id=$2`, strings.Repeat("a", 255)},
		{"display name", `UPDATE users SET display_name=$1 WHERE id=$2`, strings.Repeat("界", 121)},
		{"password hash", `UPDATE users SET password_hash=$1 WHERE id=$2`, strings.Repeat("a", 513)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.pool.Exec(ctx, tc.query, tc.value, f.owner.ID)
			if !errors.Is(normalizeError(err), apperror.ErrInvalidArgument) {
				t.Fatalf("constraint bypassed: %v", err)
			}
		})
	}
	resourceExec(t, f, `UPDATE users SET display_name=$1 WHERE id=$2`, strings.Repeat("界", 120), f.owner.ID)
	if _, err := f.pool.Exec(ctx, `UPDATE families SET name=$1 WHERE id=$2`, strings.Repeat("界", 121), f.family.ID); !errors.Is(normalizeError(err), apperror.ErrInvalidArgument) {
		t.Fatal("family name constraint bypassed")
	}
	var housing domain.ID
	if err := f.pool.QueryRow(ctx, `INSERT INTO housing_options(family_id,name,housing_type,purchase_price_cents,loan_type) VALUES($1,'Home','bto',1,'cash') RETURNING id::text`, f.family.ID).Scan(&housing); err != nil {
		t.Fatal(err)
	}
	for _, col := range []string{"location", "unit_type", "name"} {
		limit := 120
		if col == "location" {
			limit = 255
		}
		_, err := f.pool.Exec(ctx, "UPDATE housing_options SET "+col+"=$1 WHERE id=$2", strings.Repeat("界", limit+1), housing)
		if !errors.Is(normalizeError(err), apperror.ErrInvalidArgument) {
			t.Fatalf("%s constraint bypassed: %v", col, err)
		}
	}
	if _, err := f.pool.Exec(ctx, `UPDATE housing_options SET dia_income_overrides=(SELECT jsonb_agg('{}'::jsonb) FROM generate_series(1,51)) WHERE id=$1`, housing); !errors.Is(normalizeError(err), apperror.ErrInvalidArgument) {
		t.Fatal("override count constraint bypassed")
	}
}

func TestResourceLimitsPostgresConcurrentFamilyCreates(t *testing.T) {
	f := familyInviteFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for i := 0; i < 8; i++ {
		if _, err := f.repo.CreateFamily(ctx, domain.Family{Name: "Home", Type: domain.FamilyTypeFamily}, f.owner.ID); err != nil {
			t.Fatal(err)
		}
	}
	start, results := make(chan struct{}), make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			_, err := f.repo.CreateFamily(ctx, domain.Family{Name: "Concurrent", Type: domain.FamilyTypeFamily}, f.owner.ID)
			results <- err
		}()
	}
	close(start)
	success, limited := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, apperror.ErrQuotaExceeded) {
			limited++
		} else {
			t.Fatal(err)
		}
	}
	var families, members int
	if err := f.pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM families),(SELECT count(*) FROM family_members)").Scan(&families, &members); err != nil {
		t.Fatal(err)
	}
	if success != 1 || limited != 1 || families != 10 || members != 10 {
		t.Fatalf("family quota race/orphan: success=%d limited=%d families=%d memberships=%d", success, limited, families, members)
	}
}

func TestResourceLimitsPostgresQuotaFailurePreservesInvite(t *testing.T) {
	f := familyInviteFixture(t)
	f.generate(t, "current")
	resourceExec(t, f, `INSERT INTO family_invites(family_id,code_hash,revoked_at) SELECT $1,'old-'||i,now() FROM generate_series(1,99) i`, f.family.ID)
	if err := f.repo.GenerateFamilyJoinCode(context.Background(), f.owner.ID, f.family.ID, security.HashToken("replacement"), nil); !errors.Is(err, apperror.ErrQuotaExceeded) {
		t.Fatal(err)
	}
	f.requireLatest(t, "current")
}

func TestResourceLimitsPostgresRetentionReleasesQuota(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	users := NewUserRepository(f.pool)
	resourceExec(t, f, `INSERT INTO sessions(user_id,token_hash,expires_at) SELECT $1,'expired-'||i,now()-interval '8 days' FROM generate_series(1,100) i`, f.owner.ID)
	if err := users.CreateSession(ctx, f.owner.ID, f.owner.PasswordHash, "new-session", time.Now().Add(time.Hour)); !errors.Is(err, apperror.ErrQuotaExceeded) {
		t.Fatal(err)
	}
	if result, err := New(f.pool).CleanupExpiredCredentials(ctx, time.Now(), 1); err != nil || result.Sessions != 1 {
		t.Fatalf("cleanup: %+v %v", result, err)
	}
	if err := users.CreateSession(ctx, f.owner.ID, f.owner.PasswordHash, "new-session", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func TestResourceLimitsPostgresRetention(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	cutoff := now.Add(-CredentialRetention)
	for _, table := range []string{"sessions", "admin_access_keys", "family_invites"} {
		owner, hash, terminal := "user_id", "token_hash", "revoked_at"
		id := f.owner.ID
		if table == "admin_access_keys" {
			terminal = "used_at"
		}
		if table == "family_invites" {
			owner, hash = "family_id", "code_hash"
			id = f.family.ID
		}
		query := fmt.Sprintf(`INSERT INTO %s(%s,%s,expires_at,%s) VALUES($1,'expired',$2,NULL),($1,'terminal',$3,$2),($1,'recent-expired',$4,NULL),($1,'recent-terminal',$3,$4),($1,'active',$3,NULL),($1,'boundary',$5,NULL),($1,'inside-boundary',$6,NULL)`, table, owner, hash, terminal)
		resourceExec(t, f, query, id, cutoff.Add(-time.Hour), now.Add(time.Hour), now.Add(-time.Hour), cutoff, cutoff.Add(time.Microsecond))
	}
	resourceExec(t, f, `INSERT INTO family_invites(family_id,code_hash) VALUES($1,'legacy-active')`, f.family.ID)
	for i := 0; i < 3; i++ {
		got, err := New(f.pool).CleanupExpiredCredentials(ctx, now, 1)
		if err != nil || got != (RetentionResult{1, 1, 1}) {
			t.Fatalf("batch %d: %+v %v", i, got, err)
		}
	}
	got, err := New(f.pool).CleanupExpiredCredentials(ctx, now, 1)
	if err != nil || got != (RetentionResult{}) {
		t.Fatalf("active/recent credentials removed: %+v %v", got, err)
	}
	for _, table := range []string{"sessions", "admin_access_keys", "family_invites"} {
		var count int
		want := 4
		if table == "family_invites" {
			want = 5
		}
		if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s retained=%d want=%d err=%v", table, count, want, err)
		}
	}
	if _, err := New(f.pool).CleanupExpiredCredentials(ctx, now, RetentionBatchSize+1); err == nil {
		t.Fatal("unbounded batch accepted")
	}
	// Concurrent workers safely share deletion; no active row can be claimed.
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := New(f.pool).CleanupExpiredCredentials(ctx, now, 2); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestResourceLimitsPostgresMigrationPreservesLegacy(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	down, err := os.ReadFile("../../../db/migrations/000008_resource_limits.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	resourceExec(t, f, string(down))
	resourceExec(t, f, `UPDATE users SET display_name=$1 WHERE id=$2`, strings.Repeat("界", 121), f.owner.ID)
	resourceExec(t, f, `INSERT INTO person_profiles(family_id,name,age,employment_status) SELECT $1,'Legacy',30,'full_time_employee' FROM generate_series(1,21)`, f.family.ID)
	up, err := os.ReadFile("../../../db/migrations/000008_resource_limits.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	resourceExec(t, f, string(up))
	var count int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM person_profiles").Scan(&count); err != nil || count != 21 {
		t.Fatal("migration removed legacy records")
	}
	if _, err := NewPersonRepository(f.pool).ListPersonProfilesByFamily(ctx, f.family.ID, f.owner.ID, domain.Page{Size: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO person_profiles(family_id,name,age,employment_status) VALUES($1,'Extra',30,'full_time_employee')`, f.family.ID); !errors.Is(normalizeError(err), apperror.ErrQuotaExceeded) {
		t.Fatal("legacy family bypassed new quota")
	}
	resourceExec(t, f, `UPDATE users SET display_name='Repaired' WHERE id=$1`, f.owner.ID)
}
