package server

import (
	"context"
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
)

func TestDashboardServerGetHouseholdDashboard(t *testing.T) {
	peopleRepo := &fakePeopleRepository{
		list: []domain.PersonProfile{
			{
				ID:                      "person_1",
				FamilyID:                "family_1",
				Age:                     30,
				EmploymentStatus:        domain.EmploymentFullTime,
				GrossMonthlyIncomeCents: 500000,
				CashSavingsCents:        2000000,
				CPFOACents:              2500000,
				MonthlyExpensesCents:    150000,
			},
		},
	}
	housingRepo := &fakeHousingRepository{
		list: []domain.HousingOption{
			{
				ID:                    "housing_1",
				FamilyID:              "family_1",
				Name:                  "BTO Plan A",
				PurchasePriceCents:    45000000,
				LoanType:              domain.LoanTypeHDB,
				InterestRateBps:       260,
				LoanTenureMonths:      300,
				DownpaymentPercentBps: 2000,
			},
		},
	}

	server := NewDashboardServer(peopleRepo, housingRepo)
	response, err := server.GetHouseholdDashboard(context.Background(), &ourneztv1.GetHouseholdDashboardRequest{
		ViewerUserId: "user_1",
		FamilyId:     "family_1",
	})
	if err != nil {
		t.Fatalf("GetHouseholdDashboard returned error: %v", err)
	}
	if response.GetFamilyId() != "family_1" {
		t.Fatalf("family id = %q, want family_1", response.GetFamilyId())
	}
	if response.GetIncome().GetCurrentGrossIncomeCents() != 500000 {
		t.Fatalf("current gross income = %d, want 500000", response.GetIncome().GetCurrentGrossIncomeCents())
	}
	if len(response.GetHousingAffordability()) != 1 {
		t.Fatalf("housing affordability len = %d, want 1", len(response.GetHousingAffordability()))
	}
}

