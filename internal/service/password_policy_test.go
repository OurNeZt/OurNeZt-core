package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/authlimit"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
)

// Any account lookup or mutation panics through the nil embedded interface.
// Bootstrap's initial existence check is the only allowed repository operation.
type passwordPolicyNoWorkRepository struct{ UserRepository }

func (passwordPolicyNoWorkRepository) HasActiveAdmin(context.Context) (bool, error) {
	return false, nil
}

func TestPasswordPolicyRejectsBeforePasswordWork(t *testing.T) {
	guard := authlimit.New(authlimit.Config{MaxConcurrent: 1})
	release, err := guard.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	svc := NewAuthService(passwordPolicyNoWorkRepository{}, fastServiceArgon2Params(), guard)
	for _, password := range []string{"", strings.Repeat(" ", 20), "x", strings.Repeat("界", 14), "passwordpassword", strings.Repeat("x", 1025), "invalid-password\xff"} {
		for _, operation := range []struct {
			name string
			run  func() error
		}{
			{"create user", func() error {
				_, err := svc.CreateUser(context.Background(), "new@example.com", "New user", password, domain.UserRoleUser)
				return err
			}},
			{"create admin", func() error {
				_, err := svc.CreateUser(context.Background(), "admin@example.com", "Admin", password, domain.UserRoleAdmin)
				return err
			}},
			{"change password", func() error {
				return svc.ChangePassword(context.Background(), "user", "old", password)
			}},
			{"bootstrap", func() error {
				_, changed, err := svc.EnsureBootstrapAdmin(context.Background(), "admin@example.com", "Admin", password)
				if changed {
					t.Fatal("invalid bootstrap password changed credentials")
				}
				return err
			}},
		} {
			t.Run(operation.name, func(t *testing.T) {
				if err := operation.run(); !errors.Is(err, apperror.ErrInvalidArgument) {
					t.Fatalf("error = %v, want validation before occupied hash capacity or account work", err)
				}
			})
		}
	}
}

func TestPasswordPolicyPreservesCredentialsWhenSettingPasswords(t *testing.T) {
	const password = "  雨の静かな庭でゆっくりお茶を飲む  "
	for _, operation := range []string{"create", "bootstrap create", "bootstrap promote"} {
		t.Run(operation, func(t *testing.T) {
			repo := &fakeUserRepository{users: map[string]domain.User{}}
			svc := NewAuthService(repo, fastServiceArgon2Params())
			if operation == "bootstrap promote" {
				repo.users["person@example.com"] = domain.User{ID: "existing", Email: "person@example.com", Role: domain.UserRoleUser, PasswordHash: "old-hash"}
			}
			if operation == "create" {
				if _, err := svc.CreateUser(context.Background(), "person@example.com", "Person", password, domain.UserRoleUser); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, changed, err := svc.EnsureBootstrapAdmin(context.Background(), "person@example.com", "Admin", password); err != nil || !changed {
					t.Fatalf("bootstrap changed=%v, err=%v", changed, err)
				}
			}
			if _, err := svc.Login(context.Background(), "person@example.com", password); err != nil {
				t.Fatalf("original password did not verify: %v", err)
			}
			if _, err := svc.Login(context.Background(), "person@example.com", strings.TrimSpace(password)); !errors.Is(err, apperror.ErrUnauthenticated) {
				t.Fatalf("trimmed password authenticated: %v", err)
			}
		})
	}
}

func TestPasswordPolicyAllowsLegacyLoginAndPasswordUpgrade(t *testing.T) {
	for _, old := range []string{"x", "passwordpassword"} {
		t.Run(old, func(t *testing.T) {
			hash, err := security.HashPassword(old, fastServiceArgon2Params())
			if err != nil {
				t.Fatal(err)
			}
			repo := &fakeUserRepository{users: map[string]domain.User{"person@example.com": {ID: "user", Email: "person@example.com", PasswordHash: hash, MustChangePassword: true}}}
			svc := NewAuthService(repo, fastServiceArgon2Params())
			if _, err := svc.Login(context.Background(), "person@example.com", old); err != nil {
				t.Fatalf("legacy login rejected: %v", err)
			}
			const replacement = "  a peaceful garden near home  "
			if err := svc.ChangePassword(context.Background(), "user", old, replacement); err != nil {
				t.Fatalf("legacy password upgrade rejected: %v", err)
			}
			user := repo.users["person@example.com"]
			if user.MustChangePassword || len(repo.passwordUpdates) != 1 || user.PasswordHash == hash {
				t.Fatal("password upgrade did not update credentials and clear the first-login flag")
			}
			for _, candidate := range []string{old, strings.TrimSpace(replacement)} {
				if _, err := svc.Login(context.Background(), "person@example.com", candidate); !errors.Is(err, apperror.ErrUnauthenticated) {
					t.Fatalf("incorrect password accepted after upgrade: %v", err)
				}
			}
			if _, err := svc.Login(context.Background(), "person@example.com", replacement); err != nil {
				t.Fatalf("new password rejected: %v", err)
			}
		})
	}
}

func TestPasswordPolicyRejectsBootstrapPromotionWithoutChangingAccount(t *testing.T) {
	before := domain.User{ID: "existing", Email: "person@example.com", Role: domain.UserRoleUser, PasswordHash: "existing-hash"}
	repo := &fakeUserRepository{users: map[string]domain.User{before.Email: before}}
	svc := NewAuthService(repo, fastServiceArgon2Params())
	if _, changed, err := svc.EnsureBootstrapAdmin(context.Background(), before.Email, "Admin", "passwordpassword"); !errors.Is(err, apperror.ErrInvalidArgument) || changed {
		t.Fatalf("invalid bootstrap promotion changed=%v, err=%v", changed, err)
	}
	if repo.users[before.Email] != before {
		t.Fatal("invalid bootstrap promotion altered existing user")
	}
}
