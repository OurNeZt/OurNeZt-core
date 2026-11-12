package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

func TestResourceOversizedCredentialsSkipPasswordWork(t *testing.T) {
	repo := &countingAuthRepository{}
	svc := NewAuthService(repo, fastServiceArgon2Params())
	svc.verifyPassword = func(string, string) (bool, error) {
		t.Fatal("oversized password reached verification")
		return false, nil
	}
	password := strings.Repeat("x", 1025)
	if _, err := svc.Login(context.Background(), "person@example.com", password); !errors.Is(err, apperror.ErrUnauthenticated) {
		t.Fatal(err)
	}
	if repo.lookups != 0 {
		t.Fatal("oversized login reached repository")
	}
	if _, err := svc.CreateUser(context.Background(), "person@example.com", "Name", password, domain.UserRoleUser); !errors.Is(err, apperror.ErrInvalidArgument) {
		t.Fatal(err)
	}
	if err := svc.ChangePassword(context.Background(), "user", "current", password); !errors.Is(err, apperror.ErrInvalidArgument) {
		t.Fatal(err)
	}
}
