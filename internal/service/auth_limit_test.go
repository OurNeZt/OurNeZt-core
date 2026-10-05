package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/authlimit"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
)

type countingAuthRepository struct {
	UserRepository
	lookups int
	entered chan struct{}
	resume  chan struct{}
}

func (r *countingAuthRepository) GetUserByEmail(context.Context, string) (domain.User, error) {
	r.lookups++
	if r.entered != nil {
		close(r.entered)
		<-r.resume
	}
	return domain.User{}, apperror.ErrNotFound
}

func TestLoginRateLimitRunsBeforeRepositoryLookup(t *testing.T) {
	repo := &countingAuthRepository{}
	svc := NewAuthService(repo, security.DefaultArgon2Params(), authlimit.New(authlimit.Config{AccountLimit: 2}))
	for _, email := range []string{" Person@Example.com ", "person@example.com"} {
		_, _ = svc.Login(context.Background(), email, "wrong")
	}
	_, err := svc.Login(context.Background(), "PERSON@example.com", "wrong")
	if !errors.Is(err, authlimit.ErrLimited) || repo.lookups != 2 {
		t.Fatalf("lookups=%d, err=%v", repo.lookups, err)
	}
	_, _ = svc.Login(context.Background(), "other@example.com", "wrong")
	if repo.lookups != 3 {
		t.Fatal("unrelated account was blocked")
	}
}

func TestAllPasswordOperationsShareConcurrencyCap(t *testing.T) {
	repo := &countingAuthRepository{UserRepository: &fakeUserRepository{}, entered: make(chan struct{}), resume: make(chan struct{})}
	guard := authlimit.New(authlimit.Config{MaxConcurrent: 1})
	svc := NewAuthService(repo, security.DefaultArgon2Params(), guard)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); _, _ = svc.Login(ctx, "a@example.com", "password") }()
	select {
	case <-repo.entered:
	case <-time.After(time.Second):
		t.Fatal("login did not enter repository")
	}
	defer func() { close(repo.resume); <-done }()
	// Cancellation must not free a slot while password work could still be running.
	cancel()
	copyOfService := svc
	_, loginErr := copyOfService.Login(context.Background(), "b@example.com", "password")
	_, createErr := copyOfService.CreateUser(context.Background(), "c@example.com", "Name", "password", domain.UserRoleUser)
	changeErr := copyOfService.ChangePassword(context.Background(), "user", "old", "new")
	_, _, bootstrapErr := copyOfService.EnsureBootstrapAdmin(context.Background(), "d@example.com", "Admin", "password")
	for _, err := range []error{loginErr, createErr, changeErr, bootstrapErr} {
		if !errors.Is(err, authlimit.ErrLimited) {
			t.Fatalf("expensive operation bypassed cap: %v", err)
		}
	}
}
