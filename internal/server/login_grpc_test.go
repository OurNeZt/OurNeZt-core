package server

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type grpcLoginRepository struct{ service.UserRepository }

func (r grpcLoginRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	if email == "lookup-failure@example.com" {
		return domain.User{}, errors.New("private database detail")
	}
	return r.UserRepository.GetUserByEmail(ctx, email)
}

func TestLoginGRPCFailureResponses(t *testing.T) {
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
	sessions := &fakeAuthSessionStore{}
	auth := NewAuthServer(service.NewAuthService(grpcLoginRepository{repo}, params), sessions, 32, time.Hour, nil)
	listener := bufconn.Listen(1 << 20)
	t.Cleanup(func() { _ = listener.Close() })
	transport := grpc.NewServer()
	pb.RegisterAuthServiceServer(transport, &auth)
	go func() { _ = transport.Serve(listener) }()
	t.Cleanup(transport.Stop)
	conn, err := grpc.NewClient("passthrough:///login-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := pb.NewAuthServiceClient(conn)
	for _, tc := range []struct{ name, email, password string }{
		{"unknown email", "unknown@example.com", "wrong"},
		{"wrong password", "active@example.com", "wrong"},
		{"disabled correct password", "disabled@example.com", "correct"},
		{"disabled wrong password", "disabled@example.com", "wrong"},
		{"lookup failure", "lookup-failure@example.com", "wrong"},
		{"malformed stored hash", "broken@example.com", "wrong"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			response, err := client.Login(ctx, &pb.LoginRequest{Email: tc.email, Password: tc.password})
			st := status.Convert(err)
			if response != nil || st.Code() != codes.Unauthenticated || st.Message() != "Invalid email or password." || len(st.Details()) != 0 {
				t.Fatalf("gRPC exposed login state: response=%v status=%v", response, st)
			}
			if sessions.created.userID != "" {
				t.Fatal("failed gRPC login created a session")
			}
		})
	}
	// A working success path prevents a broken or disconnected server from
	// appearing to pass merely because every attempt failed.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	response, err := client.Login(ctx, &pb.LoginRequest{Email: "active@example.com", Password: "correct"})
	if err != nil || response.GetUser().GetId() != "active" || response.GetSessionToken() == "" || sessions.created.userID != "active" {
		t.Fatalf("valid gRPC login failed: response=%v err=%v", response, err)
	}
}
