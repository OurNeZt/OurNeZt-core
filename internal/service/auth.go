package service

import (
	"context"
	"strings"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/authlimit"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	ListUsers(ctx context.Context, pages ...domain.Page) ([]domain.User, error)
	HasActiveAdmin(ctx context.Context) (bool, error)
	GetUserByID(ctx context.Context, userID domain.ID) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	UpdateUserAccount(ctx context.Context, userID domain.ID, email string, displayName string) (domain.User, error)
	PromoteToBootstrapAdmin(ctx context.Context, userID domain.ID, displayName string, passwordHash string) (domain.User, error)
	// UpdateUserPassword compares the verified hash and atomically revokes all
	// sessions with the password update. Failed updates must leave both intact.
	UpdateUserPassword(ctx context.Context, userID domain.ID, expectedPasswordHash, passwordHash string, mustChangePassword bool) error
	DisableUser(ctx context.Context, userID domain.ID) error
	CreateAdminAccessKey(ctx context.Context, userID domain.ID, tokenHash string, expiresAt time.Time) error
	ConsumeAdminAccessKey(ctx context.Context, userID domain.ID, tokenHash string, now time.Time) (bool, error)
}

type AuthService struct {
	users             UserRepository
	argon2Params      security.Argon2Params
	limits            *authlimit.Limiter
	dummyPasswordHash string
	verifyPassword    func(string, string) (bool, error)
}

func NewAuthService(users UserRepository, argon2Params security.Argon2Params, limits ...*authlimit.Limiter) AuthService {
	var guard *authlimit.Limiter
	if len(limits) > 0 && limits[0] != nil {
		guard = limits[0]
	}
	if guard == nil {
		guard = authlimit.New(authlimit.Config{})
	}
	return AuthService{
		users: users, argon2Params: argon2Params, limits: guard,
		dummyPasswordHash: security.DummyPasswordHash(argon2Params),
		verifyPassword:    security.VerifyPassword,
	}
}

func (s AuthService) beginPasswordWork(ctx context.Context, account string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if retry := s.limits.AllowAccount(account); retry > 0 {
		return nil, &authlimit.LimitError{RetryAfter: retry}
	}
	return s.limits.Acquire(ctx)
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

	release, err := s.beginPasswordWork(ctx, "create:"+email)
	if err != nil {
		return domain.User{}, err
	}
	defer release()

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

func (s AuthService) ListUsers(ctx context.Context, pages ...domain.Page) ([]domain.User, error) {
	return s.users.ListUsers(ctx, pages...)
}

func (s AuthService) Login(ctx context.Context, email, password string) (domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	release, err := s.beginPasswordWork(ctx, "login:"+email)
	if err != nil {
		return domain.User{}, err
	}
	defer release()

	user, lookupErr := s.users.GetUserByEmail(ctx, email)
	hash := user.PasswordHash
	if lookupErr != nil {
		hash = s.dummyPasswordHash
	}

	// Unknown accounts and lookup failures do the same bounded password work as
	// known accounts. Disabled accounts are verified before rejecting them.
	ok, verifyErr := s.verifyPassword(password, hash)
	if verifyErr != nil && lookupErr == nil {
		// A malformed stored hash fails before the KDF; do dummy work as well.
		_, _ = s.verifyPassword(password, s.dummyPasswordHash)
	}
	if lookupErr != nil || verifyErr != nil || !ok || user.DisabledAt != nil {
		return domain.User{}, apperror.ErrUnauthenticated
	}

	return user, nil
}

func (s AuthService) ChangePassword(ctx context.Context, userID domain.ID, currentPassword, newPassword string) error {
	if strings.TrimSpace(string(userID)) == "" || strings.TrimSpace(currentPassword) == "" || strings.TrimSpace(newPassword) == "" {
		return apperror.ErrInvalidArgument
	}

	release, err := s.beginPasswordWork(ctx, "password:"+string(userID))
	if err != nil {
		return err
	}
	defer release()

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

	return s.users.UpdateUserPassword(ctx, userID, user.PasswordHash, newHash, false)
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

	release, err := s.limits.Acquire(ctx)
	if err != nil {
		return domain.User{}, false, err
	}
	defer release()

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
