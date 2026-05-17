package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
)

type fakeUserRepository struct {
	created  domain.User
	users    map[string]domain.User
	disabled []domain.ID
}

func (r *fakeUserRepository) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
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

func (r *fakeUserRepository) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	user, ok := r.users[email]
	if !ok {
		return domain.User{}, apperror.ErrNotFound
	}
	return user, nil
}

func (r *fakeUserRepository) DisableUser(_ context.Context, userID domain.ID) error {
	r.disabled = append(r.disabled, userID)
	return nil
}

func TestCreateUserNormalizesEmailAndHashesPassword(t *testing.T) {
	repo := &fakeUserRepository{}
	service := NewAuthService(repo, fastServiceArgon2Params())

	got, err := service.CreateUser(context.Background(), " PERSON@Example.COM ", " Alex ", "temporary-pass", "")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if got.Email != "person@example.com" {
		t.Fatalf("Email = %q, want person@example.com", got.Email)
	}
	if got.DisplayName != "Alex" {
		t.Fatalf("DisplayName = %q, want Alex", got.DisplayName)
	}
	if got.Role != domain.UserRoleUser {
		t.Fatalf("Role = %q, want user", got.Role)
	}
	if got.PasswordHash == "temporary-pass" {
		t.Fatal("PasswordHash contains the raw password")
	}

	ok, err := security.VerifyPassword("temporary-pass", repo.created.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if !ok {
		t.Fatal("created password hash did not verify")
	}
}

func TestCreateUserRejectsMissingEmailOrPassword(t *testing.T) {
	service := NewAuthService(&fakeUserRepository{}, fastServiceArgon2Params())

	_, err := service.CreateUser(context.Background(), "", "Alex", "temporary-pass", domain.UserRoleUser)
	if !errors.Is(err, apperror.ErrInvalidArgument) {
		t.Fatalf("missing email error = %v, want ErrInvalidArgument", err)
	}

	_, err = service.CreateUser(context.Background(), "person@example.com", "Alex", " ", domain.UserRoleUser)
	if !errors.Is(err, apperror.ErrInvalidArgument) {
		t.Fatalf("missing password error = %v, want ErrInvalidArgument", err)
	}
}

func TestLoginAcceptsCorrectPassword(t *testing.T) {
	hash, err := security.HashPassword("temporary-pass", fastServiceArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	repo := &fakeUserRepository{
		users: map[string]domain.User{
			"person@example.com": {
				ID:           "user_1",
				Email:        "person@example.com",
				PasswordHash: hash,
			},
		},
	}
	service := NewAuthService(repo, fastServiceArgon2Params())

	got, err := service.Login(context.Background(), " PERSON@example.com ", "temporary-pass")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if got.ID != "user_1" {
		t.Fatalf("ID = %q, want user_1", got.ID)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	hash, err := security.HashPassword("temporary-pass", fastServiceArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	repo := &fakeUserRepository{
		users: map[string]domain.User{
			"person@example.com": {
				ID:           "user_1",
				Email:        "person@example.com",
				PasswordHash: hash,
			},
		},
	}
	service := NewAuthService(repo, fastServiceArgon2Params())

	_, err = service.Login(context.Background(), "person@example.com", "wrong-pass")
	if !errors.Is(err, apperror.ErrUnauthenticated) {
		t.Fatalf("Login error = %v, want ErrUnauthenticated", err)
	}
}

func TestLoginRejectsDisabledUser(t *testing.T) {
	disabledAt := time.Now()
	repo := &fakeUserRepository{
		users: map[string]domain.User{
			"person@example.com": {
				ID:           "user_1",
				Email:        "person@example.com",
				PasswordHash: "$argon2id$v=19$m=1024,t=1,p=1$YWJj$ZGVm",
				DisabledAt:   &disabledAt,
			},
		},
	}
	service := NewAuthService(repo, fastServiceArgon2Params())

	_, err := service.Login(context.Background(), "person@example.com", "temporary-pass")
	if !errors.Is(err, apperror.ErrDisabledUser) {
		t.Fatalf("Login error = %v, want ErrDisabledUser", err)
	}
}

func TestDisableUserRejectsMissingID(t *testing.T) {
	service := NewAuthService(&fakeUserRepository{}, fastServiceArgon2Params())

	err := service.DisableUser(context.Background(), "")
	if !errors.Is(err, apperror.ErrInvalidArgument) {
		t.Fatalf("DisableUser error = %v, want ErrInvalidArgument", err)
	}
}

func TestDisableUserDelegatesToRepository(t *testing.T) {
	repo := &fakeUserRepository{}
	service := NewAuthService(repo, fastServiceArgon2Params())

	if err := service.DisableUser(context.Background(), "user_9"); err != nil {
		t.Fatalf("DisableUser returned error: %v", err)
	}
	if len(repo.disabled) != 1 || repo.disabled[0] != "user_9" {
		t.Fatalf("repo disabled users = %#v, want [user_9]", repo.disabled)
	}
}

func fastServiceArgon2Params() security.Argon2Params {
	return security.Argon2Params{
		MemoryKB:    1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
}
