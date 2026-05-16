package calculation

import "github.com/OurNeZt/ournezt-core/internal/domain"

const cpfOrdinaryWageCeilingCents2026 int64 = 800000

type CPFContribution struct {
	CPFApplicableWageCents int64
	EmployeeCents          int64
	EmployerCents          int64
	TotalCents             int64
	OrdinaryAccountCents   int64
	SpecialAccountCents    int64
	MedisaveAccountCents   int64
}

type cpfRate struct {
	employerBps int64
	employeeBps int64
}

func CalculateCPFContribution(age int, monthlyWageCents int64, status domain.EmploymentStatus) CPFContribution {
	if !normalEmployeeCPFApplies(status) || monthlyWageCents <= 5000 {
		return CPFContribution{}
	}

	rate := cpfRateForAge(age)
	applicableWage := minInt64(monthlyWageCents, cpfOrdinaryWageCeilingCents2026)

	var employee int64
	var total int64
	switch {
	case applicableWage <= 50000:
		total = centsByBps(applicableWage, rate.employerBps)
	case applicableWage <= 75000:
		employee = centsByBps(applicableWage-50000, 1500)
		total = centsByBps(applicableWage, rate.employerBps) + employee
	default:
		employee = centsByBps(applicableWage, rate.employeeBps)
		total = centsByBps(applicableWage, rate.employeeBps+rate.employerBps)
	}

	employee = floorToDollar(employee)
	total = roundToNearestDollar(total)
	employer := maxInt64(total-employee, 0)

	oa, sa, ma := estimateCPFAllocation(age, total)
	return CPFContribution{
		CPFApplicableWageCents: applicableWage,
		EmployeeCents:          employee,
		EmployerCents:          employer,
		TotalCents:             total,
		OrdinaryAccountCents:   oa,
		SpecialAccountCents:    sa,
		MedisaveAccountCents:   ma,
	}
}

func normalEmployeeCPFApplies(status domain.EmploymentStatus) bool {
	return status == domain.EmploymentFullTime || status == domain.EmploymentPartTime
}

func cpfRateForAge(age int) cpfRate {
	switch {
	case age <= 55:
		return cpfRate{employerBps: 1700, employeeBps: 2000}
	case age <= 60:
		return cpfRate{employerBps: 1600, employeeBps: 1800}
	case age <= 65:
		return cpfRate{employerBps: 1250, employeeBps: 1250}
	case age <= 70:
		return cpfRate{employerBps: 900, employeeBps: 750}
	default:
		return cpfRate{employerBps: 750, employeeBps: 500}
	}
}

func estimateCPFAllocation(age int, totalCents int64) (oa int64, sa int64, ma int64) {
	switch {
	case age <= 35:
		return centsByBps(totalCents, 6216), centsByBps(totalCents, 1621), totalCents - centsByBps(totalCents, 6216) - centsByBps(totalCents, 1621)
	case age <= 45:
		return centsByBps(totalCents, 5677), centsByBps(totalCents, 1892), totalCents - centsByBps(totalCents, 5677) - centsByBps(totalCents, 1892)
	case age <= 50:
		return centsByBps(totalCents, 5136), centsByBps(totalCents, 2162), totalCents - centsByBps(totalCents, 5136) - centsByBps(totalCents, 2162)
	default:
		return centsByBps(totalCents, 4054), centsByBps(totalCents, 3108), totalCents - centsByBps(totalCents, 4054) - centsByBps(totalCents, 3108)
	}
}

func centsByBps(cents, bps int64) int64 {
	return cents * bps / 10000
}

func floorToDollar(cents int64) int64 {
	return cents / 100 * 100
}

func roundToNearestDollar(cents int64) int64 {
	dollars := cents / 100
	if cents%100 >= 50 {
		dollars++
	}
	return dollars * 100
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

