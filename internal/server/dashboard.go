package server

import (
	"context"

	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/repository"
	"github.com/OurNeZt/ournezt-core/internal/service"
)

type DashboardServer struct {
	ourneztv1.UnimplementedDashboardServiceServer
	people  repository.People
	housing repository.Housing
	auth    Authenticator
}

func NewDashboardServer(people repository.People, housing repository.Housing, auth ...Authenticator) DashboardServer {
	var authenticator Authenticator
	if len(auth) > 0 {
		authenticator = auth[0]
	}

	return DashboardServer{
		people:  people,
		housing: housing,
		auth:    authenticator,
	}
}

func (s DashboardServer) GetHouseholdDashboard(ctx context.Context, req *ourneztv1.GetHouseholdDashboardRequest) (*ourneztv1.HouseholdDashboard, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	viewerID, err := requestActorID(ctx, s.auth, req.GetViewerUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	familyID, err := requireID(req.GetFamilyId())
	if err != nil {
		return nil, toStatusError(err)
	}

	people, err := s.people.ListPersonProfilesByFamily(ctx, familyID, viewerID)
	if err != nil {
		return nil, toStatusError(err)
	}
	housingOptions, err := s.housing.ListHousingOptions(ctx, familyID, viewerID)
	if err != nil {
		return nil, toStatusError(err)
	}

	dashboard := service.BuildHouseholdDashboard(familyID, people, housingOptions)
	response := &ourneztv1.HouseholdDashboard{
		FamilyId:             string(dashboard.FamilyID),
		Income:               householdIncomeToProto(dashboard.Income),
		CashSavingsCents:     dashboard.CashSavingsCents,
		HousingAffordability: make([]*ourneztv1.HousingAffordability, 0, len(dashboard.HousingAffordability)),
	}
	for _, affordability := range dashboard.HousingAffordability {
		response.HousingAffordability = append(response.HousingAffordability, housingAffordabilityToProto(affordability))
	}
	return response, nil
}
