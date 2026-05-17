package server

import (
	"context"
	"crypto/rand"
	"strings"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/repository"
)

type FamilyServer struct {
	ourneztv1.UnimplementedFamilyServiceServer

	families repository.Families
	codeTTL  time.Duration
	now      func() time.Time
}

func NewFamilyServer(families repository.Families, codeTTL time.Duration, now func() time.Time) FamilyServer {
	if codeTTL <= 0 {
		codeTTL = 7 * 24 * time.Hour
	}
	if now == nil {
		now = time.Now
	}
	return FamilyServer{
		families: families,
		codeTTL:  codeTTL,
		now:      now,
	}
}

func (s FamilyServer) CreateFamily(ctx context.Context, req *ourneztv1.CreateFamilyRequest) (*ourneztv1.Family, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	ownerID, err := requireID(req.GetOwnerUserId())
	if err != nil {
		return nil, toStatusError(err)
	}

	family, err := s.families.CreateFamily(ctx, domainFamilyFromCreateRequest(req), ownerID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return familyToProto(family), nil
}

func (s FamilyServer) GetFamily(ctx context.Context, req *ourneztv1.GetFamilyRequest) (*ourneztv1.Family, error) {
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

	family, err := s.families.GetFamily(ctx, familyID, viewerID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return familyToProto(family), nil
}

func (s FamilyServer) ListUserFamilies(ctx context.Context, req *ourneztv1.ListUserFamiliesRequest) (*ourneztv1.ListUserFamiliesResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	userID, err := requireID(req.GetUserId())
	if err != nil {
		return nil, toStatusError(err)
	}

	families, err := s.families.ListUserFamilies(ctx, userID)
	if err != nil {
		return nil, toStatusError(err)
	}

	response := &ourneztv1.ListUserFamiliesResponse{
		Families: make([]*ourneztv1.Family, 0, len(families)),
	}
	for _, family := range families {
		response.Families = append(response.Families, familyToProto(family))
	}
	return response, nil
}

func (s FamilyServer) JoinFamilyByCode(ctx context.Context, req *ourneztv1.JoinFamilyByCodeRequest) (*ourneztv1.Family, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	userID, err := requireID(req.GetUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	code := strings.TrimSpace(req.GetCode())
	if code == "" {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	family, err := s.families.JoinFamilyByCode(ctx, userID, security.HashToken(code))
	if err != nil {
		return nil, toStatusError(err)
	}
	return familyToProto(family), nil
}

func (s FamilyServer) GenerateFamilyJoinCode(ctx context.Context, req *ourneztv1.GenerateFamilyJoinCodeRequest) (*ourneztv1.GenerateFamilyJoinCodeResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requireID(req.GetActorUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	familyID, err := requireID(req.GetFamilyId())
	if err != nil {
		return nil, toStatusError(err)
	}

	code, err := newFamilyJoinCode(10)
	if err != nil {
		return nil, toStatusError(err)
	}
	expiresAt := s.now().Add(s.codeTTL)
	if err := s.families.GenerateFamilyJoinCode(ctx, actorID, familyID, security.HashToken(code), &expiresAt); err != nil {
		return nil, toStatusError(err)
	}

	return &ourneztv1.GenerateFamilyJoinCodeResponse{Code: code}, nil
}

func (s FamilyServer) ListFamilyMembers(ctx context.Context, req *ourneztv1.ListFamilyMembersRequest) (*ourneztv1.ListFamilyMembersResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	actorID, err := requireID(req.GetActorUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	familyID, err := requireID(req.GetFamilyId())
	if err != nil {
		return nil, toStatusError(err)
	}

	members, err := s.families.ListFamilyMembers(ctx, actorID, familyID)
	if err != nil {
		return nil, toStatusError(err)
	}

	response := &ourneztv1.ListFamilyMembersResponse{
		Members: make([]*ourneztv1.FamilyMember, 0, len(members)),
	}
	for _, member := range members {
		response.Members = append(response.Members, &ourneztv1.FamilyMember{
			FamilyId:    string(member.FamilyID),
			UserId:      string(member.UserID),
			DisplayName: member.DisplayName,
			Role:        string(member.Role),
		})
	}
	return response, nil
}

func domainFamilyFromCreateRequest(req *ourneztv1.CreateFamilyRequest) domain.Family {
	return domain.Family{
		Name: strings.TrimSpace(req.GetName()),
		Type: domain.FamilyType(strings.TrimSpace(req.GetFamilyType())),
	}
}

func newFamilyJoinCode(length int) (string, error) {
	if length <= 0 {
		length = 10
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	out := make([]byte, length)
	for i := range buffer {
		out[i] = alphabet[int(buffer[i])%len(alphabet)]
	}
	return string(out), nil
}
