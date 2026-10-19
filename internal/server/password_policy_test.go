package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestPasswordPolicyOverGRPC(t *testing.T) {
	auth, repo, sessions := sessionFixture(t)
	admin := repo.users["other@example.com"]
	admin.Role = domain.UserRoleAdmin
	repo.users[admin.Email] = admin
	user := repo.users["user@example.com"]
	user.MustChangePassword = true
	repo.users[user.Email] = user
	client := sessionTransport(t, auth, grpc.UnaryInterceptor(InputLimitInterceptor))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	login := func(email, password string) string {
		t.Helper()
		resp, err := client.Login(ctx, &pb.LoginRequest{Email: email, Password: password})
		if err != nil {
			t.Fatalf("login rejected: %v", err)
		}
		return resp.GetSessionToken()
	}
	// Legacy passwords remain valid, including for first-login password changes.
	adminCtx := metadata.AppendToOutgoingContext(ctx, "x-session-token", login(admin.Email, "old-password"))
	token := login(user.Email, "old-password")
	userCtx := metadata.AppendToOutgoingContext(ctx, "x-session-token", token)
	for _, tc := range []struct{ name, password, reason string }{
		{"blank", "", "blank or whitespace-only"},
		{"whitespace", strings.Repeat(" ", 20), "blank or whitespace-only"},
		{"short", "x", "at least 15 characters"},
		{"unicode short", strings.Repeat("界", 14), "at least 15 characters"},
		{"too long", strings.Repeat("a", 1025), "must not exceed 1024 bytes"},
		{"unicode too long", strings.Repeat("界", 342), "must not exceed 1024 bytes"},
		{"common", "passwordpassword", "too common"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			created, createErr := client.CreateUser(adminCtx, &pb.CreateUserRequest{Email: "new@example.com", Password: tc.password})
			changed, changeErr := client.ChangePassword(userCtx, &pb.ChangePasswordRequest{CurrentPassword: "old-password", NewPassword: tc.password})
			for _, err := range []error{createErr, changeErr} {
				st := status.Convert(err)
				if st.Code() != codes.InvalidArgument || !strings.Contains(st.Message(), tc.reason) || len(st.Details()) != 0 {
					t.Fatalf("unexpected validation response: %v", err)
				}
				if tc.password != "" && strings.Contains(st.Message(), tc.password) {
					t.Fatal("validation response exposed the password")
				}
			}
			if created != nil || changed != nil || repo.created.ID != "" || len(repo.users) != 2 || repo.users[user.Email] != user || len(sessions.revokedUsers) != 0 {
				t.Fatal("rejected password changed an account or its sessions")
			}
			if _, err := client.ValidateSession(ctx, &pb.ValidateSessionRequest{SessionToken: token}); err != nil {
				t.Fatalf("invalid password revoked the current session: %v", err)
			}
		})
	}
	const replacement = "  a peaceful garden near home  "
	if _, err := client.ChangePassword(userCtx, &pb.ChangePasswordRequest{CurrentPassword: "old-password", NewPassword: replacement}); err != nil {
		t.Fatalf("valid password change rejected: %v", err)
	}
	if _, err := client.ValidateSession(ctx, &pb.ValidateSessionRequest{SessionToken: token}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("successful change did not revoke the session: %v", err)
	}
	resp, err := client.Login(ctx, &pb.LoginRequest{Email: user.Email, Password: replacement})
	if err != nil || resp.GetUser().GetMustChangePassword() {
		t.Fatalf("updated password or first-login state is incorrect: %v", err)
	}
	if _, err := client.Login(ctx, &pb.LoginRequest{Email: user.Email, Password: strings.TrimSpace(replacement)}); status.Code(err) != codes.Unauthenticated || status.Convert(err).Message() != "Invalid email or password." {
		t.Fatalf("password spaces lost or login failure changed: %v", err)
	}
	created, err := client.CreateUser(adminCtx, &pb.CreateUserRequest{Email: "new@example.com", Password: "quiet otter bay"})
	if err != nil || created.GetEmail() != "new@example.com" {
		t.Fatalf("minimum-length passphrase was not provisioned: %v", err)
	}
	login("new@example.com", "quiet otter bay")
}
