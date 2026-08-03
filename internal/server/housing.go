package server

import (
	"context"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/calculation"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/repository"
)

type HousingServer struct {
	ourneztv1.UnimplementedHousingServiceServer
	housing repository.Housing
	people  repository.People
	auth    Authenticator
}

func NewHousingServer(housing repository.Housing, people repository.People, auth ...Authenticator) HousingServer {
	var authenticator Authenticator
	if len(auth) > 0 {
		authenticator = auth[0]
	}

	return HousingServer{
		housing: housing,
		people:  people,
		auth:    authenticator,
	}
}

func (s HousingServer) CreateHousingOption(ctx context.Context, req *ourneztv1.HousingOption) (*ourneztv1.HousingOption, error) {
	option, err := housingFromProto(req)
	if err != nil {
		return nil, toStatusError(err)
	}
	if err := validateHousingOption(option); err != nil {
		return nil, toStatusError(err)
	}

	actorID, err := optionalAuthenticatedActorID(ctx, s.auth)
	if err != nil {
		return nil, toStatusError(err)
	}
	if option, err = s.applyGrantEstimate(ctx, option, actorID); err != nil {
		return nil, toStatusError(err)
	}

	created, err := s.housing.CreateHousingOption(ctx, option, actorID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return housingToProto(created), nil
}

func (s HousingServer) GetHousingOption(ctx context.Context, req *ourneztv1.GetHousingOptionRequest) (*ourneztv1.HousingOption, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	viewerID, err := requestActorID(ctx, s.auth, req.GetViewerUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	housingID, err := requireID(req.GetHousingId())
	if err != nil {
		return nil, toStatusError(err)
	}

	option, err := s.housing.GetHousingOption(ctx, housingID, viewerID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return housingToProto(option), nil
}

func (s HousingServer) ListHousingOptions(ctx context.Context, req *ourneztv1.ListHousingOptionsRequest) (*ourneztv1.ListHousingOptionsResponse, error) {
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

	options, err := s.housing.ListHousingOptions(ctx, familyID, viewerID)
	if err != nil {
		return nil, toStatusError(err)
	}

	response := &ourneztv1.ListHousingOptionsResponse{
		HousingOptions: make([]*ourneztv1.HousingOption, 0, len(options)),
	}
	for _, option := range options {
		response.HousingOptions = append(response.HousingOptions, housingToProto(option))
	}
	return response, nil
}

func (s HousingServer) UpdateHousingOption(ctx context.Context, req *ourneztv1.HousingOption) (*ourneztv1.HousingOption, error) {
	option, err := housingFromProto(req)
	if err != nil {
		return nil, toStatusError(err)
	}
	if option.ID == "" {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	if err := validateHousingOption(option); err != nil {
		return nil, toStatusError(err)
	}

	actorID, err := optionalAuthenticatedActorID(ctx, s.auth)
	if err != nil {
		return nil, toStatusError(err)
	}
	if req.HousingGroupId == nil || req.VisibleOnDashboard == nil {
		existing, getErr := s.housing.GetHousingOption(ctx, option.ID, actorID)
		if getErr != nil {
			return nil, toStatusError(getErr)
		}
		if req.HousingGroupId == nil {
			option.GroupID = existing.GroupID
		}
		if req.VisibleOnDashboard == nil {
			option.VisibleOnDashboard = existing.VisibleOnDashboard
		}
	}
	if option, err = s.applyGrantEstimate(ctx, option, actorID); err != nil {
		return nil, toStatusError(err)
	}

	updated, err := s.housing.UpdateHousingOption(ctx, option, actorID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return housingToProto(updated), nil
}

func (s HousingServer) DeleteHousingOption(ctx context.Context, req *ourneztv1.DeleteHousingOptionRequest) (*ourneztv1.DeleteHousingOptionResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requestActorID(ctx, s.auth, req.GetActorUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	housingID, err := requireID(req.GetHousingId())
	if err != nil {
		return nil, toStatusError(err)
	}

	if err := s.housing.DeleteHousingOption(ctx, housingID, actorID); err != nil {
		return nil, toStatusError(err)
	}
	return &ourneztv1.DeleteHousingOptionResponse{}, nil
}

func (s HousingServer) CreateHousingGroup(ctx context.Context, req *ourneztv1.HousingGroup) (*ourneztv1.HousingGroup, error) {
	group, err := housingGroupFromProto(req)
	if err != nil {
		return nil, toStatusError(err)
	}
	if strings.TrimSpace(string(group.FamilyID)) == "" || strings.TrimSpace(group.Name) == "" {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	actorID, err := optionalAuthenticatedActorID(ctx, s.auth)
	if err != nil {
		return nil, toStatusError(err)
	}

	created, err := s.housing.CreateHousingGroup(ctx, group, actorID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return housingGroupToProto(created), nil
}

func (s HousingServer) ListHousingGroups(ctx context.Context, req *ourneztv1.ListHousingGroupsRequest) (*ourneztv1.ListHousingGroupsResponse, error) {
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

	groups, err := s.housing.ListHousingGroups(ctx, familyID, viewerID)
	if err != nil {
		return nil, toStatusError(err)
	}

	response := &ourneztv1.ListHousingGroupsResponse{
		HousingGroups: make([]*ourneztv1.HousingGroup, 0, len(groups)),
	}
	for _, group := range groups {
		response.HousingGroups = append(response.HousingGroups, housingGroupToProto(group))
	}
	return response, nil
}

func (s HousingServer) UpdateHousingGroup(ctx context.Context, req *ourneztv1.HousingGroup) (*ourneztv1.HousingGroup, error) {
	group, err := housingGroupFromProto(req)
	if err != nil {
		return nil, toStatusError(err)
	}
	if strings.TrimSpace(string(group.ID)) == "" || strings.TrimSpace(group.Name) == "" {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	actorID, err := optionalAuthenticatedActorID(ctx, s.auth)
	if err != nil {
		return nil, toStatusError(err)
	}

	updated, err := s.housing.UpdateHousingGroup(ctx, group, actorID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return housingGroupToProto(updated), nil
}

func (s HousingServer) DeleteHousingGroup(ctx context.Context, req *ourneztv1.DeleteHousingGroupRequest) (*ourneztv1.DeleteHousingGroupResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requestActorID(ctx, s.auth, req.GetActorUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	groupID, err := requireID(req.GetHousingGroupId())
	if err != nil {
		return nil, toStatusError(err)
	}

	if err := s.housing.DeleteHousingGroup(ctx, groupID, actorID); err != nil {
		return nil, toStatusError(err)
	}
	return &ourneztv1.DeleteHousingGroupResponse{}, nil
}

func (s HousingServer) AssignHousingOptionGroup(ctx context.Context, req *ourneztv1.AssignHousingOptionGroupRequest) (*ourneztv1.HousingOption, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requestActorID(ctx, s.auth, req.GetActorUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	housingID, err := requireID(req.GetHousingId())
	if err != nil {
		return nil, toStatusError(err)
	}

	updated, err := s.housing.AssignHousingOptionGroup(ctx, housingID, domain.ID(strings.TrimSpace(req.GetHousingGroupId())), actorID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return housingToProto(updated), nil
}

func (s HousingServer) UpdateHousingOptionVisibility(ctx context.Context, req *ourneztv1.UpdateHousingOptionVisibilityRequest) (*ourneztv1.HousingOption, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requestActorID(ctx, s.auth, req.GetActorUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	housingID, err := requireID(req.GetHousingId())
	if err != nil {
		return nil, toStatusError(err)
	}

	updated, err := s.housing.UpdateHousingOptionVisibility(ctx, housingID, req.GetVisibleOnDashboard(), actorID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return housingToProto(updated), nil
}

func (s HousingServer) BulkUpdateHousingGroupVisibility(ctx context.Context, req *ourneztv1.BulkUpdateHousingGroupVisibilityRequest) (*ourneztv1.BulkUpdateHousingGroupVisibilityResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requestActorID(ctx, s.auth, req.GetActorUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	groupID, err := requireID(req.GetHousingGroupId())
	if err != nil {
		return nil, toStatusError(err)
	}

	options, err := s.housing.BulkUpdateHousingGroupVisibility(ctx, groupID, req.GetVisibleOnDashboard(), actorID)
	if err != nil {
		return nil, toStatusError(err)
	}

	response := &ourneztv1.BulkUpdateHousingGroupVisibilityResponse{
		HousingOptions: make([]*ourneztv1.HousingOption, 0, len(options)),
	}
	for _, option := range options {
		response.HousingOptions = append(response.HousingOptions, housingToProto(option))
	}
	return response, nil
}

func (s HousingServer) CalculateHousingAffordability(_ context.Context, req *ourneztv1.CalculateHousingAffordabilityRequest) (*ourneztv1.HousingAffordability, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	option, err := housingFromProto(req.GetHousingOption())
	if err != nil {
		return nil, toStatusError(err)
	}
	if err := validateHousingOption(option); err != nil {
		return nil, toStatusError(err)
	}
	assets := calculation.HouseholdAssets{
		CashSavingsCents:     req.GetCashSavingsCents(),
		CPFOACents:           req.GetCpfOaCents(),
		TakeHomeCents:        req.GetTakeHomeCents(),
		MonthlyExpensesCents: req.GetMonthlyExpensesCents(),
	}

	result := calculation.CalculateHousingAffordability(option, assets)
	return housingAffordabilityToProto(result), nil
}

func validateHousingOption(option domain.HousingOption) error {
	if err := validateHousingCondoCombination(option); err != nil {
		return err
	}
	return validateHousingLoanTenure(option)
}

func validateHousingCondoCombination(option domain.HousingOption) error {
	if !calculation.IsNonHDBHousingType(option.Type) {
		return nil
	}
	if option.LoanType == domain.LoanTypeHDB {
		return apperror.ErrInvalidArgument
	}
	if option.GrantAmountCents > 0 {
		return apperror.ErrInvalidArgument
	}
	return nil
}

func validateHousingLoanTenure(option domain.HousingOption) error {
	if option.LoanType == domain.LoanTypeCash || option.LoanTenureMonths <= 0 {
		return nil
	}
	if option.LoanTenureMonths > calculation.MaxLoanTenureMonthsForHousingType(option.Type) {
		return apperror.ErrInvalidArgument
	}
	return nil
}

func (s HousingServer) EstimateHousingGrant(ctx context.Context, req *ourneztv1.EstimateHousingGrantRequest) (*ourneztv1.EstimateHousingGrantResponse, error) {
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
	estimate := calculation.EstimateHousingGrantAmount(domain.HousingType(req.GetHousingType()), people)
	return &ourneztv1.EstimateHousingGrantResponse{
		GrantAmountCents:                 estimate.GrantAmountCents,
		HouseholdGrossMonthlyIncomeCents: estimate.HouseholdGrossMonthlyIncomeCents,
		Eligible:                         estimate.Eligible,
	}, nil
}

func (s HousingServer) applyGrantEstimate(ctx context.Context, option domain.HousingOption, viewerID domain.ID) (domain.HousingOption, error) {
	if s.people == nil || option.FamilyID == "" {
		return option, nil
	}
	if calculation.IsNonHDBHousingType(option.Type) {
		option.GrantAmountCents = 0
		return option, nil
	}
	if calculation.IsDeferredHousingOption(option) {
		option.GrantAmountCents = 0
		return option, nil
	}
	people, err := s.people.ListPersonProfilesByFamily(ctx, option.FamilyID, viewerID)
	if err != nil {
		return option, err
	}
	estimate := calculation.EstimateHousingGrantAmount(option.Type, people)
	option.GrantAmountCents = estimate.GrantAmountCents
	return option, nil
}
