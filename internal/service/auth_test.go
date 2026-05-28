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
	created         domain.User
	listed          []domain.User
	users           map[string]domain.User
	disabled        []domain.ID
	hasActiveAdmin  bool
	passwordUpdates []struct {
		userID     domain.ID
		hash       string
		mustChange bool
	}
	adminAccessKeys map[string]time.Time
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

func (r *fakeUserRepository) ListUsers(_ context.Context) ([]domain.User, error) {
	if r.listed != nil {
		return r.listed, nil
	}
	users := make([]domain.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	return users, nil
}

func (r *fakeUserRepository) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	user, ok := r.users[email]
	if !ok {
		return domain.User{}, apperror.ErrNotFound
	}
	return user, nil
}

func (r *fakeUserRepository) HasActiveAdmin(_ context.Context) (bool, error) {
	return r.hasActiveAdmin, nil
}

func (r *fakeUserRepository) GetUserByID(_ context.Context, userID domain.ID) (domain.User, error) {
	for _, user := range r.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return domain.User{}, apperror.ErrNotFound
}

func (r *fakeUserRepository) PromoteToBootstrapAdmin(_ context.Context, userID domain.ID, displayName string, passwordHash string) (domain.User, error) {
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

func (r *fakeUserRepository) UpdateUserPassword(_ context.Context, userID domain.ID, passwordHash string, mustChangePassword bool) error {
	r.passwordUpdates = append(r.passwordUpdates, struct {
		userID     domain.ID
		hash       string
		mustChange bool
	}{
		userID:     userID,
		hash:       passwordHash,
		mustChange: mustChangePassword,
	})

	for email, user := range r.users {
		if user.ID == userID {
			user.PasswordHash = passwordHash
			user.MustChangePassword = mustChangePassword
			r.users[email] = user
		}
	}
	return nil
}

func (r *fakeUserRepository) DisableUser(_ context.Context, userID domain.ID) error {
	r.disabled = append(r.disabled, userID)
	return nil
}

func (r *fakeUserRepository) CreateAdminAccessKey(_ context.Context, userID domain.ID, tokenHash string, expiresAt time.Time) error {
	if r.adminAccessKeys == nil {
		r.adminAccessKeys = map[string]time.Time{}
	}
	r.adminAccessKeys[string(userID)+"|"+tokenHash] = expiresAt
	return nil
}

func (r *fakeUserRepository) ConsumeAdminAccessKey(_ context.Context, userID domain.ID, tokenHash string, now time.Time) (bool, error) {
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
	if got.MustChangePassword {
		t.Fatal("MustChangePassword = true, want false for normal users")
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

func TestCreateAdminUserRequiresPasswordChange(t *testing.T) {
	repo := &fakeUserRepository{}
	service := NewAuthService(repo, fastServiceArgon2Params())

	got, err := service.CreateUser(context.Background(), "admin@example.com", "Admin", "temporary-pass", domain.UserRoleAdmin)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if !got.MustChangePassword {
		t.Fatal("MustChangePassword = false, want true for admin users")
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

func TestListUsersReturnsRepositoryResults(t *testing.T) {
	expected := []domain.User{
		{ID: "admin_1", Email: "admin@example.com", Role: domain.UserRoleAdmin},
		{ID: "user_1", Email: "user@example.com", Role: domain.UserRoleUser},
	}
	repo := &fakeUserRepository{listed: expected}
	service := NewAuthService(repo, fastServiceArgon2Params())

	got, err := service.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("len(users) = %d, want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i].ID != expected[i].ID {
			t.Fatalf("users[%d].ID = %q, want %q", i, got[i].ID, expected[i].ID)
		}
		if got[i].Email != expected[i].Email {
			t.Fatalf("users[%d].Email = %q, want %q", i, got[i].Email, expected[i].Email)
		}
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

func TestChangePasswordUpdatesHashAndClearsFlag(t *testing.T) {
	hash, err := security.HashPassword("temporary-pass", fastServiceArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	repo := &fakeUserRepository{
		users: map[string]domain.User{
			"admin@example.com": {
				ID:                 "admin_1",
				Email:              "admin@example.com",
				PasswordHash:       hash,
				MustChangePassword: true,
				Role:               domain.UserRoleAdmin,
			},
		},
	}
	service := NewAuthService(repo, fastServiceArgon2Params())

	err = service.ChangePassword(context.Background(), "admin_1", "temporary-pass", "safer-pass")
	if err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}
	if len(repo.passwordUpdates) != 1 {
		t.Fatalf("password updates = %d, want 1", len(repo.passwordUpdates))
	}
	if repo.passwordUpdates[0].mustChange {
		t.Fatal("mustChange flag stayed true after password change")
	}
	ok, verifyErr := security.VerifyPassword("safer-pass", repo.passwordUpdates[0].hash)
	if verifyErr != nil {
		t.Fatalf("VerifyPassword returned error: %v", verifyErr)
	}
	if !ok {
		t.Fatal("updated hash did not verify")
	}
}

func TestGenerateAndConsumeAdminAccessKey(t *testing.T) {
	repo := &fakeUserRepository{
		users: map[string]domain.User{
			"admin@example.com": {
				ID:    "admin_1",
				Email: "admin@example.com",
				Role:  domain.UserRoleAdmin,
			},
		},
	}
	service := NewAuthService(repo, fastServiceArgon2Params())
	now := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)

	key, expiresAt, err := service.GenerateAdminAccessKey(context.Background(), "admin_1", 32, 5*time.Minute, now)
	if err != nil {
		t.Fatalf("GenerateAdminAccessKey returned error: %v", err)
	}
	if key == "" {
		t.Fatal("generated key is empty")
	}
	if !expiresAt.Equal(now.Add(5 * time.Minute)) {
		t.Fatalf("expiresAt = %v, want %v", expiresAt, now.Add(5*time.Minute))
	}

	err = service.ConsumeAdminAccessKey(context.Background(), "admin_1", key, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("ConsumeAdminAccessKey returned error: %v", err)
	}
}

func TestEnsureBootstrapAdminCreatesWhenNoAdminExists(t *testing.T) {
	repo := &fakeUserRepository{users: map[string]domain.User{}}
	service := NewAuthService(repo, fastServiceArgon2Params())

	user, changed, err := service.EnsureBootstrapAdmin(context.Background(), "admin@ournezt.local", "Bootstrap Admin", "TempPass123!")
	if err != nil {
		t.Fatalf("EnsureBootstrapAdmin returned error: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if user.Role != domain.UserRoleAdmin {
		t.Fatalf("Role = %q, want admin", user.Role)
	}
	if !user.MustChangePassword {
		t.Fatal("MustChangePassword = false, want true")
	}
}

func TestEnsureBootstrapAdminPromotesExistingUser(t *testing.T) {
	userHash, err := security.HashPassword("old", fastServiceArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	repo := &fakeUserRepository{
		users: map[string]domain.User{
			"member@ournezt.local": {
				ID:           "user_22",
				Email:        "member@ournezt.local",
				Role:         domain.UserRoleUser,
				PasswordHash: userHash,
			},
		},
	}
	service := NewAuthService(repo, fastServiceArgon2Params())

	user, changed, err := service.EnsureBootstrapAdmin(context.Background(), "member@ournezt.local", "Primary Admin", "TempPass123!")
	if err != nil {
		t.Fatalf("EnsureBootstrapAdmin returned error: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if user.Role != domain.UserRoleAdmin {
		t.Fatalf("Role = %q, want admin", user.Role)
	}
	if user.DisplayName != "Primary Admin" {
		t.Fatalf("DisplayName = %q, want Primary Admin", user.DisplayName)
	}
}

func TestEnsureBootstrapAdminSkipsWhenAdminExists(t *testing.T) {
	repo := &fakeUserRepository{
		users:          map[string]domain.User{},
		hasActiveAdmin: true,
	}
	service := NewAuthService(repo, fastServiceArgon2Params())

	user, changed, err := service.EnsureBootstrapAdmin(context.Background(), "admin@ournezt.local", "Bootstrap Admin", "TempPass123!")
	if err != nil {
		t.Fatalf("EnsureBootstrapAdmin returned error: %v", err)
	}
	if changed {
		t.Fatal("changed = true, want false")
	}
	if user.ID != "" {
		t.Fatalf("user.ID = %q, want empty", user.ID)
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
