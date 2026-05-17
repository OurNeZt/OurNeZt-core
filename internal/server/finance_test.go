package server

import (
	"context"
	"testing"

	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
)

func TestFinanceServerCalculateIncomeBreakdown(t *testing.T) {
	server := NewFinanceServer()

	response, err := server.CalculateIncomeBreakdown(context.Background(), &ourneztv1.CalculateIncomeBreakdownRequest{
		Person: &ourneztv1.PersonProfile{
			Id:                     "person_1",
			Age:                    30,
			EmploymentStatus:       "full_time_employee",
			GrossMonthlyIncomeCents: 500000,
			MonthlyExpensesCents:   150000,
		},
	})
	if err != nil {
		t.Fatalf("CalculateIncomeBreakdown returned error: %v", err)
	}
	if response.GetPersonId() != "person_1" {
		t.Fatalf("person id = %q, want person_1", response.GetPersonId())
	}
	if response.GetCurrentGrossIncomeCents() != 500000 {
		t.Fatalf("current gross income = %d, want 500000", response.GetCurrentGrossIncomeCents())
	}
}

func TestFinanceServerCalculateHouseholdIncomeSummary(t *testing.T) {
	server := NewFinanceServer()

	response, err := server.CalculateHouseholdIncomeSummary(context.Background(), &ourneztv1.CalculateHouseholdIncomeSummaryRequest{
		People: []*ourneztv1.PersonProfile{
			{
				Id:                     "person_1",
				Age:                    30,
				EmploymentStatus:       "full_time_employee",
				GrossMonthlyIncomeCents: 500000,
				CpfOaCents:             2000000,
				MonthlyExpensesCents:   150000,
			},
			{
				Id:                        "person_2",
				EmploymentStatus:          "student",
				ExpectedFutureIncomeCents: 300000,
			},
		},
	})
	if err != nil {
		t.Fatalf("CalculateHouseholdIncomeSummary returned error: %v", err)
	}
	if response.GetCurrentGrossIncomeCents() != 500000 {
		t.Fatalf("current gross income = %d, want 500000", response.GetCurrentGrossIncomeCents())
	}
	if !response.GetMayNeedDeferredAssessment() {
		t.Fatal("MayNeedDeferredAssessment = false, want true")
	}
}

func TestFinanceServerCalculateCPFContribution(t *testing.T) {
	server := NewFinanceServer()

	response, err := server.CalculateCPFContribution(context.Background(), &ourneztv1.CalculateCPFContributionRequest{
		Age:              30,
		MonthlyWageCents: 500000,
		EmploymentStatus: "full_time_employee",
	})
	if err != nil {
		t.Fatalf("CalculateCPFContribution returned error: %v", err)
	}
	if response.GetTotalCents() <= 0 {
		t.Fatalf("total cpf cents = %d, want > 0", response.GetTotalCents())
	}
}

