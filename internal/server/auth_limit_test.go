package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/authlimit"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/service"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptorLimitsTransportIPAcrossMethodsAndIgnoresSpoofing(t *testing.T) {
	guard := authlimit.New(authlimit.Config{IPLimit: 2})
	interceptor := AuthRateLimitInterceptor(guard, nil)
	calls := 0
	handler := func(context.Context, any) (any, error) { calls++; return nil, nil }
	methods := []string{pb.AuthService_Login_FullMethodName, pb.AuthService_CreateUser_FullMethodName, pb.AuthService_ChangePassword_FullMethodName}
	for i, method := range methods {
		ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1000 + i}})
		ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-forwarded-for", net.IPv4(198, 51, 100, byte(i+1)).String()))
		_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: method}, handler)
		if i < 2 && err != nil {
			t.Fatal(err)
		}
		if i == 2 && status.Code(err) != codes.ResourceExhausted {
			t.Fatalf("IP limit bypassed: %v", err)
		}
		if i == 2 {
			details := status.Convert(err).Details()
			if len(details) != 1 {
				t.Fatalf("missing retry information: %v", err)
			}
			info, ok := details[0].(*errdetails.RetryInfo)
			if !ok || info.GetRetryDelay().AsDuration() <= 0 {
				t.Fatalf("invalid retry details: %v", details)
			}
		}
	}
	if calls != 2 {
		t.Fatalf("handler called %d times", calls)
	}
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.2")}})
	if _, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: methods[0]}, handler); err != nil {
		t.Fatal("other peer blocked", err)
	}
	if _, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: pb.AuthService_ValidateSession_FullMethodName}, handler); err != nil {
		t.Fatal("session validation blocked", err)
	}
}

func TestLoginReturnsGenericCredentialFailures(t *testing.T) {
	hash, err := security.HashPassword("correct", security.Argon2Params{MemoryKB: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32})
	if err != nil {
		t.Fatal(err)
	}
	disabledAt := time.Now()
	repo := &fakeServerUserRepository{users: map[string]domain.User{
		"active@example.com":   {ID: "active", PasswordHash: hash},
		"disabled@example.com": {ID: "disabled", DisabledAt: &disabledAt},
	}}
	srv := NewAuthServer(service.NewAuthService(repo, security.DefaultArgon2Params()), &fakeAuthSessionStore{}, 32, time.Hour, nil)
	for _, email := range []string{"missing@example.com", "disabled@example.com", "active@example.com"} {
		_, err := srv.Login(context.Background(), &pb.LoginRequest{Email: email, Password: "wrong"})
		if status.Code(err) != codes.Unauthenticated || status.Convert(err).Message() != "unauthenticated" {
			t.Fatalf("credential details leaked for %s: %v", email, err)
		}
	}
}
