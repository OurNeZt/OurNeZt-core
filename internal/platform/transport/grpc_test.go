package transport

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/platform/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func transportCertificate(t *testing.T) (string, string, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "core.test"},
		DNSNames: []string{"core.test", "localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	certFile, keyFile := filepath.Join(directory, "cert.pem"), filepath.Join(directory, "key.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certPEM) {
		t.Fatal("test certificate was not loaded")
	}
	return certFile, keyFile, roots
}

func TestTransportServerCredentialsFailClosed(t *testing.T) {
	cert, key, _ := transportCertificate(t)
	_, otherKey, _ := transportCertificate(t)
	bad := filepath.Join(t.TempDir(), "bad.pem")
	if err := os.WriteFile(bad, []byte("invalid certificate material"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		cfg       config.Config
		wantError bool
		wantTLS   bool
		wantWarn  bool
	}{
		{"production default", config.Config{AppEnv: "production"}, true, false, false},
		{"development default", config.Config{AppEnv: "development"}, false, false, false},
		{"explicit production plaintext", config.Config{AppEnv: "production", GRPCAllowInsecure: true}, false, false, true},
		{"production TLS", config.Config{AppEnv: "production", GRPCTLSCertFile: cert, GRPCTLSKeyFile: key}, false, true, false},
		{"opt-in retains TLS", config.Config{AppEnv: "production", GRPCAllowInsecure: true, GRPCTLSCertFile: cert, GRPCTLSKeyFile: key}, false, true, false},
		{"partial TLS never downgrades", config.Config{AppEnv: "production", GRPCAllowInsecure: true, GRPCTLSCertFile: cert}, true, false, false},
		{"bad certificate never downgrades", config.Config{AppEnv: "production", GRPCAllowInsecure: true, GRPCTLSCertFile: bad, GRPCTLSKeyFile: key}, true, false, false},
		{"bad key never downgrades", config.Config{AppEnv: "production", GRPCAllowInsecure: true, GRPCTLSCertFile: cert, GRPCTLSKeyFile: bad}, true, false, false},
		{"mismatched key never downgrades", config.Config{AppEnv: "production", GRPCAllowInsecure: true, GRPCTLSCertFile: cert, GRPCTLSKeyFile: otherKey}, true, false, false},
		{"missing file never downgrades", config.Config{AppEnv: "production", GRPCAllowInsecure: true, GRPCTLSCertFile: bad + ".missing", GRPCTLSKeyFile: key}, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			creds, err := ServerCredentials(tc.cfg, slog.New(slog.NewJSONHandler(&output, nil)))
			if (err != nil) != tc.wantError || (creds != nil) != tc.wantTLS {
				t.Fatalf("TLS=%t error=%v", creds != nil, err)
			}
			warned := strings.Contains(output.String(), `"level":"WARN"`)
			if warned != tc.wantWarn || warned && (!strings.Contains(output.String(), "GRPC_ALLOW_INSECURE") || !strings.Contains(output.String(), "unencrypted")) {
				t.Fatalf("incorrect transport warning: %s", output.String())
			}
		})
	}
}

func TestTransportTLSProtectsSessionMetadata(t *testing.T) {
	cert, key, roots := transportCertificate(t)
	creds, err := ServerCredentials(config.Config{AppEnv: "production", GRPCTLSCertFile: cert, GRPCTLSKeyFile: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	transport := grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		calls.Add(1)
		remote, _ := peer.FromContext(ctx)
		if remote == nil {
			t.Error("missing transport peer")
		} else if info, ok := remote.AuthInfo.(credentials.TLSInfo); !ok || info.State.Version < tls.VersionTLS12 {
			t.Error("session metadata reached an unprotected RPC")
		}
		md, _ := metadata.FromIncomingContext(ctx)
		if tokens := md.Get("x-session-token"); len(tokens) != 1 || tokens[0] != "transport-test-token" {
			t.Error("session metadata changed")
		}
		return next(ctx, req)
	}))
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(transport, healthServer)
	go func() { _ = transport.Serve(listener) }()
	t.Cleanup(transport.Stop)
	check := func(t *testing.T, clientCreds credentials.TransportCredentials, success bool) {
		t.Helper()
		conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(clientCreds))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		ctx = metadata.AppendToOutgoingContext(ctx, "x-session-token", "transport-test-token")
		_, err = grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if (err == nil) != success {
			t.Fatalf("RPC success=%t error=%v", success, err)
		}
	}
	check(t, credentials.NewTLS(&tls.Config{RootCAs: roots, ServerName: "core.test", MinVersion: tls.VersionTLS12}), true)
	check(t, insecure.NewCredentials(), false)
	check(t, credentials.NewTLS(&tls.Config{RootCAs: roots, ServerName: "wrong.test", MinVersion: tls.VersionTLS12}), false)
	if calls.Load() != 1 {
		t.Fatalf("rejected handshakes reached RPC handlers: calls=%d", calls.Load())
	}
}
