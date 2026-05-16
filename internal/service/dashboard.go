package service

import (
	"github.com/OurNeZt/ournezt-core/internal/calculation"
	"github.com/OurNeZt/ournezt-core/internal/domain"
)

type HouseholdDashboard struct {
	FamilyID              domain.ID
	Income                calculation.HouseholdIncomeSummary
	CashSavingsCents      int64
	HousingAffordability  []calculation.HousingAffordability
}

func BuildHouseholdDashboard(familyID domain.ID, people []domain.PersonProfile, housingOptions []domain.HousingOption) HouseholdDashboard {
	income := calculation.CalculateHouseholdIncomeSummary(people)

	var cash int64
	for _, person := range people {
		cash += person.CashSavingsCents
	}

	assets := calculation.HouseholdAssets{
		CashSavingsCents:    cash,
		CPFOACents:          income.CurrentCPFOACents,
		TakeHomeCents:       income.TakeHomeIncomeCents,
		MonthlyExpensesCents: income.MonthlyExpensesCents,
	}

	results := make([]calculation.HousingAffordability, 0, len(housingOptions))
	for _, option := range housingOptions {
		results = append(results, calculation.CalculateHousingAffordability(option, assets))
	}

	return HouseholdDashboard{
		FamilyID:             familyID,
		Income:               income,
		CashSavingsCents:     cash,
		HousingAffordability: results,
	}
}

