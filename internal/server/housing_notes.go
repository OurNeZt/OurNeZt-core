package server

import (
	"context"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
)

func (s HousingServer) UpdateHousingNotes(ctx context.Context, req *pb.UpdateHousingNotesRequest) (*pb.UpdateHousingNotesResponse, error) {
	actorID, err := authenticateUserID(ctx, s.auth)
	if err != nil {
		return nil, toStatusError(err)
	}
	housingID, err := requireID(req.GetHousingId())
	if err != nil {
		return nil, toStatusError(err)
	}
	if err := domain.ValidateHousingNotes(req.GetNotes()); err != nil {
		return nil, toStatusError(err)
	}
	notes, err := s.housing.UpdateHousingNotes(ctx, housingID, req.GetNotes(), actorID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return &pb.UpdateHousingNotesResponse{Notes: notes}, nil
}
