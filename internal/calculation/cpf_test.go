package calculation

import (
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
)

func TestCalculateCPFContributionUnder55AtCeiling(t *testing.T) {
	got := CalculateCPFContribution(30, 900000, domain.EmploymentFullTime)

	if got.CPFApplicableWageCents != 800000 {
		t.Fatalf("CPFApplicableWageCents = %d, want 800000", got.CPFApplicableWageCents)
	}
	if got.EmployeeCents != 160000 {
		t.Fatalf("EmployeeCents = %d, want 160000", got.EmployeeCents)
	}
	if got.EmployerCents != 136000 {
		t.Fatalf("EmployerCents = %d, want 136000", got.EmployerCents)
	}
	if got.TotalCents != 296000 {
		t.Fatalf("TotalCents = %d, want 296000", got.TotalCents)
	}
}

func TestCalculateCPFContributionSkipsStudent(t *testing.T) {
	got := CalculateCPFContribution(24, 300000, domain.EmploymentStudent)

	if got.TotalCents != 0 {
		t.Fatalf("TotalCents = %d, want 0", got.TotalCents)
	}
}

func TestCalculateCPFContributionSkipsLowWage(t *testing.T) {
	got := CalculateCPFContribution(30, 5000, domain.EmploymentFullTime)

	if got.TotalCents != 0 {
		t.Fatalf("TotalCents = %d, want 0", got.TotalCents)
	}
}

func TestCalculateCPFContributionUsesOlderWorkerRates(t *testing.T) {
	got := CalculateCPFContribution(62, 400000, domain.EmploymentFullTime)

	if got.EmployeeCents != 50000 {
		t.Fatalf("EmployeeCents = %d, want 50000", got.EmployeeCents)
	}
	if got.EmployerCents != 50000 {
		t.Fatalf("EmployerCents = %d, want 50000", got.EmployerCents)
	}
	if got.TotalCents != 100000 {
		t.Fatalf("TotalCents = %d, want 100000", got.TotalCents)
	}
}
