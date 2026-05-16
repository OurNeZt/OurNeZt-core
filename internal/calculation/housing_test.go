package calculation

import (
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
)

func TestCalculateHousingAffordability(t *testing.T) {
	option := domain.HousingOption{
		ID:                      "housing_1",
		PurchasePriceCents:      50000000,
		GrantAmountCents:        5000000,
		LoanType:                domain.LoanTypeHDB,
		InterestRateBps:         260,
		LoanTenureMonths:        300,
		DownpaymentPercentBps:   2000,
		MonthlyMaintenanceCents: 9000,
	}
	assets := HouseholdAssets{
		CashSavingsCents:     8000000,
		CPFOACents:           6000000,
		TakeHomeCents:        650000,
		MonthlyExpensesCents: 250000,
	}

	got := CalculateHousingAffordability(option, assets)

	if got.NetPurchasePriceCents != 45000000 {
		t.Fatalf("NetPurchasePriceCents = %d, want 45000000", got.NetPurchasePriceCents)
	}
	if got.RequiredDownpaymentCents != 9000000 {
		t.Fatalf("RequiredDownpaymentCents = %d, want 9000000", got.RequiredDownpaymentCents)
	}
	if got.MonthlyMortgageCents <= 0 {
		t.Fatalf("MonthlyMortgageCents = %d, want > 0", got.MonthlyMortgageCents)
	}
}

func TestCalculateHousingAffordabilityUsesExplicitLoanAmount(t *testing.T) {
	option := domain.HousingOption{
		ID:                    "housing_1",
		PurchasePriceCents:    50000000,
		LoanAmountCents:       30000000,
		LoanTenureMonths:      300,
		DownpaymentPercentBps: 2000,
	}
	assets := HouseholdAssets{
		CashSavingsCents:     20000000,
		CPFOACents:           10000000,
		TakeHomeCents:        700000,
		MonthlyExpensesCents: 200000,
	}

	got := CalculateHousingAffordability(option, assets)

	if got.EstimatedLoanAmountCents != 30000000 {
		t.Fatalf("EstimatedLoanAmountCents = %d, want 30000000", got.EstimatedLoanAmountCents)
	}
}

func TestCalculateHousingAffordabilityRejectsNoIncomeByRating(t *testing.T) {
	option := domain.HousingOption{
		ID:                    "housing_1",
		PurchasePriceCents:    50000000,
		LoanTenureMonths:      300,
		DownpaymentPercentBps: 2000,
	}

	got := CalculateHousingAffordability(option, HouseholdAssets{})

	if got.Rating != domain.AffordabilityNotRecommended {
		t.Fatalf("Rating = %q, want not_recommended", got.Rating)
	}
}
