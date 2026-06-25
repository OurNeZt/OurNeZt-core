package server

import (
	"context"
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	peopleRepo := &fakePeopleRepository{
		list: []domain.PersonProfile{
			{ID: "person_1", FamilyID: "family_1", GrossMonthlyIncomeCents: 320000},
			{ID: "person_2", FamilyID: "family_1", GrossMonthlyIncomeCents: 180000},
		},
	}
	server := NewHousingServer(repo, peopleRepo)

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
	if repo.createInput.GrantAmountCents != 6500000 {
		t.Fatalf("create input grant amount = %d, want 6500000", repo.createInput.GrantAmountCents)
	}
}

func TestHousingServerCalculateAffordability(t *testing.T) {
	server := NewHousingServer(&fakeHousingRepository{}, &fakePeopleRepository{})

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
	if response.GetInitialDownpaymentCents() != 2250000 {
		t.Fatalf("initial downpayment = %d, want 2250000", response.GetInitialDownpaymentCents())
	}
	if response.GetFinalDownpaymentCents() != 6750000 {
		t.Fatalf("final downpayment = %d, want 6750000", response.GetFinalDownpaymentCents())
	}
}

func TestHousingServerRejectsLoanTenureAboveHousingTypeCap(t *testing.T) {
	server := NewHousingServer(&fakeHousingRepository{}, &fakePeopleRepository{})

	tests := []struct {
		name          string
		housingType   string
		tenureMonths  int32
		wantErrorCode codes.Code
	}{
		{
			name:          "HDB above 30 years",
			housingType:   "bto",
			tenureMonths:  31 * 12,
			wantErrorCode: codes.InvalidArgument,
		},
		{
			name:          "non-HDB above 35 years",
			housingType:   "private_condo",
			tenureMonths:  36 * 12,
			wantErrorCode: codes.InvalidArgument,
		},
		{
			name:         "non-HDB at 35 years",
			housingType:  "private_condo",
			tenureMonths: 35 * 12,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.CalculateHousingAffordability(context.Background(), &ourneztv1.CalculateHousingAffordabilityRequest{
				HousingOption: &ourneztv1.HousingOption{
					Id:                    "housing_1",
					FamilyId:              "family_1",
					Name:                  "Option",
					HousingType:           tc.housingType,
					LoanType:              "bank",
					PurchasePriceCents:    45000000,
					InterestRateBps:       260,
					LoanTenureMonths:      tc.tenureMonths,
					DownpaymentPercentBps: 2500,
				},
				TakeHomeCents: 650000,
			})
			if tc.wantErrorCode == codes.OK {
				if err != nil {
					t.Fatalf("CalculateHousingAffordability returned error: %v", err)
				}
				return
			}
			if status.Code(err) != tc.wantErrorCode {
				t.Fatalf("status code = %v, want %v", status.Code(err), tc.wantErrorCode)
			}
		})
	}
}

func TestHousingServerDeleteOption(t *testing.T) {
	repo := &fakeHousingRepository{}
	server := NewHousingServer(repo, &fakePeopleRepository{})

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

func TestHousingServerEstimateHousingGrant(t *testing.T) {
	server := NewHousingServer(&fakeHousingRepository{}, &fakePeopleRepository{
		list: []domain.PersonProfile{
			{ID: "person_1", FamilyID: "family_1", GrossMonthlyIncomeCents: 300000},
			{ID: "person_2", FamilyID: "family_1", GrossMonthlyIncomeCents: 200000},
		},
	})

	response, err := server.EstimateHousingGrant(context.Background(), &ourneztv1.EstimateHousingGrantRequest{
		ViewerUserId: "user_1",
		FamilyId:     "family_1",
		HousingType:  "bto",
	})
	if err != nil {
		t.Fatalf("EstimateHousingGrant returned error: %v", err)
	}
	if response.GetHouseholdGrossMonthlyIncomeCents() != 500000 {
		t.Fatalf("household gross = %d, want 500000", response.GetHouseholdGrossMonthlyIncomeCents())
	}
	if response.GetGrantAmountCents() != 6500000 {
		t.Fatalf("grant amount = %d, want 6500000", response.GetGrantAmountCents())
	}
	if !response.GetEligible() {
		t.Fatalf("eligible = false, want true")
	}
}

var _ repository.Housing = (*fakeHousingRepository)(nil)
