package calculation

import (
	"testing"
	"time"

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
	if got.InitialDownpaymentCents != 2500000 {
		t.Fatalf("InitialDownpaymentCents = %d, want 2500000", got.InitialDownpaymentCents)
	}
	if got.FinalDownpaymentCents != 6500000 {
		t.Fatalf("FinalDownpaymentCents = %d, want 6500000", got.FinalDownpaymentCents)
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

func TestCalculateHousingAffordabilityExcludesRenovationAndFurnitureFromUpfrontCost(t *testing.T) {
	option := domain.HousingOption{
		ID:                    "housing_1",
		PurchasePriceCents:    50000000,
		GrantAmountCents:      5000000,
		DownpaymentPercentBps: 2000,
		LegalFeesCents:        1500000,
		BuyerStampDutyCents:   1750000,
		RenovationBudgetCents: 8000000,
		FurnitureBudgetCents:  3000000,
	}

	got := CalculateHousingAffordability(option, HouseholdAssets{})

	if got.UpfrontCostCents != 12250000 {
		t.Fatalf("UpfrontCostCents = %d, want 12250000", got.UpfrontCostCents)
	}
}

func TestCalculateHousingAffordabilityUsesDeferredPlanningAssumptions(t *testing.T) {
	keyDate := time.Date(2030, time.June, 1, 0, 0, 0, 0, time.UTC)
	option := domain.HousingOption{
		ID:                        "housing_1",
		Type:                      domain.HousingTypeBTO,
		PurchasePriceCents:        50000000,
		LoanType:                  domain.LoanTypeBank,
		LoanAmountCents:           0,
		InterestRateBps:           260,
		LoanTenureMonths:          300,
		DownpaymentPercentBps:     2500,
		ExpectedKeyCollectionDate: &keyDate,
	}

	got := CalculateHousingAffordability(option, HouseholdAssets{
		CashSavingsCents:     4000000,
		CPFOACents:           2000000,
		TakeHomeCents:        650000,
		MonthlyExpensesCents: 250000,
	})

	if got.RequiredDownpaymentCents != 12500000 {
		t.Fatalf("RequiredDownpaymentCents = %d, want 12500000", got.RequiredDownpaymentCents)
	}
	if got.InitialDownpaymentCents != 1250000 {
		t.Fatalf("InitialDownpaymentCents = %d, want 1250000", got.InitialDownpaymentCents)
	}
	if got.FinalDownpaymentCents != 11250000 {
		t.Fatalf("FinalDownpaymentCents = %d, want 11250000", got.FinalDownpaymentCents)
	}
	if got.EstimatedLoanAmountCents != 37500000 {
		t.Fatalf("EstimatedLoanAmountCents = %d, want 37500000", got.EstimatedLoanAmountCents)
	}
}

func TestCalculateHousingAffordabilityDoesNotUseDeferredAssumptionsForResaleHDB(t *testing.T) {
	keyDate := time.Date(2030, time.June, 1, 0, 0, 0, 0, time.UTC)
	option := domain.HousingOption{
		ID:                        "housing_1",
		Type:                      domain.HousingTypeResaleHDB,
		PurchasePriceCents:        50000000,
		LoanType:                  domain.LoanTypeBank,
		LoanAmountCents:           0,
		InterestRateBps:           260,
		LoanTenureMonths:          300,
		DownpaymentPercentBps:     2500,
		ExpectedKeyCollectionDate: &keyDate,
	}

	got := CalculateHousingAffordability(option, HouseholdAssets{})

	if got.InitialDownpaymentCents != 2500000 {
		t.Fatalf("InitialDownpaymentCents = %d, want 2500000", got.InitialDownpaymentCents)
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

func TestEstimateHousingGrantAmountUsesHouseholdGrossIncomeChart(t *testing.T) {
	people := []domain.PersonProfile{
		{ID: "person_1", GrossMonthlyIncomeCents: 320000},
		{ID: "person_2", GrossMonthlyIncomeCents: 180000},
	}

	got := EstimateHousingGrantAmount(domain.HousingTypeBTO, people)

	if got.HouseholdGrossMonthlyIncomeCents != 500000 {
		t.Fatalf("HouseholdGrossMonthlyIncomeCents = %d, want 500000", got.HouseholdGrossMonthlyIncomeCents)
	}
	if !got.Eligible {
		t.Fatalf("Eligible = false, want true")
	}
	if got.GrantAmountCents != 6500000 {
		t.Fatalf("GrantAmountCents = %d, want 6500000", got.GrantAmountCents)
	}
}

func TestEstimateHousingGrantAmountReturnsZeroForIneligibleHousingType(t *testing.T) {
	people := []domain.PersonProfile{
		{ID: "person_1", GrossMonthlyIncomeCents: 250000},
		{ID: "person_2", GrossMonthlyIncomeCents: 250000},
	}

	for _, housingType := range []domain.HousingType{domain.HousingTypeExecutive, domain.HousingTypePrivate, domain.HousingTypeLanded, domain.HousingTypeOther} {
		t.Run(string(housingType), func(t *testing.T) {
			got := EstimateHousingGrantAmount(housingType, people)

			if got.Eligible {
				t.Fatalf("Eligible = true, want false")
			}
			if got.GrantAmountCents != 0 {
				t.Fatalf("GrantAmountCents = %d, want 0", got.GrantAmountCents)
			}
		})
	}
}
