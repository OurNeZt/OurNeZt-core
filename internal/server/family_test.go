package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeFamiliesRepository struct {
	createdFamily      domain.Family
	createdOwnerID     domain.ID
	listFamilies       []domain.Family
	getFamily          domain.Family
	joinedFamily       domain.Family
	joinCodeHash       string
	generatedCodeHash  string
	generatedActorID   domain.ID
	generatedFamilyID  domain.ID
	generatedExpiresAt *time.Time
	members            []repository.FamilyMember
}

func (r *fakeFamiliesRepository) CreateFamily(_ context.Context, family domain.Family, ownerID domain.ID) (domain.Family, error) {
	r.createdFamily = family
	r.createdOwnerID = ownerID
	return domain.Family{ID: "family_1", Name: family.Name, Type: family.Type}, nil
}

func (r *fakeFamiliesRepository) GetFamily(_ context.Context, _ domain.ID, _ domain.ID) (domain.Family, error) {
	return r.getFamily, nil
}

func (r *fakeFamiliesRepository) ListUserFamilies(_ context.Context, _ domain.ID) ([]domain.Family, error) {
	return r.listFamilies, nil
}

func (r *fakeFamiliesRepository) JoinFamilyByCode(_ context.Context, _ domain.ID, codeHash string) (domain.Family, error) {
	r.joinCodeHash = codeHash
	return r.joinedFamily, nil
}

func (r *fakeFamiliesRepository) GenerateFamilyJoinCode(_ context.Context, actorID domain.ID, familyID domain.ID, codeHash string, expiresAt *time.Time) error {
	r.generatedActorID = actorID
	r.generatedFamilyID = familyID
	r.generatedCodeHash = codeHash
	r.generatedExpiresAt = expiresAt
	return nil
}

func (r *fakeFamiliesRepository) ListFamilyMembers(_ context.Context, _ domain.ID, _ domain.ID) ([]repository.FamilyMember, error) {
	return r.members, nil
}

func TestFamilyServerCreateFamily(t *testing.T) {
	repo := &fakeFamiliesRepository{}
	server := NewFamilyServer(repo, time.Hour, time.Now)

	response, err := server.CreateFamily(context.Background(), &ourneztv1.CreateFamilyRequest{
		OwnerUserId: "user_1",
		Name:        "Home Team",
		FamilyType:  "couple",
	})
	if err != nil {
		t.Fatalf("CreateFamily returned error: %v", err)
	}
	if response.GetId() != "family_1" {
		t.Fatalf("family id = %q, want family_1", response.GetId())
	}
	if repo.createdOwnerID != "user_1" {
		t.Fatalf("owner id = %q, want user_1", repo.createdOwnerID)
	}
	if repo.createdFamily.Name != "Home Team" {
		t.Fatalf("family name = %q, want Home Team", repo.createdFamily.Name)
	}
}

func TestFamilyServerGenerateJoinCodeHashesBeforePersisting(t *testing.T) {
	repo := &fakeFamiliesRepository{}
	now := time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC)
	server := NewFamilyServer(repo, 24*time.Hour, func() time.Time { return now })

	response, err := server.GenerateFamilyJoinCode(context.Background(), &ourneztv1.GenerateFamilyJoinCodeRequest{
		ActorUserId: "user_1",
		FamilyId:    "family_1",
	})
	if err != nil {
		t.Fatalf("GenerateFamilyJoinCode returned error: %v", err)
	}
	if len(response.GetCode()) != 10 {
		t.Fatalf("code length = %d, want 10", len(response.GetCode()))
	}
	if strings.ContainsAny(response.GetCode(), "01IO") {
		t.Fatalf("code contains ambiguous characters: %q", response.GetCode())
	}
	if repo.generatedCodeHash != security.HashToken(response.GetCode()) {
		t.Fatalf("stored code hash = %q, want hash of %q", repo.generatedCodeHash, response.GetCode())
	}
	wantExpiry := now.Add(24 * time.Hour)
	if repo.generatedExpiresAt == nil || !repo.generatedExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expires at = %v, want %v", repo.generatedExpiresAt, wantExpiry)
	}
}

func TestFamilyServerJoinFamilyByCodeHashesInput(t *testing.T) {
	repo := &fakeFamiliesRepository{
		joinedFamily: domain.Family{ID: "family_9", Name: "Joined", Type: domain.FamilyTypeFamily},
	}
	server := NewFamilyServer(repo, time.Hour, time.Now)

	response, err := server.JoinFamilyByCode(context.Background(), &ourneztv1.JoinFamilyByCodeRequest{
		UserId: "user_2",
		Code:   "ABCD2345",
	})
	if err != nil {
		t.Fatalf("JoinFamilyByCode returned error: %v", err)
	}
	if response.GetId() != "family_9" {
		t.Fatalf("family id = %q, want family_9", response.GetId())
	}
	if repo.joinCodeHash != security.HashToken("ABCD2345") {
		t.Fatalf("join code hash = %q, want hash of code", repo.joinCodeHash)
	}
}

func TestFamilyServerListFamilyMembers(t *testing.T) {
	repo := &fakeFamiliesRepository{
		members: []repository.FamilyMember{
			{FamilyID: "family_1", UserID: "user_1", DisplayName: "Alex", Role: domain.FamilyRoleOwner},
			{FamilyID: "family_1", UserID: "user_2", DisplayName: "Sam", Role: domain.FamilyRoleMember},
		},
	}
	server := NewFamilyServer(repo, time.Hour, time.Now)

	response, err := server.ListFamilyMembers(context.Background(), &ourneztv1.ListFamilyMembersRequest{
		ActorUserId: "user_1",
		FamilyId:    "family_1",
	})
	if err != nil {
		t.Fatalf("ListFamilyMembers returned error: %v", err)
	}
	if len(response.GetMembers()) != 2 {
		t.Fatalf("members len = %d, want 2", len(response.GetMembers()))
	}
	if response.GetMembers()[0].GetDisplayName() != "Alex" {
		t.Fatalf("first member display name = %q, want Alex", response.GetMembers()[0].GetDisplayName())
	}
}

func TestFamilyServerCreateFamilyRequiresOwnerID(t *testing.T) {
	server := NewFamilyServer(&fakeFamiliesRepository{}, time.Hour, time.Now)

	_, err := server.CreateFamily(context.Background(), &ourneztv1.CreateFamilyRequest{Name: "Home", FamilyType: "family"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want InvalidArgument", status.Code(err))
	}
}

var _ repository.Families = (*fakeFamiliesRepository)(nil)
