package server

import (
	"context"

	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/repository"
)

type PersonServer struct {
	ourneztv1.UnimplementedPersonServiceServer
	people repository.People
}

func NewPersonServer(people repository.People) PersonServer {
	return PersonServer{people: people}
}

func (s PersonServer) CreatePersonProfile(ctx context.Context, req *ourneztv1.PersonProfile) (*ourneztv1.PersonProfile, error) {
	profile, err := personFromProto(req)
	if err != nil {
		return nil, toStatusError(err)
	}

	created, err := s.people.CreatePersonProfile(ctx, profile)
	if err != nil {
		return nil, toStatusError(err)
	}
	return personToProto(created), nil
}

func (s PersonServer) GetPersonProfile(ctx context.Context, req *ourneztv1.GetPersonProfileRequest) (*ourneztv1.PersonProfile, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	viewerID, err := requireID(req.GetViewerUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	personID, err := requireID(req.GetPersonId())
	if err != nil {
		return nil, toStatusError(err)
	}

	profile, err := s.people.GetPersonProfile(ctx, personID, viewerID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return personToProto(profile), nil
}

func (s PersonServer) ListPersonProfilesByFamily(ctx context.Context, req *ourneztv1.ListPersonProfilesByFamilyRequest) (*ourneztv1.ListPersonProfilesByFamilyResponse, error) {
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

	profiles, err := s.people.ListPersonProfilesByFamily(ctx, familyID, viewerID)
	if err != nil {
		return nil, toStatusError(err)
	}

	response := &ourneztv1.ListPersonProfilesByFamilyResponse{
		People: make([]*ourneztv1.PersonProfile, 0, len(profiles)),
	}
	for _, profile := range profiles {
		response.People = append(response.People, personToProto(profile))
	}
	return response, nil
}

func (s PersonServer) UpdatePersonProfile(ctx context.Context, req *ourneztv1.PersonProfile) (*ourneztv1.PersonProfile, error) {
	profile, err := personFromProto(req)
	if err != nil {
		return nil, toStatusError(err)
	}
	if profile.ID == "" {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	updated, err := s.people.UpdatePersonProfile(ctx, profile)
	if err != nil {
		return nil, toStatusError(err)
	}
	return personToProto(updated), nil
}

func (s PersonServer) DeletePersonProfile(ctx context.Context, req *ourneztv1.DeletePersonProfileRequest) (*ourneztv1.DeletePersonProfileResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requireID(req.GetActorUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	personID, err := requireID(req.GetPersonId())
	if err != nil {
		return nil, toStatusError(err)
	}

	if err := s.people.DeletePersonProfile(ctx, personID, actorID); err != nil {
		return nil, toStatusError(err)
	}
	return &ourneztv1.DeletePersonProfileResponse{}, nil
}

