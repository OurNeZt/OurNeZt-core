package server

import (
	"context"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/service"
)

type AuthSessionStore interface {
	CreateSession(ctx context.Context, userID domain.ID, tokenHash string, expiresAt time.Time) error
	GetUserBySessionTokenHash(ctx context.Context, tokenHash string, now time.Time) (domain.User, error)
	RevokeSessionsByUserID(ctx context.Context, userID domain.ID) error
}

type AuthServer struct {
	ourneztv1.UnimplementedAuthServiceServer

	auth       service.AuthService
	sessions   AuthSessionStore
	tokenBytes int
	sessionTTL time.Duration
	now        func() time.Time
}

func NewAuthServer(auth service.AuthService, sessions AuthSessionStore, tokenBytes int, sessionTTL time.Duration, now func() time.Time) AuthServer {
	if tokenBytes <= 0 {
		tokenBytes = 32
	}
	if sessionTTL <= 0 {
		sessionTTL = 24 * time.Hour
	}
	if now == nil {
		now = time.Now
	}
	return AuthServer{
		auth:       auth,
		sessions:   sessions,
		tokenBytes: tokenBytes,
		sessionTTL: sessionTTL,
		now:        now,
	}
}

func (s AuthServer) CreateUser(ctx context.Context, req *ourneztv1.CreateUserRequest) (*ourneztv1.User, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	role := domain.UserRole(req.GetRole())
	if role == "" {
		role = domain.UserRoleUser
	}

	if role == domain.UserRoleAdmin {
		if _, err := authenticatedAdmin(ctx, s); err != nil {
			return nil, toStatusError(err)
		}
	}

	user, err := s.auth.CreateUser(ctx, req.GetEmail(), req.GetDisplayName(), req.GetPassword(), role)
	if err != nil {
		return nil, toStatusError(err)
	}
	return userToProto(user), nil
}

func (s AuthServer) Login(ctx context.Context, req *ourneztv1.LoginRequest) (*ourneztv1.LoginResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	user, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, toStatusError(err)
	}

	token, err := security.NewSessionToken(s.tokenBytes)
	if err != nil {
		return nil, toStatusError(err)
	}
	tokenHash := security.HashToken(token)

	if err := s.sessions.CreateSession(ctx, user.ID, tokenHash, s.now().Add(s.sessionTTL)); err != nil {
		return nil, toStatusError(err)
	}

	return &ourneztv1.LoginResponse{
		User:         userToProto(user),
		SessionToken: token,
	}, nil
}

func (s AuthServer) ValidateSession(ctx context.Context, req *ourneztv1.ValidateSessionRequest) (*ourneztv1.ValidateSessionResponse, error) {
	if req == nil || req.GetSessionToken() == "" {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	user, err := s.sessions.GetUserBySessionTokenHash(ctx, security.HashToken(req.GetSessionToken()), s.now())
	if err != nil {
		return nil, toStatusError(err)
	}
	if user.DisabledAt != nil {
		return nil, toStatusError(apperror.ErrDisabledUser)
	}

	return &ourneztv1.ValidateSessionResponse{User: userToProto(user)}, nil
}

func (s AuthServer) DisableUser(ctx context.Context, req *ourneztv1.DisableUserRequest) (*ourneztv1.DisableUserResponse, error) {
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	actor, err := authenticatedAdmin(ctx, s)
	if err != nil {
		return nil, toStatusError(err)
	}

	userID, err := requireID(req.GetUserId())
	if err != nil {
		return nil, toStatusError(err)
	}

	if userID == actor.ID {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}

	if err := s.auth.DisableUser(ctx, userID); err != nil {
		return nil, toStatusError(err)
	}
	if err := s.sessions.RevokeSessionsByUserID(ctx, userID); err != nil {
		return nil, toStatusError(err)
	}

	return &ourneztv1.DisableUserResponse{}, nil
}
