package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeAuthSessionStore struct {
	created struct {
		userID    domain.ID
		tokenHash string
		expiresAt time.Time
	}
	usersByToken map[string]domain.User
	revokedUsers []domain.ID
}

type fakeServerUserRepository struct {
	created         domain.User
	users           map[string]domain.User
	disabled        []domain.ID
	adminAccessKeys map[string]time.Time
}

func (s *fakeAuthSessionStore) CreateSession(_ context.Context, userID domain.ID, tokenHash string, expiresAt time.Time) error {
	s.created.userID = userID
	s.created.tokenHash = tokenHash
	s.created.expiresAt = expiresAt
	if s.usersByToken == nil {
		s.usersByToken = map[string]domain.User{}
	}
	return nil
}

func (s *fakeAuthSessionStore) GetUserBySessionTokenHash(_ context.Context, tokenHash string, _ time.Time) (domain.User, error) {
	user, ok := s.usersByToken[tokenHash]
	if !ok {
		return domain.User{}, apperror.ErrUnauthenticated
	}
	return user, nil
}

func (s *fakeAuthSessionStore) RevokeSessionsByUserID(_ context.Context, userID domain.ID) error {
	s.revokedUsers = append(s.revokedUsers, userID)
	return nil
}

func (r *fakeServerUserRepository) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	if r.users == nil {
		r.users = map[string]domain.User{}
	}
	if user.ID == "" {
		user.ID = "user_1"
	}
	r.created = user
	r.users[user.Email] = user
	return user, nil
}

func (r *fakeServerUserRepository) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	user, ok := r.users[email]
	if !ok {
		return domain.User{}, apperror.ErrNotFound
	}
	return user, nil
}

