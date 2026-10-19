package server

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

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
)

func sessionFixture(t *testing.T) (AuthServer, *fakeServerUserRepository, *fakeAuthSessionStore) {
	t.Helper()
	hash, err := security.HashPassword("old-password", fastServerArgon2Params())
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeServerUserRepository{users: map[string]domain.User{
		"user@example.com":  {ID: "user", Email: "user@example.com", Role: domain.UserRoleUser, PasswordHash: hash},
		"other@example.com": {ID: "other", Email: "other@example.com", Role: domain.UserRoleUser, PasswordHash: hash},
	}}
	sessions := &fakeAuthSessionStore{repository: repo}
	repo.sessions = sessions
	return NewAuthServer(service.NewAuthService(repo, fastServerArgon2Params()), sessions, 32, time.Hour, time.Now), repo, sessions
}

func sessionTransport(t *testing.T, auth AuthServer, options ...grpc.ServerOption) pb.AuthServiceClient {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	transport := grpc.NewServer(options...)
	pb.RegisterAuthServiceServer(transport, auth)
	go func() { _ = transport.Serve(listener) }()
	t.Cleanup(transport.Stop)
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewAuthServiceClient(conn)
}

func TestSessionRevocationLifecycleOverGRPC(t *testing.T) {
	auth, _, _ := sessionFixture(t)
	client := sessionTransport(t, auth)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	login := func(email, password string) string {
		t.Helper()
		response, err := client.Login(ctx, &pb.LoginRequest{Email: email, Password: password})
		if err != nil {
			t.Fatal(err)
		}
		return response.GetSessionToken()
	}
	valid := func(token string, want bool) {
		t.Helper()
		_, err := client.ValidateSession(ctx, &pb.ValidateSessionRequest{SessionToken: token})
		if want && err != nil {
			t.Fatalf("valid token rejected: %v", err)
		}
		if !want && status.Code(err) != codes.Unauthenticated {
			t.Fatalf("revoked token reusable: %v", err)
		}
	}
	first, second := login("user@example.com", "old-password"), login("user@example.com", "old-password")
	other := login("other@example.com", "old-password")
	valid(first, true)
	logoutCtx := metadata.AppendToOutgoingContext(ctx, "x-session-token", first)
	for i := 0; i < 2; i++ {
		if _, err := client.RevokeCurrentSession(logoutCtx, &pb.RevokeCurrentSessionRequest{}); err != nil {
			t.Fatal(err)
		}
	}
	valid(first, false)
	valid(second, true)
	valid(other, true)
	// Bearer metadata is supported too, including first-login password changes.
	changeCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+second)
	if _, err := client.ChangePassword(changeCtx, &pb.ChangePasswordRequest{CurrentPassword: "old-password", NewPassword: "new-password-for-tests"}); err != nil {
		t.Fatal(err)
	}
	valid(first, false)
	valid(second, false)
	valid(other, true)
	if _, err := client.UpdateMyAccount(changeCtx, &pb.UpdateMyAccountRequest{Email: "stolen@example.com", DisplayName: "Stolen"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("revoked token authorized a business RPC: %v", err)
	}
	if _, err := client.Login(ctx, &pb.LoginRequest{Email: "user@example.com", Password: "old-password"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("old password still works: %v", err)
	}
	valid(login("user@example.com", "new-password-for-tests"), true)
}

func TestSessionRevocationPasswordFailuresLeaveSessionsIntact(t *testing.T) {
	for _, failedStore := range []bool{false, true} {
		auth, repo, sessions := sessionFixture(t)
		login, err := auth.Login(context.Background(), &pb.LoginRequest{Email: "user@example.com", Password: "old-password"})
		if err != nil {
			t.Fatal(err)
		}
		before := repo.users["user@example.com"].PasswordHash
		current := "wrong-password"
		expected := codes.Unauthenticated
		if failedStore {
			current, expected = "old-password", codes.Internal
			repo.passwordError = errors.New("database update failed")
		}
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-session-token", login.GetSessionToken()))
		_, err = auth.ChangePassword(ctx, &pb.ChangePasswordRequest{CurrentPassword: current, NewPassword: "new-password-for-tests"})
		if status.Code(err) != expected || repo.users["user@example.com"].PasswordHash != before || len(sessions.revokedUsers) != 0 {
			t.Fatalf("failed change altered credentials/sessions: %v", err)
		}
		if _, err := auth.ValidateSession(context.Background(), &pb.ValidateSessionRequest{SessionToken: login.GetSessionToken()}); err != nil {
			t.Fatalf("failed password change revoked session: %v", err)
		}
	}
}

func TestSessionRevocationRequiresTokenAndIsIdempotent(t *testing.T) {
	auth, _, _ := sessionFixture(t)
	if _, err := auth.RevokeCurrentSession(context.Background(), &pb.RevokeCurrentSessionRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing token accepted: %v", err)
	}
	if _, err := auth.RevokeCurrentSession(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil request accepted: %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-session-token", "already-invalid-token"))
	if _, err := auth.RevokeCurrentSession(ctx, &pb.RevokeCurrentSessionRequest{}); err != nil {
		t.Fatalf("unknown token should be safely idempotent: %v", err)
	}
}

type staleLoginSessionStore struct{ *fakeAuthSessionStore }

func (s staleLoginSessionStore) CreateSession(ctx context.Context, userID domain.ID, verifiedHash, tokenHash string, expiresAt time.Time) error {
	if err := s.repository.UpdateUserPassword(ctx, userID, verifiedHash, "changed-while-login-was-running", false); err != nil {
		return err
	}
	return s.fakeAuthSessionStore.CreateSession(ctx, userID, verifiedHash, tokenHash, expiresAt)
}

func TestSessionRevocationRejectsStaleLogin(t *testing.T) {
	auth, _, sessions := sessionFixture(t)
	auth.sessions = staleLoginSessionStore{sessions}
	if _, err := auth.Login(context.Background(), &pb.LoginRequest{Email: "user@example.com", Password: "old-password"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("login minted a token after password change: %v", err)
	}
	if len(sessions.usersByToken) != 0 {
		t.Fatal("stale login created a session")
	}
}

type missingSessionStore struct{ *fakeAuthSessionStore }

func (s missingSessionStore) GetUserBySessionTokenHash(context.Context, string, time.Time) (domain.User, error) {
	return domain.User{}, apperror.ErrNotFound
}

func TestSessionRevocationMissingStoredTokenIsUnauthenticated(t *testing.T) {
	auth, _, sessions := sessionFixture(t)
	auth.sessions = missingSessionStore{sessions}
	_, err := auth.ValidateSession(context.Background(), &pb.ValidateSessionRequest{SessionToken: "revoked-or-expired"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing stored token did not invalidate the session: %v", err)
	}
}
