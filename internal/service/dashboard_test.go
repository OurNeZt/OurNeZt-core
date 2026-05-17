package service

import (
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
)

func TestBuildHouseholdDashboardAggregatesIncomeCashAndHousing(t *testing.T) {
	people := []domain.PersonProfile{
		{
			ID:                      "person_1",
			FamilyID:                "family_1",
			Age:                     30,
			EmploymentStatus:        domain.EmploymentFullTime,
			GrossMonthlyIncomeCents: 500000,
			CashSavingsCents:        2000000,
			CPFOACents:              3000000,
			MonthlyExpensesCents:    150000,
		},
		{
			ID:                        "person_2",
			FamilyID:                  "family_1",
			Age:                       25,
			EmploymentStatus:          domain.EmploymentFutureEmployee,
			ExpectedFutureIncomeCents: 420000,
			CashSavingsCents:          1000000,
			MonthlyExpensesCents:      50000,
		},
	}
	housing := []domain.HousingOption{
		{
			ID:                    "housing_1",
			FamilyID:              "family_1",
			PurchasePriceCents:    45000000,
			LoanType:              domain.LoanTypeHDB,
			InterestRateBps:       260,
			LoanTenureMonths:      300,
			DownpaymentPercentBps: 2000,
		},
	}

	got := BuildHouseholdDashboard("family_1", people, housing)

	if got.FamilyID != "family_1" {
		t.Fatalf("FamilyID = %q, want family_1", got.FamilyID)
	}
	if got.CashSavingsCents != 3000000 {
		t.Fatalf("CashSavingsCents = %d, want 3000000", got.CashSavingsCents)
	}
	if got.Income.CurrentGrossIncomeCents != 500000 {
		t.Fatalf("CurrentGrossIncomeCents = %d, want 500000", got.Income.CurrentGrossIncomeCents)
	}
	if got.Income.ProjectedGrossIncomeCents != 920000 {
		t.Fatalf("ProjectedGrossIncomeCents = %d, want 920000", got.Income.ProjectedGrossIncomeCents)
	}
	if !got.Income.MayNeedDeferredAssessment {
		t.Fatal("MayNeedDeferredAssessment = false, want true")
	}
	if len(got.HousingAffordability) != 1 {
		t.Fatalf("len(HousingAffordability) = %d, want 1", len(got.HousingAffordability))
	}
	if got.HousingAffordability[0].HousingOptionID != "housing_1" {
		t.Fatalf("HousingOptionID = %q, want housing_1", got.HousingAffordability[0].HousingOptionID)
	}
}
