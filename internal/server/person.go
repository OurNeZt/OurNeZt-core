package server

import (
	"context"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/repository"
)

type PersonServer struct {
	ourneztv1.UnimplementedPersonServiceServer
	people repository.People
	auth   Authenticator
}

func NewPersonServer(people repository.People, auth ...Authenticator) PersonServer {
	var authenticator Authenticator
	if len(auth) > 0 {
		authenticator = auth[0]
	}

	return PersonServer{
		people: people,
		auth:   authenticator,
	}
}

func (s PersonServer) CreatePersonProfile(ctx context.Context, req *ourneztv1.PersonProfile) (*ourneztv1.PersonProfile, error) {
	profile, err := personFromProto(req)
	if err != nil {
		return nil, toStatusError(err)
	}
	if err := validateAndNormalizePersonProfile(&profile); err != nil {
		return nil, toStatusError(err)
	}

	actorID, err := optionalAuthenticatedActorID(ctx, s.auth)
	if err != nil {
		return nil, toStatusError(err)
	}

	created, err := s.people.CreatePersonProfile(ctx, profile, actorID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return personToProto(created), nil
}

func (s PersonServer) GetPersonProfile(ctx context.Context, req *ourneztv1.GetPersonProfileRequest) (*ourneztv1.PersonProfile, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	viewerID, err := requestActorID(ctx, s.auth, req.GetViewerUserId())
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
	viewerID, err := requestActorID(ctx, s.auth, req.GetViewerUserId())
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

func (s PersonServer) ListIncomeHistoryByFamily(ctx context.Context, req *ourneztv1.ListIncomeHistoryByFamilyRequest) (*ourneztv1.ListIncomeHistoryByFamilyResponse, error) {
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

	entries, err := s.people.ListPersonIncomeHistoryByFamily(ctx, familyID, viewerID)
	if err != nil {
		return nil, toStatusError(err)
	}

	response := &ourneztv1.ListIncomeHistoryByFamilyResponse{
		Entries: make([]*ourneztv1.IncomeHistoryEntry, 0, len(entries)),
	}
	for _, entry := range entries {
		response.Entries = append(response.Entries, personIncomeHistoryEntryToProto(entry))
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
	if err := validateAndNormalizePersonProfile(&profile); err != nil {
		return nil, toStatusError(err)
	}

	actorID, err := optionalAuthenticatedActorID(ctx, s.auth)
	if err != nil {
		return nil, toStatusError(err)
	}

	updated, err := s.people.UpdatePersonProfile(ctx, profile, actorID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return personToProto(updated), nil
}

func (s PersonServer) DeletePersonProfile(ctx context.Context, req *ourneztv1.DeletePersonProfileRequest) (*ourneztv1.DeletePersonProfileResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requestActorID(ctx, s.auth, req.GetActorUserId())
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

func validateAndNormalizePersonProfile(profile *domain.PersonProfile) error {
	if profile == nil {
		return apperror.ErrInvalidArgument
	}
	if strings.TrimSpace(profile.Name) == "" {
		return apperror.ErrInvalidArgument
	}

	switch strings.TrimSpace(strings.ToLower(profile.RelationshipLabel)) {
	case "spouse", "fiance", "fiancee", "occupant", "self", "me":
	default:
		return apperror.ErrInvalidArgument
	}

	switch profile.EmploymentStatus {
	case domain.EmploymentFullTime:
		profile.ExpectedFutureIncomeCents = 0
		profile.ExpectedIncomeStartDate = nil
		profile.GraduationDate = nil
		profile.ORDDate = nil
	case domain.EmploymentStudent:
		profile.ORDDate = nil
		if profile.GraduationDate == nil {
			return apperror.ErrInvalidArgument
		}
	case domain.EmploymentFullTimeNSF:
		profile.GraduationDate = nil
		if profile.ORDDate == nil {
			return apperror.ErrInvalidArgument
		}
	default:
		return apperror.ErrInvalidArgument
	}

	if profile.GrossMonthlyIncomeCents < 0 ||
		profile.ExpectedFutureIncomeCents < 0 ||
		profile.CashSavingsCents < 0 ||
		profile.CPFOACents < 0 ||
		profile.CPFSACents < 0 ||
		profile.CPFMACents < 0 ||
		profile.MonthlyExpensesCents < 0 {
		return apperror.ErrInvalidArgument
	}

	return nil
}
