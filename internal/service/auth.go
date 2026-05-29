package service

import (
	"context"
	"strings"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	ListUsers(ctx context.Context) ([]domain.User, error)
	HasActiveAdmin(ctx context.Context) (bool, error)
	GetUserByID(ctx context.Context, userID domain.ID) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	UpdateUserAccount(ctx context.Context, userID domain.ID, email string, displayName string) (domain.User, error)
	PromoteToBootstrapAdmin(ctx context.Context, userID domain.ID, displayName string, passwordHash string) (domain.User, error)
	UpdateUserPassword(ctx context.Context, userID domain.ID, passwordHash string, mustChangePassword bool) error
	DisableUser(ctx context.Context, userID domain.ID) error
	CreateAdminAccessKey(ctx context.Context, userID domain.ID, tokenHash string, expiresAt time.Time) error
	ConsumeAdminAccessKey(ctx context.Context, userID domain.ID, tokenHash string, now time.Time) (bool, error)
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
	if role != domain.UserRoleUser && role != domain.UserRoleAdmin {
		return domain.User{}, apperror.ErrInvalidArgument
	}

	hash, err := security.HashPassword(password, s.argon2Params)
	if err != nil {
		return domain.User{}, err
	}

	return s.users.CreateUser(ctx, domain.User{
		Email:              email,
		DisplayName:        strings.TrimSpace(displayName),
		Role:               role,
		PasswordHash:       hash,
		MustChangePassword: role == domain.UserRoleAdmin,
	})
}

func (s AuthService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.users.ListUsers(ctx)
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

func (s AuthService) ChangePassword(ctx context.Context, userID domain.ID, currentPassword, newPassword string) error {
	if strings.TrimSpace(string(userID)) == "" || strings.TrimSpace(currentPassword) == "" || strings.TrimSpace(newPassword) == "" {
		return apperror.ErrInvalidArgument
	}

	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.DisabledAt != nil {
		return apperror.ErrDisabledUser
	}

	ok, err := security.VerifyPassword(currentPassword, user.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.ErrUnauthenticated
	}

	newHash, err := security.HashPassword(newPassword, s.argon2Params)
	if err != nil {
		return err
	}

	return s.users.UpdateUserPassword(ctx, userID, newHash, false)
}

func (s AuthService) UpdateMyAccount(ctx context.Context, userID domain.ID, email, displayName string) (domain.User, error) {
	if strings.TrimSpace(string(userID)) == "" {
		return domain.User{}, apperror.ErrInvalidArgument
	}
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedName := strings.TrimSpace(displayName)
	if normalizedEmail == "" || normalizedName == "" {
		return domain.User{}, apperror.ErrInvalidArgument
	}
	return s.users.UpdateUserAccount(ctx, userID, normalizedEmail, normalizedName)
}

func (s AuthService) GenerateAdminAccessKey(ctx context.Context, adminUserID domain.ID, keyBytes int, ttl time.Duration, now time.Time) (string, time.Time, error) {
	if strings.TrimSpace(string(adminUserID)) == "" {
		return "", time.Time{}, apperror.ErrInvalidArgument
	}
	if keyBytes <= 0 {
		keyBytes = 32
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	user, err := s.users.GetUserByID(ctx, adminUserID)
	if err != nil {
		return "", time.Time{}, err
	}
	if user.Role != domain.UserRoleAdmin {
		return "", time.Time{}, apperror.ErrForbidden
	}
	if user.MustChangePassword {
		return "", time.Time{}, apperror.ErrPasswordChangeRequired
	}

	rawKey, err := security.NewSessionToken(keyBytes)
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := now.Add(ttl)
	if err := s.users.CreateAdminAccessKey(ctx, adminUserID, security.HashToken(rawKey), expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return rawKey, expiresAt, nil
}

func (s AuthService) ConsumeAdminAccessKey(ctx context.Context, adminUserID domain.ID, rawKey string, now time.Time) error {
	if strings.TrimSpace(string(adminUserID)) == "" || strings.TrimSpace(rawKey) == "" {
		return apperror.ErrInvalidArgument
	}

	user, err := s.users.GetUserByID(ctx, adminUserID)
	if err != nil {
		return err
	}
	if user.Role != domain.UserRoleAdmin {
		return apperror.ErrForbidden
	}

	ok, err := s.users.ConsumeAdminAccessKey(ctx, adminUserID, security.HashToken(rawKey), now)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.ErrForbidden
	}
	return nil
}

func (s AuthService) DisableUser(ctx context.Context, userID domain.ID) error {
	if strings.TrimSpace(string(userID)) == "" {
		return apperror.ErrInvalidArgument
	}
	return s.users.DisableUser(ctx, userID)
}

func (s AuthService) EnsureBootstrapAdmin(ctx context.Context, email, displayName, temporaryPassword string) (domain.User, bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	temporaryPassword = strings.TrimSpace(temporaryPassword)
	if email == "" || temporaryPassword == "" {
		return domain.User{}, false, apperror.ErrInvalidArgument
	}
	if displayName == "" {
		displayName = "Bootstrap Admin"
	}

	hasAdmin, err := s.users.HasActiveAdmin(ctx)
	if err != nil {
		return domain.User{}, false, err
	}
	if hasAdmin {
		return domain.User{}, false, nil
	}

	hash, err := security.HashPassword(temporaryPassword, s.argon2Params)
	if err != nil {
		return domain.User{}, false, err
	}

	existing, err := s.users.GetUserByEmail(ctx, email)
	if err == nil {
		updated, promoteErr := s.users.PromoteToBootstrapAdmin(ctx, existing.ID, displayName, hash)
		if promoteErr != nil {
			return domain.User{}, false, promoteErr
		}
		return updated, true, nil
	}
	if err != nil && err != apperror.ErrNotFound {
		return domain.User{}, false, err
	}

	created, createErr := s.users.CreateUser(ctx, domain.User{
		Email:              email,
		DisplayName:        displayName,
		Role:               domain.UserRoleAdmin,
		PasswordHash:       hash,
		MustChangePassword: true,
	})
	if createErr != nil {
		return domain.User{}, false, createErr
	}
	return created, true, nil
}
