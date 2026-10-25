package transport

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/platform/config"
	"google.golang.org/grpc/credentials"
)

// ServerCredentials validates policy and local certificate material before
// startup has side effects. A nil result explicitly selects allowed plaintext.
func ServerCredentials(cfg config.Config, logger *slog.Logger) (credentials.TransportCredentials, error) {
	if err := cfg.ValidateTransport(); err != nil {
		return nil, err
	}
	if certFile := strings.TrimSpace(cfg.GRPCTLSCertFile); certFile != "" {
		certificate, err := tls.LoadX509KeyPair(certFile, strings.TrimSpace(cfg.GRPCTLSKeyFile))
		if err != nil {
			return nil, fmt.Errorf("load gRPC TLS certificate/key: %w", err)
		}
		return credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}}), nil
	}
	if cfg.GRPCAllowInsecure {
		if logger == nil {
			logger = slog.Default()
		}
		logger.Warn("plaintext gRPC explicitly enabled; bearer session tokens are unencrypted; restrict access to a trusted internal network or TLS-terminating proxy", "setting", "GRPC_ALLOW_INSECURE", "env", cfg.AppEnv)
	}
	return nil, nil
}
