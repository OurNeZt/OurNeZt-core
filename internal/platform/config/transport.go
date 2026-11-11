package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func (c Config) IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(c.AppEnv), "production")
}

// ValidateTransport checks policy before any database connection or listener is
// opened. Supplying only one TLS file must never select plaintext implicitly.
func (c Config) ValidateTransport() error {
	if c.transportError != nil {
		return c.transportError
	}
	cert := strings.TrimSpace(c.GRPCTLSCertFile) != ""
	key := strings.TrimSpace(c.GRPCTLSKeyFile) != ""
	if cert != key {
		return fmt.Errorf("GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE must be configured together")
	}
	if c.IsProduction() && !cert && !c.GRPCAllowInsecure {
		return fmt.Errorf("production requires gRPC TLS: configure GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE; set GRPC_ALLOW_INSECURE=true only for a trusted internal network or TLS-terminating proxy")
	}
	return nil
}

func transportBooleanError(keys ...string) error {
	for _, key := range keys {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		if value == "" {
			continue
		}
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("%s must be a boolean (true or false)", key)
		}
	}
	return nil
}
