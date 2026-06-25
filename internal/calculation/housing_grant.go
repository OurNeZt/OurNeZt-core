package calculation

import "github.com/OurNeZt/ournezt-core/internal/domain"

const (
	hdbMaxLoanTenureYears    = 30
	nonHDBMaxLoanTenureYears = 35
	monthsPerYear            = 12
)

type HousingGrantEstimate struct {
	GrantAmountCents                 int64
	HouseholdGrossMonthlyIncomeCents int64
	Eligible                         bool
}

func EstimateHousingGrantAmount(housingType domain.HousingType, people []domain.PersonProfile) HousingGrantEstimate {
	if len(people) == 0 {
		return HousingGrantEstimate{
			GrantAmountCents:                 0,
			HouseholdGrossMonthlyIncomeCents: 0,
			Eligible:                         isHousingGrantEligibleType(housingType),
		}
	}

	summary := CalculateHouseholdIncomeSummary(people)
	return EstimateHousingGrantAmountFromGrossMonthlyIncome(housingType, summary.CurrentGrossIncomeCents)
}

func EstimateHousingGrantAmountFromGrossMonthlyIncome(housingType domain.HousingType, grossMonthlyIncomeCents int64) HousingGrantEstimate {
	estimate := HousingGrantEstimate{
		GrantAmountCents:                 0,
		HouseholdGrossMonthlyIncomeCents: maxInt64(grossMonthlyIncomeCents, 0),
		Eligible:                         isHousingGrantEligibleType(housingType),
	}
	if !estimate.Eligible {
		return estimate
	}

	income := estimate.HouseholdGrossMonthlyIncomeCents
	brackets := []struct {
		maxIncomeCents int64
		grantCents     int64
	}{
		{maxIncomeCents: 150000, grantCents: 12000000},
		{maxIncomeCents: 200000, grantCents: 11000000},
		{maxIncomeCents: 250000, grantCents: 10500000},
		{maxIncomeCents: 300000, grantCents: 9500000},
		{maxIncomeCents: 350000, grantCents: 9000000},
		{maxIncomeCents: 400000, grantCents: 8000000},
		{maxIncomeCents: 450000, grantCents: 7000000},
		{maxIncomeCents: 500000, grantCents: 6500000},
		{maxIncomeCents: 550000, grantCents: 5500000},
		{maxIncomeCents: 600000, grantCents: 5000000},
		{maxIncomeCents: 650000, grantCents: 4000000},
		{maxIncomeCents: 700000, grantCents: 3000000},
		{maxIncomeCents: 750000, grantCents: 2500000},
		{maxIncomeCents: 800000, grantCents: 2000000},
		{maxIncomeCents: 850000, grantCents: 1000000},
		{maxIncomeCents: 900000, grantCents: 500000},
	}

	for _, bracket := range brackets {
		if income <= bracket.maxIncomeCents {
			estimate.GrantAmountCents = bracket.grantCents
			return estimate
		}
	}

	return estimate
}

func IsDeferredHousingOption(option domain.HousingOption) bool {
	if option.LoanType == domain.LoanTypeCash {
		return false
	}
	if IsCondoHousingType(option.Type) {
		return false
	}

	hasValidKeyDate := option.ExpectedKeyCollectionDate != nil && !option.ExpectedKeyCollectionDate.IsZero()
	looksDeferredByLegacyShape := option.LoanAmountCents == 0 && option.DownpaymentPercentBps == 2500
	looksDeferredByKeyDate := hasValidKeyDate &&
		option.LoanAmountCents == 0 &&
		option.GrantAmountCents == 0 &&
		(option.LoanType == "" || option.LoanType == domain.LoanTypeBank || option.LoanType == domain.LoanTypeHDB)
	looksDeferredByDefaultedLoan := hasValidKeyDate &&
		option.LoanType == domain.LoanTypeBank &&
		option.LoanAmountCents == 0 &&
		(option.LoanTenureMonths == 300 || option.LoanTenureMonths == MaxLoanTenureMonthsForHousingType(option.Type))
	looksDeferredByOverrides := hasValidKeyDate && len(option.DIAIncomeOverrides) > 0

	return looksDeferredByLegacyShape || looksDeferredByKeyDate || looksDeferredByDefaultedLoan || looksDeferredByOverrides
}

func IsCondoHousingType(housingType domain.HousingType) bool {
	switch housingType {
	case domain.HousingTypeExecutive, domain.HousingTypePrivate:
		return true
	default:
		return false
	}
}

func MaxLoanTenureYearsForHousingType(housingType domain.HousingType) int {
	switch housingType {
	case domain.HousingTypeBTO, domain.HousingTypeResaleHDB:
		return hdbMaxLoanTenureYears
	default:
		return nonHDBMaxLoanTenureYears
	}
}

func MaxLoanTenureMonthsForHousingType(housingType domain.HousingType) int {
	return MaxLoanTenureYearsForHousingType(housingType) * monthsPerYear
}

func isHousingGrantEligibleType(housingType domain.HousingType) bool {
	switch housingType {
	case domain.HousingTypeBTO, domain.HousingTypeResaleHDB:
		return true
	default:
		return false
	}
}
