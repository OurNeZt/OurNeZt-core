package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/authlimit"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
)

type loginLookupRepository struct {
	UserRepository
	user domain.User
	err  error
}

func (r loginLookupRepository) GetUserByEmail(context.Context, string) (domain.User, error) {
	return r.user, r.err
}

func TestLoginFailurePasswordWork(t *testing.T) {
	params := fastServiceArgon2Params()
	hash, err := security.HashPassword("correct", params)
	if err != nil {
		t.Fatal(err)
	}
	dummy := security.DummyPasswordHash(params)
	disabled := time.Now()
	for _, tc := range []struct {
		name     string
		user     domain.User
		lookup   error
		password string
		work     []string
	}{
		{"unknown", domain.User{}, apperror.ErrNotFound, "wrong", []string{dummy}},
		{"lookup failure", domain.User{}, errors.New("private database error"), "wrong", []string{dummy}},
		{"partial lookup result", domain.User{ID: "user", PasswordHash: hash}, apperror.ErrNotFound, "correct", []string{dummy}},
		{"wrong password", domain.User{ID: "user", PasswordHash: hash}, nil, "wrong", []string{hash}},
		{"disabled correct password", domain.User{ID: "user", PasswordHash: hash, DisabledAt: &disabled}, nil, "correct", []string{hash}},
		{"disabled wrong password", domain.User{ID: "user", PasswordHash: hash, DisabledAt: &disabled}, nil, "wrong", []string{hash}},
		{"malformed stored hash", domain.User{ID: "user", PasswordHash: "invalid"}, nil, "correct", []string{"invalid", dummy}},
		{"zero iterations", domain.User{ID: "user", PasswordHash: "$argon2id$v=19$m=1024,t=0,p=1$YWJj$ZGVm"}, nil, "correct", []string{"$argon2id$v=19$m=1024,t=0,p=1$YWJj$ZGVm", dummy}},
		{"empty stored key", domain.User{ID: "user", PasswordHash: "$argon2id$v=19$m=1024,t=1,p=1$YWJj$"}, nil, "correct", []string{"$argon2id$v=19$m=1024,t=1,p=1$YWJj$", dummy}},
		{"empty password unknown", domain.User{}, apperror.ErrNotFound, "", []string{dummy}},
		{"empty password known", domain.User{ID: "user", PasswordHash: hash}, nil, "", []string{hash}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewAuthService(loginLookupRepository{user: tc.user, err: tc.lookup}, params)
			var work []string
			svc.verifyPassword = func(password, target string) (bool, error) {
				if password != tc.password {
					t.Fatal("verification did not receive the submitted password")
				}
				work = append(work, target)
				return security.VerifyPassword(password, target)
			}
			user, err := svc.Login(context.Background(), "person@example.com", tc.password)
			if err != apperror.ErrUnauthenticated || user != (domain.User{}) {
				t.Fatalf("failure returned account information: user=%+v err=%v", user, err)
			}
			if !reflect.DeepEqual(work, tc.work) {
				t.Fatalf("password verification targets = %v, want %v", work, tc.work)
			}
		})
	}
}

func TestLoginDummyMatchNeverAuthenticates(t *testing.T) {
	svc := NewAuthService(loginLookupRepository{err: apperror.ErrNotFound}, fastServiceArgon2Params())
	svc.verifyPassword = func(string, string) (bool, error) { return true, nil }
	user, err := svc.Login(context.Background(), "unknown@example.com", "password")
	if err != apperror.ErrUnauthenticated || user != (domain.User{}) {
		t.Fatalf("dummy match authorized login: user=%+v err=%v", user, err)
	}
}

func TestLoginDummyWorkRespectsLimits(t *testing.T) {
	guard := authlimit.New(authlimit.Config{AccountLimit: 1, MaxConcurrent: 1})
	repo := &countingAuthRepository{}
	svc := NewAuthService(repo, fastServiceArgon2Params(), guard)
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	svc.verifyPassword = func(password, hash string) (bool, error) {
		close(entered)
		<-resume
		return security.VerifyPassword(password, hash)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		defer close(done)
		_, _ = svc.Login(ctx, "unknown@example.com", "password")
	}()
	defer func() { close(resume); <-done }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("unknown login did not enter dummy password verification")
	}
	cancel()
	for _, email := range []string{"unknown@example.com", "another@example.com"} {
		_, err := svc.Login(context.Background(), email, "password")
		if !errors.Is(err, authlimit.ErrLimited) {
			t.Fatalf("dummy verification bypassed rate/concurrency limits: %v", err)
		}
	}
	if repo.lookups != 1 {
		t.Fatalf("limited requests reached lookup: %d", repo.lookups)
	}
}
