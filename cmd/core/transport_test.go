package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestTransportStartupHelper(t *testing.T) {
	if os.Getenv("OURNEZT_TEST_TRANSPORT_STARTUP") == "1" {
		main()
	}
}

func TestTransportStartupRejectsBeforeDatabase(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, allow, cert, key, want string }{
		{"missing TLS", "false", "", "", "production requires gRPC TLS"},
		{"invalid opt-in", "tru", "", "", "GRPC_ALLOW_INSECURE must be a boolean"},
		{"partial pair with opt-in", "true", "missing-cert.pem", "", "must be configured together"},
		{"bad pair with opt-in", "true", "missing-cert.pem", "missing-key.pem", "load gRPC TLS certificate/key"},
		{"trusted plaintext warning", "true", "", "", "plaintext gRPC explicitly enabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, executable, "-test.run=^TestTransportStartupHelper$")
			command.Env = append(os.Environ(), "OURNEZT_TEST_TRANSPORT_STARTUP=1", "APP_ENV=production", "DATABASE_URL=", "GRPC_ALLOW_INSECURE="+tc.allow, "GRPC_TLS_CERT_FILE="+tc.cert, "GRPC_TLS_KEY_FILE="+tc.key)
			output, err := command.CombinedOutput()
			var exited *exec.ExitError
			if !errors.As(err, &exited) || exited.ExitCode() != 1 || !strings.Contains(string(output), tc.want) {
				t.Fatalf("unexpected startup exit: err=%v logs=%s", err, output)
			}
			if tc.cert != "" || tc.allow != "true" {
				if strings.Contains(string(output), "connect database") {
					t.Fatal("invalid transport reached the database before being rejected")
				}
			} else if !strings.Contains(string(output), "connect database") || !strings.Contains(string(output), `"level":"WARN"`) {
				t.Fatal("explicit plaintext did not pass policy with a clear warning")
			}
		})
	}
}
