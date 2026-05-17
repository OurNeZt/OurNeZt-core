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
	created  domain.User
	users    map[string]domain.User
	disabled []domain.ID
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

func (r *fakeServerUserRepository) DisableUser(_ context.Context, userID domain.ID) error {
	r.disabled = append(r.disabled, userID)
	return nil
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
	sessions := &fakeAuthSessionStore{}
	server := NewAuthServer(authService, sessions, 32, time.Hour, time.Now)

	_, err := server.DisableUser(context.Background(), &ourneztv1.DisableUserRequest{UserId: "user_9"})
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
