package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/server"
	"github.com/OurNeZt/ournezt-core/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestResourceLimitsPostgresPagination(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	resourceExec(t, f, `UPDATE users SET role='admin' WHERE id=$1`, f.owner.ID)
	resourceExec(t, f, `INSERT INTO users(email,role,password_hash) SELECT 'other-'||i||'@test','user','hash' FROM generate_series(1,5) i`)
	resourceExec(t, f, `INSERT INTO family_members(family_id,user_id,role) SELECT $1,id,'member' FROM users WHERE id<>$2`, f.family.ID, f.owner.ID)
	for i := 0; i < 5; i++ {
		if _, err := f.repo.CreateFamily(ctx, domain.Family{Name: fmt.Sprintf("Home %d", i), Type: domain.FamilyTypeFamily}, f.owner.ID); err != nil {
			t.Fatal(err)
		}
	}
	resourceExec(t, f, `INSERT INTO person_profiles(family_id,name,age,employment_status) SELECT $1,'Person '||i,30,'full_time_employee' FROM generate_series(1,5) i`, f.family.ID)
	resourceExec(t, f, `INSERT INTO person_income_history(family_id,person_profile_id,person_name) SELECT family_id,id,name FROM person_profiles WHERE family_id=$1`, f.family.ID)
	resourceExec(t, f, `INSERT INTO housing_groups(family_id,name) SELECT $1,'Group '||i FROM generate_series(1,5) i`, f.family.ID)
	resourceExec(t, f, `INSERT INTO housing_options(family_id,name,housing_type,purchase_price_cents,loan_type) SELECT $1,'Home '||i,'bto',1,'cash' FROM generate_series(1,5) i`, f.family.ID)
	users := NewUserRepository(f.pool)
	if err := users.CreateSession(ctx, f.owner.ID, f.owner.PasswordHash, security.HashToken("fixture-token"), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	auth := server.NewAuthServer(service.NewAuthService(users, security.DefaultArgon2Params()), users, 32, time.Hour, nil)
	families := server.NewFamilyServer(f.repo, time.Hour, nil, auth)
	people := server.NewPersonServer(NewPersonRepository(f.pool), auth)
	housing := server.NewHousingServer(NewHousingRepository(f.pool), NewPersonRepository(f.pool), auth)
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-session-token", "fixture-token"))
	for _, tc := range []struct {
		name  string
		count int
		fetch func(string) ([]string, string, error)
	}{
		{"users", 6, func(token string) ([]string, string, error) {
			r, e := auth.ListUsers(ctx, &pb.ListUsersRequest{PageSize: 2, PageToken: token})
			if e != nil {
				return nil, "", e
			}
			ids := []string{}
			for _, v := range r.Users {
				ids = append(ids, v.Id)
			}
			return ids, r.NextPageToken, nil
		}},
		{"families", 6, func(token string) ([]string, string, error) {
			r, e := families.ListUserFamilies(ctx, &pb.ListUserFamiliesRequest{PageSize: 2, PageToken: token})
			if e != nil {
				return nil, "", e
			}
			ids := []string{}
			for _, v := range r.Families {
				ids = append(ids, v.Id)
			}
			return ids, r.NextPageToken, nil
		}},
		{"members", 6, func(token string) ([]string, string, error) {
			r, e := families.ListFamilyMembers(ctx, &pb.ListFamilyMembersRequest{FamilyId: string(f.family.ID), PageSize: 2, PageToken: token})
			if e != nil {
				return nil, "", e
			}
			ids := []string{}
			for _, v := range r.Members {
				ids = append(ids, v.UserId)
			}
			return ids, r.NextPageToken, nil
		}},
		{"profiles", 5, func(token string) ([]string, string, error) {
			r, e := people.ListPersonProfilesByFamily(ctx, &pb.ListPersonProfilesByFamilyRequest{FamilyId: string(f.family.ID), PageSize: 2, PageToken: token})
			if e != nil {
				return nil, "", e
			}
			ids := []string{}
			for _, v := range r.People {
				ids = append(ids, v.Id)
			}
			return ids, r.NextPageToken, nil
		}},
		{"income history", 5, func(token string) ([]string, string, error) {
			r, e := people.ListIncomeHistoryByFamily(ctx, &pb.ListIncomeHistoryByFamilyRequest{FamilyId: string(f.family.ID), PageSize: 2, PageToken: token})
			if e != nil {
				return nil, "", e
			}
			ids := []string{}
			for _, v := range r.Entries {
				ids = append(ids, v.PersonId)
			}
			return ids, r.NextPageToken, nil
		}},
		{"housing", 5, func(token string) ([]string, string, error) {
			r, e := housing.ListHousingOptions(ctx, &pb.ListHousingOptionsRequest{FamilyId: string(f.family.ID), PageSize: 2, PageToken: token})
			if e != nil {
				return nil, "", e
			}
			ids := []string{}
			for _, v := range r.HousingOptions {
				ids = append(ids, v.Id)
			}
			return ids, r.NextPageToken, nil
		}},
		{"groups", 5, func(token string) ([]string, string, error) {
			r, e := housing.ListHousingGroups(ctx, &pb.ListHousingGroupsRequest{FamilyId: string(f.family.ID), PageSize: 2, PageToken: token})
			if e != nil {
				return nil, "", e
			}
			ids := []string{}
			for _, v := range r.HousingGroups {
				ids = append(ids, v.Id)
			}
			return ids, r.NextPageToken, nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seen := map[string]bool{}
			token := ""
			for page := 0; page < 10; page++ {
				ids, next, err := tc.fetch(token)
				if err != nil {
					t.Fatal(err)
				}
				if len(ids) > 2 {
					t.Fatal("page size exceeded")
				}
				for _, id := range ids {
					if seen[id] {
						t.Fatalf("duplicate record across tied timestamps: %s", id)
					}
					seen[id] = true
				}
				token = next
				if token == "" {
					break
				}
				if page == 9 {
					t.Fatal("pagination did not terminate")
				}
			}
			if len(seen) != tc.count {
				t.Fatalf("records=%d want=%d", len(seen), tc.count)
			}
		})
	}
	// Cursors survive deletion of their boundary row; they store the sort key,
	// rather than looking up the deleted row or shifting an OFFSET window.
	first, err := housing.ListHousingGroups(ctx, &pb.ListHousingGroupsRequest{FamilyId: string(f.family.ID), PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	resourceExec(t, f, `DELETE FROM housing_groups WHERE id=$1`, first.HousingGroups[1].Id)
	next, err := housing.ListHousingGroups(ctx, &pb.ListHousingGroupsRequest{FamilyId: string(f.family.ID), PageSize: 100, PageToken: first.NextPageToken})
	if err != nil || len(next.GetHousingGroups()) != 3 {
		t.Fatalf("deleted boundary broke cursor: %v %v", next, err)
	}
	other := f.user(t, "outsider")
	if err := users.CreateSession(ctx, other.ID, other.PasswordHash, security.HashToken("outside"), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	outside := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-session-token", "outside"))
	if _, err := housing.ListHousingGroups(outside, &pb.ListHousingGroupsRequest{FamilyId: string(f.family.ID)}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("pagination bypassed family access: %v", err)
	}
	if _, err := housing.ListHousingGroups(outside, &pb.ListHousingGroupsRequest{FamilyId: string(f.family.ID), PageToken: first.NextPageToken}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("cursor crossed viewer scope: %v", err)
	}
}

func TestResourceLimitsPostgresHistoryQuotaRollsBackProfileUpdate(t *testing.T) {
	f := familyInviteFixture(t)
	ctx := context.Background()
	people := NewPersonRepository(f.pool)
	profile, err := people.CreatePersonProfile(ctx, domain.PersonProfile{FamilyID: f.family.ID, Name: "Person", Age: 30, EmploymentStatus: domain.EmploymentFullTime}, f.owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	resourceExec(t, f, `INSERT INTO person_income_history(family_id,person_profile_id,person_name) SELECT $1,$2,'Person' FROM generate_series(1,4999)`, f.family.ID, profile.ID)
	profile.GrossMonthlyIncomeCents = 999
	if _, err := people.UpdatePersonProfile(ctx, profile, f.owner.ID); !errors.Is(err, apperror.ErrQuotaExceeded) {
		t.Fatalf("history quota bypassed: %v", err)
	}
	got, err := people.GetPersonProfile(ctx, profile.ID, f.owner.ID)
	if err != nil || got.GrossMonthlyIncomeCents != 0 {
		t.Fatalf("quota failure partially updated profile: %+v %v", got, err)
	}
}
