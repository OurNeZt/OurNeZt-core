package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/authlimit"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// A missing/expired/revoked session produces ErrNotFound in PostgreSQL.
type missingCreateUserSession struct{ AuthSessionStore }

func (missingCreateUserSession) GetUserBySessionTokenHash(context.Context, string, time.Time) (domain.User, error) {
	return domain.User{}, fmt.Errorf("session lookup: %w", apperror.ErrNotFound)
}

func TestCreateUserRequiresAdminBeforePasswordWork(t *testing.T) {
	disabledAt := time.Now()
	cases := []struct {
		name           string
		token          string
		user           domain.User
		missingSession bool
		want           codes.Code
	}{
		{name: "anonymous", want: codes.Unauthenticated},
		{name: "invalid session", token: "invalid", want: codes.Unauthenticated},
		{name: "expired or revoked session", token: "old", missingSession: true, want: codes.Unauthenticated},
		{name: "ordinary user", token: "member", user: domain.User{ID: "member", Role: domain.UserRoleUser}, want: codes.PermissionDenied},
		{name: "unknown caller role", token: "unknown-role", user: domain.User{ID: "unknown", Role: "unknown"}, want: codes.PermissionDenied},
		{name: "disabled admin", token: "disabled", user: domain.User{ID: "admin", Role: domain.UserRoleAdmin, DisabledAt: &disabledAt}, want: codes.Unauthenticated},
		{name: "admin must change password", token: "forced", user: domain.User{ID: "admin", Role: domain.UserRoleAdmin, MustChangePassword: true}, want: codes.FailedPrecondition},
	}
	for _, tc := range cases {
		for _, role := range []string{"", "user", "admin", "invalid"} {
			t.Run(tc.name+"/role="+role, func(t *testing.T) {
				repo := &fakeServerUserRepository{}
				limits := authlimit.New(authlimit.Config{MaxConcurrent: 1})
				// Occupied password capacity must not mask an authorization error.
				release, err := limits.Acquire(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				defer release()
				var sessions AuthSessionStore = &fakeAuthSessionStore{usersByToken: map[string]domain.User{}}
				if tc.user.ID != "" {
					sessions.(*fakeAuthSessionStore).usersByToken[security.HashToken(tc.token)] = tc.user
				}
				if tc.missingSession {
					sessions = missingCreateUserSession{sessions}
				}
				srv := NewAuthServer(service.NewAuthService(repo, fastServerArgon2Params(), limits), sessions, 32, time.Hour, nil)
				ctx := context.Background()
				if tc.token != "" {
					ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-session-token", tc.token))
				}
				resp, err := srv.CreateUser(ctx, &pb.CreateUserRequest{Email: "new@example.com", DisplayName: "New user", Password: "temporary-password", Role: role})
				if status.Code(err) != tc.want {
					t.Fatalf("code=%s, want %s: %v", status.Code(err), tc.want, err)
				}
				if resp != nil || repo.created.ID != "" || len(repo.users) != 0 {
					t.Fatal("unauthorized caller created a user")
				}
			})
		}
	}
}

func TestCreateUserAdminCanProvisionAllSupportedRoles(t *testing.T) {
	for _, role := range []string{"", "user", "admin"} {
		for _, header := range []string{"x-session-token", "authorization"} {
			t.Run("role="+role+"/"+header, func(t *testing.T) {
				repo := &fakeServerUserRepository{}
				sessions := &fakeAuthSessionStore{usersByToken: map[string]domain.User{security.HashToken("admin-token"): {ID: "admin", Role: domain.UserRoleAdmin}}}
				srv := NewAuthServer(service.NewAuthService(repo, fastServerArgon2Params()), sessions, 32, time.Hour, nil)
				value := "admin-token"
				if header == "authorization" {
					value = "Bearer " + value
				}
				ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(header, value))
				resp, err := srv.CreateUser(ctx, &pb.CreateUserRequest{Email: " NEW@example.com ", DisplayName: " New user ", Password: "temporary-password", Role: role})
				if err != nil {
					t.Fatal(err)
				}
				wantRole := role
				if wantRole == "" {
					wantRole = "user"
				}
				if resp.GetRole() != wantRole || resp.GetEmail() != "new@example.com" || repo.created.DisplayName != "New user" {
					t.Fatalf("unexpected created user: %v", resp)
				}
				if resp.GetMustChangePassword() != (role == "admin") {
					t.Fatal("admin password-change requirement changed")
				}
				ok, err := security.VerifyPassword("temporary-password", repo.created.PasswordHash)
				if err != nil || !ok {
					t.Fatalf("password not hashed correctly: %v", err)
				}
			})
		}
	}
}

func TestCreateUserDeniedAttemptsDoNotConsumeAccountBudget(t *testing.T) {
	repo := &fakeServerUserRepository{}
	sessions := &fakeAuthSessionStore{usersByToken: map[string]domain.User{security.HashToken("admin-token"): {ID: "admin", Role: domain.UserRoleAdmin}}}
	limits := authlimit.New(authlimit.Config{AccountLimit: 1})
	srv := NewAuthServer(service.NewAuthService(repo, fastServerArgon2Params(), limits), sessions, 32, time.Hour, nil)
	req := &pb.CreateUserRequest{Email: "new@example.com", Password: "temporary-password"}
	for range 6 {
		if _, err := srv.CreateUser(context.Background(), req); status.Code(err) != codes.Unauthenticated {
			t.Fatalf("unexpected anonymous response: %v", err)
		}
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-session-token", "admin-token"))
	if _, err := srv.CreateUser(ctx, req); err != nil {
		t.Fatalf("unauthorized attempts consumed account budget: %v", err)
	}
}

func TestCreateUserAuthorizationOverGRPC(t *testing.T) {
	repo := &fakeServerUserRepository{}
	sessions := &fakeAuthSessionStore{usersByToken: map[string]domain.User{
		security.HashToken("member-token"): {ID: "member", Role: domain.UserRoleUser},
		security.HashToken("admin-token"):  {ID: "admin", Role: domain.UserRoleAdmin},
	}}
	limits := authlimit.New(authlimit.Config{})
	srv := NewAuthServer(service.NewAuthService(repo, fastServerArgon2Params(), limits), sessions, 32, time.Hour, nil)
	listener := bufconn.Listen(1024 * 1024)
	transport := grpc.NewServer(grpc.ChainUnaryInterceptor(AuthRateLimitInterceptor(limits, nil)))
	pb.RegisterAuthServiceServer(transport, srv)
	go func() { _ = transport.Serve(listener) }()
	t.Cleanup(func() { transport.Stop(); _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///auth-test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := pb.NewAuthServiceClient(conn)
	for _, role := range []string{"", "user", "admin"} {
		for _, caller := range []struct {
			token string
			want  codes.Code
		}{{"", codes.Unauthenticated}, {"member-token", codes.PermissionDenied}, {"admin-token", codes.OK}} {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if caller.token != "" {
				ctx = metadata.AppendToOutgoingContext(ctx, "x-session-token", caller.token)
			}
			_, err := client.CreateUser(ctx, &pb.CreateUserRequest{Email: fmt.Sprintf("new-%s@example.com", role), Password: "temporary-password", Role: role})
			cancel()
			if status.Code(err) != caller.want {
				t.Fatalf("caller=%q role=%q: code=%s, want %s: %v", caller.token, role, status.Code(err), caller.want, err)
			}
		}
	}
	if len(repo.users) != 3 {
		t.Fatalf("created %d users, want exactly the 3 admin-authorized users", len(repo.users))
	}
}
