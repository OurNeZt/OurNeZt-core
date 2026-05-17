package calculation

import "github.com/OurNeZt/ournezt-core/internal/domain"

type IncomeBreakdown struct {
	PersonID                  domain.ID
	CurrentGrossIncomeCents   int64
	ProjectedGrossIncomeCents int64
	EmployeeCPFCents          int64
	EmployerCPFCents          int64
	TakeHomeIncomeCents       int64
	MonthlyExpensesCents      int64
	MonthlySurplusCents       int64
	CPFContribution           CPFContribution
	CPFPlanningNote           string
}

type HouseholdIncomeSummary struct {
	CurrentGrossIncomeCents   int64
	ProjectedGrossIncomeCents int64
	TakeHomeIncomeCents       int64
	EmployeeCPFCents          int64
	EmployerCPFCents          int64
	MonthlyExpensesCents      int64
	MonthlySurplusCents       int64
	CurrentCPFOACents         int64
	CurrentCPFSACents         int64
	CurrentCPFMACents         int64
	MayNeedDeferredAssessment bool
}

func CalculateIncomeBreakdown(person domain.PersonProfile) IncomeBreakdown {
	currentIncome := currentIncomeFor(person)
	projectedIncome := projectedIncomeFor(person)
	cpf := CalculateCPFContribution(person.Age, currentIncome, person.EmploymentStatus)
	takeHome := currentIncome - cpf.EmployeeCents

	return IncomeBreakdown{
		PersonID:                  person.ID,
		CurrentGrossIncomeCents:   currentIncome,
		ProjectedGrossIncomeCents: projectedIncome,
		EmployeeCPFCents:          cpf.EmployeeCents,
		EmployerCPFCents:          cpf.EmployerCents,
		TakeHomeIncomeCents:       takeHome,
		MonthlyExpensesCents:      person.MonthlyExpensesCents,
		MonthlySurplusCents:       takeHome - person.MonthlyExpensesCents,
		CPFContribution:           cpf,
		CPFPlanningNote:           cpfPlanningNote(person.EmploymentStatus),
	}
}

func CalculateHouseholdIncomeSummary(people []domain.PersonProfile) HouseholdIncomeSummary {
	var summary HouseholdIncomeSummary
	for _, person := range people {
		breakdown := CalculateIncomeBreakdown(person)
		summary.CurrentGrossIncomeCents += breakdown.CurrentGrossIncomeCents
		summary.ProjectedGrossIncomeCents += breakdown.ProjectedGrossIncomeCents
		summary.TakeHomeIncomeCents += breakdown.TakeHomeIncomeCents
		summary.EmployeeCPFCents += breakdown.EmployeeCPFCents
		summary.EmployerCPFCents += breakdown.EmployerCPFCents
		summary.MonthlyExpensesCents += breakdown.MonthlyExpensesCents
		summary.MonthlySurplusCents += breakdown.MonthlySurplusCents
		summary.CurrentCPFOACents += person.CPFOACents
		summary.CurrentCPFSACents += person.CPFSACents
		summary.CurrentCPFMACents += person.CPFMACents
		if mayNeedDeferredAssessment(person.EmploymentStatus) {
			summary.MayNeedDeferredAssessment = true
		}
	}
	return summary
}

func currentIncomeFor(person domain.PersonProfile) int64 {
	switch person.EmploymentStatus {
	case domain.EmploymentFullTime, domain.EmploymentPartTime, domain.EmploymentSelfEmployed, domain.EmploymentFullTimeNSF, domain.EmploymentOther:
		return maxInt64(person.GrossMonthlyIncomeCents, 0)
	default:
		return 0
	}
}

func projectedIncomeFor(person domain.PersonProfile) int64 {
	if person.ExpectedFutureIncomeCents > 0 {
		return person.ExpectedFutureIncomeCents
	}
	return currentIncomeFor(person)
}

func mayNeedDeferredAssessment(status domain.EmploymentStatus) bool {
	return status == domain.EmploymentStudent || status == domain.EmploymentFullTimeNSF || status == domain.EmploymentFutureEmployee
}

func cpfPlanningNote(status domain.EmploymentStatus) string {
	switch status {
	case domain.EmploymentSelfEmployed:
		return "Self-employed CPF handling differs; treat this as a placeholder until MediSave rules are modelled."
	case domain.EmploymentStudent, domain.EmploymentFullTimeNSF, domain.EmploymentUnemployed, domain.EmploymentFutureEmployee:
		return "Normal employee CPF is not applied to this profile by default."
	default:
		return "CPF is estimated for ordinary wages using the configured planning rates."
	}
}
