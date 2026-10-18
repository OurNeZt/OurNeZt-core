package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/authlimit"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/service"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type failingLoginRepository struct {
	service.UserRepository
	err error
}

func (r failingLoginRepository) GetUserByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, r.err
}

type failingLoginSessionStore struct {
	AuthSessionStore
	err error
}

func (s failingLoginSessionStore) CreateSession(context.Context, domain.ID, string, string, time.Time) error {
	return s.err
}

func TestLoginFailuresHaveIdenticalStatus(t *testing.T) {
	params := fastServerArgon2Params()
	hash, err := security.HashPassword("correct", params)
	if err != nil {
		t.Fatal(err)
	}
	disabled := time.Now()
	repo := &fakeServerUserRepository{users: map[string]domain.User{
		"active@example.com":   {ID: "active", PasswordHash: hash},
		"disabled@example.com": {ID: "disabled", PasswordHash: hash, DisabledAt: &disabled},
		"broken@example.com":   {ID: "broken", PasswordHash: "malformed"},
	}}
	for _, tc := range []struct {
		name     string
		email    string
		password string
		lookup   error
		session  error
	}{
		{name: "unknown", email: "unknown@example.com", password: "wrong"},
		{name: "wrong password", email: "active@example.com", password: "wrong"},
		{name: "disabled correct password", email: "disabled@example.com", password: "correct"},
		{name: "disabled wrong password", email: "disabled@example.com", password: "wrong"},
		{name: "malformed hash", email: "broken@example.com", password: "correct"},
		{name: "empty credentials"},
		{name: "lookup error", lookup: errors.New("private database detail")},
		{name: "lookup grpc error", lookup: status.Error(codes.NotFound, "private account detail")},
		{name: "session stale password", email: "active@example.com", password: "correct", session: apperror.ErrUnauthenticated},
		{name: "session disabled", email: "active@example.com", password: "correct", session: apperror.ErrDisabledUser},
		{name: "session missing user", email: "active@example.com", password: "correct", session: apperror.ErrNotFound},
		{name: "session store error", email: "active@example.com", password: "correct", session: errors.New("private storage detail")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var users service.UserRepository = repo
			if tc.lookup != nil {
				users = failingLoginRepository{err: tc.lookup}
			}
			sessions := &fakeAuthSessionStore{}
			var store AuthSessionStore = sessions
			if tc.session != nil {
				store = failingLoginSessionStore{err: tc.session}
			}
			srv := NewAuthServer(service.NewAuthService(users, params), store, 32, time.Hour, nil)
			response, err := srv.Login(context.Background(), &pb.LoginRequest{Email: tc.email, Password: tc.password})
			st := status.Convert(err)
			if response != nil || st.Code() != codes.Unauthenticated || st.Message() != "Invalid email or password." || len(st.Details()) != 0 {
				t.Fatalf("login failure leaked state: response=%v status=%v", response, st)
			}
			if sessions.created.userID != "" {
				t.Fatal("failed authentication created a session")
			}
		})
	}
}

func TestLoginPreservesTrafficLimitsAndCancellation(t *testing.T) {
	guard := authlimit.New(authlimit.Config{AccountLimit: 1})
	srv := NewAuthServer(service.NewAuthService(&fakeServerUserRepository{}, fastServerArgon2Params(), guard), &fakeAuthSessionStore{}, 32, time.Hour, nil)
	_, _ = srv.Login(context.Background(), &pb.LoginRequest{Email: "unknown@example.com", Password: "wrong"})
	_, err := srv.Login(context.Background(), &pb.LoginRequest{Email: "unknown@example.com", Password: "wrong"})
	st := status.Convert(err)
	if st.Code() != codes.ResourceExhausted || len(st.Details()) != 1 {
		t.Fatalf("traffic limit was masked: %v", st)
	}
	if retry, ok := st.Details()[0].(*errdetails.RetryInfo); !ok || retry.GetRetryDelay().AsDuration() <= 0 {
		t.Fatal("missing retry delay")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = srv.Login(ctx, &pb.LoginRequest{})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("cancellation was masked: %v", err)
	}
}