func (r *fakeServerUserRepository) HasActiveAdmin(_ context.Context) (bool, error) {
	for _, user := range r.users {
		if user.Role == domain.UserRoleAdmin && user.DisabledAt == nil {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeServerUserRepository) GetUserByID(_ context.Context, userID domain.ID) (domain.User, error) {
	for _, user := range r.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return domain.User{}, apperror.ErrNotFound
}

func (r *fakeServerUserRepository) PromoteToBootstrapAdmin(_ context.Context, userID domain.ID, displayName string, passwordHash string) (domain.User, error) {
	for email, user := range r.users {
		if user.ID == userID {
			user.Role = domain.UserRoleAdmin
			user.DisplayName = displayName
			user.PasswordHash = passwordHash
			user.MustChangePassword = true
			user.DisabledAt = nil
			r.users[email] = user
			return user, nil
		}
	}
	return domain.User{}, apperror.ErrNotFound
}

func (r *fakeServerUserRepository) DisableUser(_ context.Context, userID domain.ID) error {
	r.disabled = append(r.disabled, userID)
	return nil
}

func (r *fakeServerUserRepository) UpdateUserPassword(_ context.Context, userID domain.ID, passwordHash string, mustChangePassword bool) error {
	for email, user := range r.users {
		if user.ID == userID {
			user.PasswordHash = passwordHash
			user.MustChangePassword = mustChangePassword
			r.users[email] = user
			return nil
		}
	}
	return apperror.ErrNotFound
}

func (r *fakeServerUserRepository) CreateAdminAccessKey(_ context.Context, userID domain.ID, tokenHash string, expiresAt time.Time) error {
	if r.adminAccessKeys == nil {
		r.adminAccessKeys = map[string]time.Time{}
	}
	r.adminAccessKeys[string(userID)+"|"+tokenHash] = expiresAt
	return nil
}

func (r *fakeServerUserRepository) ConsumeAdminAccessKey(_ context.Context, userID domain.ID, tokenHash string, now time.Time) (bool, error) {
	expiresAt, ok := r.adminAccessKeys[string(userID)+"|"+tokenHash]
	if !ok {
		return false, nil
	}
	if now.After(expiresAt) {
		return false, nil
	}
	delete(r.adminAccessKeys, string(userID)+"|"+tokenHash)
	return true, nil
}

func TestAuthServerLoginCreatesSessionAndReturnsToken(t *testing.T) {
	hash, err := security.HashPassword("temporary-pass", fastServerArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	repo := &fakeServerUserRepository{
		users: map[string]domain.User{
			"person@example.com": {
				ID:           "user_1",
				Email:        "person@example.com",
				PasswordHash: hash,
				Role:         domain.UserRoleUser,
			},
		},
	}
	authService := service.NewAuthService(repo, fastServerArgon2Params())
	sessions := &fakeAuthSessionStore{}
	now := time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC)
	server := NewAuthServer(authService, sessions, 32, 2*time.Hour, func() time.Time { return now })

	response, err := server.Login(context.Background(), &ourneztv1.LoginRequest{
		Email:    "person@example.com",
		Password: "temporary-pass",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if response.GetSessionToken() == "" {
		t.Fatal("session token is empty")
	}
	if response.GetUser().GetId() != "user_1" {
		t.Fatalf("user id = %q, want user_1", response.GetUser().GetId())
	}
	if sessions.created.userID != "user_1" {
		t.Fatalf("created session userID = %q, want user_1", sessions.created.userID)
	}
	if sessions.created.tokenHash == "" {
		t.Fatal("created token hash is empty")
	}
	wantExpiry := now.Add(2 * time.Hour)
	if !sessions.created.expiresAt.Equal(wantExpiry) {
		t.Fatalf("expiresAt = %v, want %v", sessions.created.expiresAt, wantExpiry)
	}
}

func TestAuthServerValidateSessionReturnsUser(t *testing.T) {
	authService := service.NewAuthService(&fakeServerUserRepository{}, fastServerArgon2Params())
	sessions := &fakeAuthSessionStore{
		usersByToken: map[string]domain.User{
			security.HashToken("session-token"): {
				ID:    "user_4",
				Email: "person@example.com",
				Role:  domain.UserRoleAdmin,
			},
		},
	}
	server := NewAuthServer(authService, sessions, 32, time.Hour, time.Now)

	response, err := server.ValidateSession(context.Background(), &ourneztv1.ValidateSessionRequest{
		SessionToken: "session-token",
	})
	if err != nil {
		t.Fatalf("ValidateSession returned error: %v", err)
	}
	if response.GetUser().GetId() != "user_4" {
		t.Fatalf("user id = %q, want user_4", response.GetUser().GetId())
	}
}

func TestAuthServerValidateSessionRejectsEmptyToken(t *testing.T) {
	authService := service.NewAuthService(&fakeServerUserRepository{}, fastServerArgon2Params())
	server := NewAuthServer(authService, &fakeAuthSessionStore{}, 32, time.Hour, time.Now)

	_, err := server.ValidateSession(context.Background(), &ourneztv1.ValidateSessionRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestAuthServerDisableUserDisablesAndRevokes(t *testing.T) {
	repo := &fakeServerUserRepository{}
	authService := service.NewAuthService(repo, fastServerArgon2Params())
	sessions := &fakeAuthSessionStore{
		usersByToken: map[string]domain.User{
			security.HashToken("admin-token"): {
				ID:    "admin_1",
				Email: "admin@example.com",
				Role:  domain.UserRoleAdmin,
			},
		},
	}
	server := NewAuthServer(authService, sessions, 32, time.Hour, time.Now)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-session-token", "admin-token"))
	_, err := server.DisableUser(ctx, &ourneztv1.DisableUserRequest{UserId: "user_9"})
	if err != nil {
		t.Fatalf("DisableUser returned error: %v", err)
	}
	if len(repo.disabled) != 1 || repo.disabled[0] != "user_9" {
		t.Fatalf("disabled users = %#v, want [user_9]", repo.disabled)
	}
	if len(sessions.revokedUsers) != 1 || sessions.revokedUsers[0] != "user_9" {
		t.Fatalf("revoked users = %#v, want [user_9]", sessions.revokedUsers)
	}
}

func TestAuthServerDisableUserRejectsNonAdmin(t *testing.T) {
	repo := &fakeServerUserRepository{}
	authService := service.NewAuthService(repo, fastServerArgon2Params())
	sessions := &fakeAuthSessionStore{
		usersByToken: map[string]domain.User{
			security.HashToken("user-token"): {
				ID:    "user_1",
				Email: "user@example.com",
				Role:  domain.UserRoleUser,
			},
		},
	}
	server := NewAuthServer(authService, sessions, 32, time.Hour, time.Now)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-session-token", "user-token"))
	_, err := server.DisableUser(ctx, &ourneztv1.DisableUserRequest{UserId: "user_9"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status code = %v, want PermissionDenied", status.Code(err))
	}
}

func TestAuthServerChangePassword(t *testing.T) {
	hash, err := security.HashPassword("temporary-pass", fastServerArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	repo := &fakeServerUserRepository{
		users: map[string]domain.User{
			"admin@example.com": {
				ID:                 "admin_1",
				Email:              "admin@example.com",
				Role:               domain.UserRoleAdmin,
				PasswordHash:       hash,
				MustChangePassword: true,
			},
		},
	}
	authService := service.NewAuthService(repo, fastServerArgon2Params())
	sessions := &fakeAuthSessionStore{
		usersByToken: map[string]domain.User{
			security.HashToken("admin-token"): {
				ID:                 "admin_1",
				Email:              "admin@example.com",
				Role:               domain.UserRoleAdmin,
				PasswordHash:       hash,
				MustChangePassword: true,
			},
		},
	}
	server := NewAuthServer(authService, sessions, 32, time.Hour, time.Now)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-session-token", "admin-token"))
	_, err = server.ChangePassword(ctx, &ourneztv1.ChangePasswordRequest{
		CurrentPassword: "temporary-pass",
		NewPassword:     "safer-pass",
	})
	if err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}
	updated := repo.users["admin@example.com"]
	if updated.MustChangePassword {
		t.Fatal("MustChangePassword = true after change, want false")
	}
}

func TestAuthServerGenerateAndConsumeAdminAccessKey(t *testing.T) {
	hash, err := security.HashPassword("safer-pass", fastServerArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	repo := &fakeServerUserRepository{
		users: map[string]domain.User{
			"admin@example.com": {
				ID:           "admin_1",
				Email:        "admin@example.com",
				Role:         domain.UserRoleAdmin,
				PasswordHash: hash,
			},
		},
	}
	authService := service.NewAuthService(repo, fastServerArgon2Params())
	sessions := &fakeAuthSessionStore{
		usersByToken: map[string]domain.User{
			security.HashToken("admin-token"): {
				ID:           "admin_1",
				Email:        "admin@example.com",
				Role:         domain.UserRoleAdmin,
				PasswordHash: hash,
			},
		},
	}
	now := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	server := NewAuthServer(authService, sessions, 32, time.Hour, func() time.Time { return now })
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-session-token", "admin-token"))

	generateResp, err := server.GenerateAdminAccessKey(ctx, &ourneztv1.GenerateAdminAccessKeyRequest{})
	if err != nil {
		t.Fatalf("GenerateAdminAccessKey returned error: %v", err)
	}
	if generateResp.GetAccessKey() == "" {
		t.Fatal("AccessKey is empty")
	}

	consumeResp, err := server.ConsumeAdminAccessKey(ctx, &ourneztv1.ConsumeAdminAccessKeyRequest{
		AccessKey: generateResp.GetAccessKey(),
	})
	if err != nil {
		t.Fatalf("ConsumeAdminAccessKey returned error: %v", err)
	}
	if !consumeResp.GetAccessGranted() {
		t.Fatal("AccessGranted = false, want true")
	}
}

func fastServerArgon2Params() security.Argon2Params {
	return security.Argon2Params{
		MemoryKB:    1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
}

var _ service.UserRepository = (*fakeServerUserRepository)(nil)
var _ AuthSessionStore = (*fakeAuthSessionStore)(nil)

func TestToStatusErrorMapsConflict(t *testing.T) {
	err := toStatusError(apperror.ErrConflict)
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("status code = %v, want AlreadyExists", status.Code(err))
	}
}

func TestToStatusErrorPreservesStatusErrors(t *testing.T) {
	original := status.Error(codes.PermissionDenied, "nope")
	got := toStatusError(original)
	if !errors.Is(got, original) {
		t.Fatalf("expected same status error, got: %v", got)
	}
}
