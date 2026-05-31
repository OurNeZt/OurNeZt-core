package calculation

import (
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
)

func TestHouseholdSummarySeparatesCurrentAndProjectedIncome(t *testing.T) {
	people := []domain.PersonProfile{
		{
			ID:                        "person_1",
			Age:                       25,
			EmploymentStatus:          domain.EmploymentFullTime,
			GrossMonthlyIncomeCents:   400000,
			ExpectedFutureIncomeCents: 450000,
			MonthlyExpensesCents:      120000,
			CPFOACents:                2000000,
		},
		{
			ID:                        "person_2",
			Age:                       24,
			EmploymentStatus:          domain.EmploymentStudent,
			GrossMonthlyIncomeCents:   90000,
			ExpectedFutureIncomeCents: 380000,
			MonthlyExpensesCents:      80000,
		},
	}

	got := CalculateHouseholdIncomeSummary(people)

	if got.CurrentGrossIncomeCents != 490000 {
		t.Fatalf("CurrentGrossIncomeCents = %d, want 490000", got.CurrentGrossIncomeCents)
	}
	if got.ProjectedGrossIncomeCents != 830000 {
		t.Fatalf("ProjectedGrossIncomeCents = %d, want 830000", got.ProjectedGrossIncomeCents)
	}
	if !got.MayNeedDeferredAssessment {
		t.Fatal("MayNeedDeferredAssessment = false, want true")
	}
}

func TestHouseholdSummaryIncludesStudentAndNsfGrossWithoutCPFDeduction(t *testing.T) {
	people := []domain.PersonProfile{
		{
			ID:                      "person_student",
			Age:                     22,
			EmploymentStatus:        domain.EmploymentStudent,
			GrossMonthlyIncomeCents: 120000,
		},
		{
			ID:                      "person_nsf",
			Age:                     20,
			EmploymentStatus:        domain.EmploymentFullTimeNSF,
			GrossMonthlyIncomeCents: 90000,
		},
		{
			ID:                      "person_part_time",
			Age:                     26,
			EmploymentStatus:        domain.EmploymentPartTime,
			GrossMonthlyIncomeCents: 140000,
		},
	}

	got := CalculateHouseholdIncomeSummary(people)

	if got.CurrentGrossIncomeCents != 350000 {
		t.Fatalf("CurrentGrossIncomeCents = %d, want 350000", got.CurrentGrossIncomeCents)
	}
	if got.EmployeeCPFCents != 0 {
		t.Fatalf("EmployeeCPFCents = %d, want 0", got.EmployeeCPFCents)
	}
	if got.TakeHomeIncomeCents != 350000 {
		t.Fatalf("TakeHomeIncomeCents = %d, want 350000", got.TakeHomeIncomeCents)
	}
}
