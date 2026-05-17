package calculation

import (
	"math"

	"github.com/OurNeZt/ournezt-core/internal/domain"
)

type HousingAffordability struct {
	HousingOptionID                 domain.ID
	NetPurchasePriceCents           int64
	RequiredDownpaymentCents        int64
	UpfrontCostCents                int64
	EstimatedLoanAmountCents        int64
	MonthlyMortgageCents            int64
	CPFUsedCents                    int64
	CashTopUpRequiredCents          int64
	RemainingCashCents              int64
	RemainingCPFOACents             int64
	MonthlyHousingCostCents         int64
	MonthlySurplusAfterHousingCents int64
	Rating                          domain.AffordabilityRating
}

type HouseholdAssets struct {
	CashSavingsCents     int64
	CPFOACents           int64
	TakeHomeCents        int64
	MonthlyExpensesCents int64
}

func CalculateHousingAffordability(option domain.HousingOption, assets HouseholdAssets) HousingAffordability {
	netPrice := maxInt64(option.PurchasePriceCents-option.GrantAmountCents, 0)
	downpayment := centsByBps(netPrice, option.DownpaymentPercentBps)
	upfront := downpayment + option.RenovationBudgetCents + option.FurnitureBudgetCents + option.LegalFeesCents + option.BuyerStampDutyCents

	loan := option.LoanAmountCents
	if loan <= 0 {
		loan = maxInt64(netPrice-downpayment, 0)
	}

	monthlyMortgage := monthlyPayment(loan, option.InterestRateBps, option.LoanTenureMonths)
	cpfUsed := minInt64(assets.CPFOACents, downpayment)
	cashTopUp := maxInt64(upfront-cpfUsed, 0)
	monthlyHousingCost := monthlyMortgage + option.MonthlyMaintenanceCents
	monthlySurplus := assets.TakeHomeCents - assets.MonthlyExpensesCents - monthlyHousingCost

	return HousingAffordability{
		HousingOptionID:                 option.ID,
		NetPurchasePriceCents:           netPrice,
		RequiredDownpaymentCents:        downpayment,
		UpfrontCostCents:                upfront,
		EstimatedLoanAmountCents:        loan,
		MonthlyMortgageCents:            monthlyMortgage,
		CPFUsedCents:                    cpfUsed,
		CashTopUpRequiredCents:          cashTopUp,
		RemainingCashCents:              assets.CashSavingsCents - cashTopUp,
		RemainingCPFOACents:             assets.CPFOACents - cpfUsed,
		MonthlyHousingCostCents:         monthlyHousingCost,
		MonthlySurplusAfterHousingCents: monthlySurplus,
		Rating:                          affordabilityRating(monthlyHousingCost, assets.TakeHomeCents, monthlySurplus),
	}
}

func monthlyPayment(principalCents, annualRateBps int64, months int) int64 {
	if principalCents <= 0 || months <= 0 {
		return 0
	}
	if annualRateBps <= 0 {
		return principalCents / int64(months)
	}

	monthlyRate := (float64(annualRateBps) / 10000) / 12
	principal := float64(principalCents)
	payment := principal * (monthlyRate * math.Pow(1+monthlyRate, float64(months))) / (math.Pow(1+monthlyRate, float64(months)) - 1)
	return int64(math.Round(payment))
}

func affordabilityRating(monthlyHousingCostCents, takeHomeCents, monthlySurplusCents int64) domain.AffordabilityRating {
	if takeHomeCents <= 0 || monthlySurplusCents < 0 {
		return domain.AffordabilityNotRecommended
	}

	ratioBps := monthlyHousingCostCents * 10000 / takeHomeCents
	switch {
	case ratioBps <= 2500:
		return domain.AffordabilityComfortable
	case ratioBps <= 3500:
		return domain.AffordabilityManageable
	case ratioBps <= 4500:
		return domain.AffordabilityTight
	case ratioBps <= 5500:
		return domain.AffordabilityRisky
	default:
		return domain.AffordabilityNotRecommended
	}
}
