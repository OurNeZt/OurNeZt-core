package server

import (
	"context"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/repository"
)

type fakePeopleRepository struct {
	createInput domain.PersonProfile
	created     domain.PersonProfile
	got         domain.PersonProfile
	list        []domain.PersonProfile
	updated     domain.PersonProfile
	deletedID   domain.ID
}

func (r *fakePeopleRepository) CreatePersonProfile(_ context.Context, profile domain.PersonProfile, _ domain.ID) (domain.PersonProfile, error) {
	r.createInput = profile
	return r.created, nil
}

func (r *fakePeopleRepository) GetPersonProfile(_ context.Context, _ domain.ID, _ domain.ID) (domain.PersonProfile, error) {
	return r.got, nil
}

func (r *fakePeopleRepository) ListPersonProfilesByFamily(_ context.Context, _ domain.ID, _ domain.ID) ([]domain.PersonProfile, error) {
	return r.list, nil
}

func (r *fakePeopleRepository) UpdatePersonProfile(_ context.Context, profile domain.PersonProfile, _ domain.ID) (domain.PersonProfile, error) {
	r.createInput = profile
	return r.updated, nil
}

func (r *fakePeopleRepository) DeletePersonProfile(_ context.Context, personID domain.ID, _ domain.ID) error {
	r.deletedID = personID
	return nil
}

func TestPersonServerCreateProfileParsesDates(t *testing.T) {
	repo := &fakePeopleRepository{
		created: domain.PersonProfile{ID: "person_1", FamilyID: "family_1", Name: "Alex", ExpectedIncomeStartDate: datePtr("2026-09-01")},
	}
	server := NewPersonServer(repo)

	response, err := server.CreatePersonProfile(context.Background(), &ourneztv1.PersonProfile{
		FamilyId:                "family_1",
		Name:                    "Alex",
		EmploymentStatus:        "student",
		ExpectedIncomeStartDate: "2026-09-01",
	})
	if err != nil {
		t.Fatalf("CreatePersonProfile returned error: %v", err)
	}
	if repo.createInput.ExpectedIncomeStartDate == nil || repo.createInput.ExpectedIncomeStartDate.Format("2006-01-02") != "2026-09-01" {
		t.Fatalf("expected income start date parsed incorrectly: %v", repo.createInput.ExpectedIncomeStartDate)
	}
	if response.GetId() != "person_1" {
		t.Fatalf("response id = %q, want person_1", response.GetId())
	}
}

func TestPersonServerListProfiles(t *testing.T) {
	repo := &fakePeopleRepository{
		list: []domain.PersonProfile{
			{ID: "person_1", FamilyID: "family_1", Name: "Alex"},
			{ID: "person_2", FamilyID: "family_1", Name: "Sam"},
		},
	}
	server := NewPersonServer(repo)

	response, err := server.ListPersonProfilesByFamily(context.Background(), &ourneztv1.ListPersonProfilesByFamilyRequest{
		ViewerUserId: "user_1",
		FamilyId:     "family_1",
	})
	if err != nil {
		t.Fatalf("ListPersonProfilesByFamily returned error: %v", err)
	}
	if len(response.GetPeople()) != 2 {
		t.Fatalf("people len = %d, want 2", len(response.GetPeople()))
	}
}

func TestPersonServerDeleteProfile(t *testing.T) {
	repo := &fakePeopleRepository{}
	server := NewPersonServer(repo)

	_, err := server.DeletePersonProfile(context.Background(), &ourneztv1.DeletePersonProfileRequest{
		ActorUserId: "user_1",
		PersonId:    "person_9",
	})
	if err != nil {
		t.Fatalf("DeletePersonProfile returned error: %v", err)
	}
	if repo.deletedID != "person_9" {
		t.Fatalf("deleted id = %q, want person_9", repo.deletedID)
	}
}

var _ repository.People = (*fakePeopleRepository)(nil)

func datePtr(v string) *time.Time {
	parsed, _ := time.Parse("2006-01-02", v)
	return &parsed
}
