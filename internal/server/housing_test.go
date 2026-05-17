package server

import (
	"context"
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/repository"
)

type fakeHousingRepository struct {
	createInput domain.HousingOption
	created     domain.HousingOption
	got         domain.HousingOption
	list        []domain.HousingOption
	updated     domain.HousingOption
	deletedID   domain.ID
}

func (r *fakeHousingRepository) CreateHousingOption(_ context.Context, option domain.HousingOption, _ domain.ID) (domain.HousingOption, error) {
	r.createInput = option
	return r.created, nil
}

func (r *fakeHousingRepository) GetHousingOption(_ context.Context, _ domain.ID, _ domain.ID) (domain.HousingOption, error) {
	return r.got, nil
}

func (r *fakeHousingRepository) ListHousingOptions(_ context.Context, _ domain.ID, _ domain.ID) ([]domain.HousingOption, error) {
	return r.list, nil
}

func (r *fakeHousingRepository) UpdateHousingOption(_ context.Context, option domain.HousingOption, _ domain.ID) (domain.HousingOption, error) {
	r.createInput = option
	return r.updated, nil
}

func (r *fakeHousingRepository) DeleteHousingOption(_ context.Context, housingID domain.ID, _ domain.ID) error {
	r.deletedID = housingID
	return nil
}

func TestHousingServerCreateOption(t *testing.T) {
	repo := &fakeHousingRepository{
		created: domain.HousingOption{ID: "housing_1", FamilyID: "family_1", Name: "BTO"},
	}
	server := NewHousingServer(repo)

	response, err := server.CreateHousingOption(context.Background(), &ourneztv1.HousingOption{
		FamilyId:              "family_1",
		Name:                  "BTO",
		HousingType:           "bto",
		LoanType:              "hdb",
		PurchasePriceCents:    45000000,
		InterestRateBps:       260,
		LoanTenureMonths:      300,
		DownpaymentPercentBps: 2000,
	})
	if err != nil {
		t.Fatalf("CreateHousingOption returned error: %v", err)
	}
	if response.GetId() != "housing_1" {
		t.Fatalf("id = %q, want housing_1", response.GetId())
	}
	if repo.createInput.Name != "BTO" {
		t.Fatalf("create input name = %q, want BTO", repo.createInput.Name)
	}
}

func TestHousingServerCalculateAffordability(t *testing.T) {
	server := NewHousingServer(&fakeHousingRepository{})

	response, err := server.CalculateHousingAffordability(context.Background(), &ourneztv1.CalculateHousingAffordabilityRequest{
		HousingOption: &ourneztv1.HousingOption{
			Id:                    "housing_1",
			FamilyId:              "family_1",
			Name:                  "BTO",
			HousingType:           "bto",
			LoanType:              "hdb",
			PurchasePriceCents:    45000000,
			InterestRateBps:       260,
			LoanTenureMonths:      300,
			DownpaymentPercentBps: 2000,
		},
		CashSavingsCents:     5000000,
		CpfOaCents:           3000000,
		TakeHomeCents:        650000,
		MonthlyExpensesCents: 200000,
	})
	if err != nil {
		t.Fatalf("CalculateHousingAffordability returned error: %v", err)
	}
	if response.GetHousingOptionId() != "housing_1" {
		t.Fatalf("housing option id = %q, want housing_1", response.GetHousingOptionId())
	}
	if response.GetMonthlyMortgageCents() <= 0 {
		t.Fatalf("monthly mortgage = %d, want > 0", response.GetMonthlyMortgageCents())
	}
}

func TestHousingServerDeleteOption(t *testing.T) {
	repo := &fakeHousingRepository{}
	server := NewHousingServer(repo)

	_, err := server.DeleteHousingOption(context.Background(), &ourneztv1.DeleteHousingOptionRequest{
		ActorUserId: "user_1",
		HousingId:   "housing_9",
	})
	if err != nil {
		t.Fatalf("DeleteHousingOption returned error: %v", err)
	}
	if repo.deletedID != "housing_9" {
		t.Fatalf("deleted id = %q, want housing_9", repo.deletedID)
	}
}

var _ repository.Housing = (*fakeHousingRepository)(nil)
