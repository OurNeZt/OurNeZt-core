package server

import (
	"context"

	"github.com/OurNeZt/ournezt-core/internal/calculation"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

type FinanceServer struct {
	ourneztv1.UnimplementedIncomeServiceServer
	ourneztv1.UnimplementedCPFServiceServer
}

func NewFinanceServer() FinanceServer {
	return FinanceServer{}
}

func (s FinanceServer) CalculateIncomeBreakdown(_ context.Context, req *ourneztv1.CalculateIncomeBreakdownRequest) (*ourneztv1.IncomeBreakdown, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	person, err := personFromProto(req.GetPerson())
	if err != nil {
		return nil, toStatusError(err)
	}

	breakdown := calculation.CalculateIncomeBreakdown(person)
	return incomeBreakdownToProto(breakdown), nil
}

func (s FinanceServer) CalculateHouseholdIncomeSummary(_ context.Context, req *ourneztv1.CalculateHouseholdIncomeSummaryRequest) (*ourneztv1.HouseholdIncomeSummary, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	people := make([]domain.PersonProfile, 0, len(req.GetPeople()))
	for _, personProto := range req.GetPeople() {
		person, err := personFromProto(personProto)
		if err != nil {
			return nil, toStatusError(err)
		}
		people = append(people, person)
	}

	summary := calculation.CalculateHouseholdIncomeSummary(people)
	return householdIncomeToProto(summary), nil
}

func (s FinanceServer) CalculateCPFContribution(_ context.Context, req *ourneztv1.CalculateCPFContributionRequest) (*ourneztv1.CPFContribution, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	contribution := calculation.CalculateCPFContribution(
		int(req.GetAge()),
		req.GetMonthlyWageCents(),
		domain.EmploymentStatus(req.GetEmploymentStatus()),
	)
	return cpfContributionToProto(contribution), nil
}
