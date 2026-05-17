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
			ExpectedFutureIncomeCents: 380000,
			MonthlyExpensesCents:      80000,
		},
	}

	got := CalculateHouseholdIncomeSummary(people)

	if got.CurrentGrossIncomeCents != 400000 {
		t.Fatalf("CurrentGrossIncomeCents = %d, want 400000", got.CurrentGrossIncomeCents)
	}
	if got.ProjectedGrossIncomeCents != 830000 {
		t.Fatalf("ProjectedGrossIncomeCents = %d, want 830000", got.ProjectedGrossIncomeCents)
	}
	if !got.MayNeedDeferredAssessment {
		t.Fatal("MayNeedDeferredAssessment = false, want true")
	}
}
