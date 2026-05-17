package server

import (
	"context"

	"github.com/OurNeZt/ournezt-core/internal/calculation"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/repository"
)

type HousingServer struct {
	ourneztv1.UnimplementedHousingServiceServer
	housing repository.Housing
}

func NewHousingServer(housing repository.Housing) HousingServer {
	return HousingServer{housing: housing}
}

func (s HousingServer) CreateHousingOption(ctx context.Context, req *ourneztv1.HousingOption) (*ourneztv1.HousingOption, error) {
	option, err := housingFromProto(req)
	if err != nil {
		return nil, toStatusError(err)
	}

	created, err := s.housing.CreateHousingOption(ctx, option)
	if err != nil {
		return nil, toStatusError(err)
	}
	return housingToProto(created), nil
}

func (s HousingServer) GetHousingOption(ctx context.Context, req *ourneztv1.GetHousingOptionRequest) (*ourneztv1.HousingOption, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	viewerID, err := requireID(req.GetViewerUserId())
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
	viewerID, err := requireID(req.GetViewerUserId())
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

	updated, err := s.housing.UpdateHousingOption(ctx, option)
	if err != nil {
		return nil, toStatusError(err)
	}
	return housingToProto(updated), nil
}

func (s HousingServer) DeleteHousingOption(ctx context.Context, req *ourneztv1.DeleteHousingOptionRequest) (*ourneztv1.DeleteHousingOptionResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requireID(req.GetActorUserId())
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

func (s HousingServer) CalculateHousingAffordability(_ context.Context, req *ourneztv1.CalculateHousingAffordabilityRequest) (*ourneztv1.HousingAffordability, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	option, err := housingFromProto(req.GetHousingOption())
	if err != nil {
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
