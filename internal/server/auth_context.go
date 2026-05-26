package server

import (
	"context"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"google.golang.org/grpc/metadata"
)

const (
	authorizationHeader = "authorization"
	sessionTokenHeader   = "x-session-token"
)

type Authenticator interface {
	Authenticate(ctx context.Context) (domain.User, error)
}

func (s AuthServer) Authenticate(ctx context.Context) (domain.User, error) {
	token := sessionTokenFromContext(ctx)
	if token == "" {
		return domain.User{}, apperror.ErrUnauthenticated
	}

	user, err := s.sessions.GetUserBySessionTokenHash(ctx, security.HashToken(token), s.now())
	if err != nil {
		return domain.User{}, err
	}
	if user.DisabledAt != nil {
		return domain.User{}, apperror.ErrDisabledUser
	}

	return user, nil
}

func sessionTokenFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	if values := md.Get(sessionTokenHeader); len(values) > 0 {
		return strings.TrimSpace(values[0])
	}

	for _, value := range md.Get(authorizationHeader) {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(strings.ToLower(value), "bearer ") {
			return strings.TrimSpace(value[len("bearer "):])
		}
	}

	return ""
}

func authenticateUserID(ctx context.Context, auth Authenticator) (domain.ID, error) {
	if auth == nil {
		return "", apperror.ErrUnauthenticated
	}

	user, err := auth.Authenticate(ctx)
	if err != nil {
		return "", err
	}

	return user.ID, nil
}

func authenticatedAdmin(ctx context.Context, auth Authenticator) (domain.User, error) {
	if auth == nil {
		return domain.User{}, apperror.ErrUnauthenticated
	}

	user, err := auth.Authenticate(ctx)
	if err != nil {
		return domain.User{}, err
	}

	if user.Role != domain.UserRoleAdmin {
		return domain.User{}, apperror.ErrForbidden
	}

	return user, nil
}

func requestActorID(ctx context.Context, auth Authenticator, fallback string) (domain.ID, error) {
	if auth != nil {
		return authenticateUserID(ctx, auth)
	}

	return requireID(fallback)
}

func optionalAuthenticatedActorID(ctx context.Context, auth Authenticator) (domain.ID, error) {
	if auth != nil {
		return authenticateUserID(ctx, auth)
	}

	return "", nil
}
