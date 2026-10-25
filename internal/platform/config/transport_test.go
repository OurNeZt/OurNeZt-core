package config

import (
	"strings"
	"testing"
)

func TestTransportProductionConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, environment, allow, cert, key, want string
	}{
		{"development default", "development", "", "", "", ""},
		{"production default", "production", "", "", "", "GRPC_ALLOW_INSECURE=true"},
		{"production normalized", " PRODUCTION ", "false", "", "", "production requires"},
		{"invalid opt-in fails closed", "production", "tru", "", "", "GRPC_ALLOW_INSECURE must be a boolean"},
		{"explicit production plaintext", "production", "true", "", "", ""},
		{"production TLS", "production", "false", "cert.pem", "key.pem", ""},
		{"certificate only", "development", "true", "cert.pem", "", "configured together"},
		{"key only", "production", "true", "", "key.pem", "configured together"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tc.environment)
			t.Setenv("GRPC_ALLOW_INSECURE", tc.allow)
			t.Setenv("GRPC_TLS_CERT_FILE", tc.cert)
			t.Setenv("GRPC_TLS_KEY_FILE", tc.key)
			cfg := Load()
			err := cfg.ValidateTransport()
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("ValidateTransport = %v, want %q", err, tc.want)
			}
			if cfg.GRPCAllowInsecure != (tc.allow == "true") {
				t.Fatal("insecure transport was not an explicit opt-in")
			}
		})
	}
}
