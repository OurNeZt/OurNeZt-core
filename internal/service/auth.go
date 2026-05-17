package service

import (
	"context"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	DisableUser(ctx context.Context, userID domain.ID) error
}

type AuthService struct {
	users        UserRepository
	argon2Params security.Argon2Params
}

func NewAuthService(users UserRepository, argon2Params security.Argon2Params) AuthService {
	return AuthService{users: users, argon2Params: argon2Params}
}

func (s AuthService) CreateUser(ctx context.Context, email, displayName, password string, role domain.UserRole) (domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || strings.TrimSpace(password) == "" {
		return domain.User{}, apperror.ErrInvalidArgument
	}
	if role == "" {
		role = domain.UserRoleUser
	}

	hash, err := security.HashPassword(password, s.argon2Params)
	if err != nil {
		return domain.User{}, err
	}

	return s.users.CreateUser(ctx, domain.User{
		Email:        email,
		DisplayName:  strings.TrimSpace(displayName),
		Role:         role,
		PasswordHash: hash,
	})
}

func (s AuthService) Login(ctx context.Context, email, password string) (domain.User, error) {
	user, err := s.users.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return domain.User{}, err
	}
	if user.DisabledAt != nil {
		return domain.User{}, apperror.ErrDisabledUser
	}

	ok, err := security.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return domain.User{}, err
	}
	if !ok {
		return domain.User{}, apperror.ErrUnauthenticated
	}

	return user, nil
}

func (s AuthService) DisableUser(ctx context.Context, userID domain.ID) error {
	if strings.TrimSpace(string(userID)) == "" {
		return apperror.ErrInvalidArgument
	}
	return s.users.DisableUser(ctx, userID)
}
