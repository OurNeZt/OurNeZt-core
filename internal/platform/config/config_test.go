package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("GRPC_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("BOOTSTRAP_ADMIN_EMAIL", "")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "")
	t.Setenv("BOOTSTRAP_ADMIN_DISPLAY_NAME", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("SESSION_TOKEN_BYTES", "")
	t.Setenv("PASSWORD_MEMORY_KB", "")
	t.Setenv("PASSWORD_ITERATIONS", "")
	t.Setenv("PASSWORD_PARALLELISM", "")

	got := Load()

	if got.AppEnv != "development" {
		t.Fatalf("AppEnv = %q, want development", got.AppEnv)
	}
	if got.GRPCAddr != ":50051" {
		t.Fatalf("GRPCAddr = %q, want :50051", got.GRPCAddr)
	}
	if got.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", got.DatabaseURL)
	}
	if got.BootstrapAdminEmail != "" {
		t.Fatalf("BootstrapAdminEmail = %q, want empty", got.BootstrapAdminEmail)
	}
	if got.BootstrapAdminPassword != "" {
		t.Fatalf("BootstrapAdminPassword = %q, want empty", got.BootstrapAdminPassword)
	}
	if got.BootstrapAdminDisplayName != "Bootstrap Admin" {
		t.Fatalf("BootstrapAdminDisplayName = %q, want Bootstrap Admin", got.BootstrapAdminDisplayName)
	}
	if got.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 10s", got.ShutdownTimeout)
	}
	if got.SessionTokenBytes != 32 {
		t.Fatalf("SessionTokenBytes = %d, want 32", got.SessionTokenBytes)
	}
	if got.LogLevel() != slog.LevelDebug {
		t.Fatalf("LogLevel = %s, want debug", got.LogLevel())
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("GRPC_ADDR", ":7000")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("BOOTSTRAP_ADMIN_EMAIL", "admin@ournezt.local")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "TempPass123!")
	t.Setenv("BOOTSTRAP_ADMIN_DISPLAY_NAME", "Owner Admin")
	t.Setenv("SHUTDOWN_TIMEOUT", "15s")
	t.Setenv("SESSION_TOKEN_BYTES", "48")
	t.Setenv("PASSWORD_MEMORY_KB", "8192")
	t.Setenv("PASSWORD_ITERATIONS", "4")
	t.Setenv("PASSWORD_PARALLELISM", "1")

	got := Load()

	if got.AppEnv != "production" {
		t.Fatalf("AppEnv = %q, want production", got.AppEnv)
	}
	if got.GRPCAddr != ":7000" {
		t.Fatalf("GRPCAddr = %q, want :7000", got.GRPCAddr)
	}
	if got.DatabaseURL != "postgres://example" {
		t.Fatalf("DatabaseURL = %q, want postgres://example", got.DatabaseURL)
	}
	if got.BootstrapAdminEmail != "admin@ournezt.local" {
		t.Fatalf("BootstrapAdminEmail = %q, want admin@ournezt.local", got.BootstrapAdminEmail)
	}
	if got.BootstrapAdminPassword != "TempPass123!" {
		t.Fatalf("BootstrapAdminPassword = %q, want TempPass123!", got.BootstrapAdminPassword)
	}
	if got.BootstrapAdminDisplayName != "Owner Admin" {
		t.Fatalf("BootstrapAdminDisplayName = %q, want Owner Admin", got.BootstrapAdminDisplayName)
	}
	if got.ShutdownTimeout != 15*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 15s", got.ShutdownTimeout)
	}
	if got.SessionTokenBytes != 48 {
		t.Fatalf("SessionTokenBytes = %d, want 48", got.SessionTokenBytes)
	}
	if got.PasswordMemoryKB != 8192 {
		t.Fatalf("PasswordMemoryKB = %d, want 8192", got.PasswordMemoryKB)
	}
	if got.PasswordIterations != 4 {
		t.Fatalf("PasswordIterations = %d, want 4", got.PasswordIterations)
	}
	if got.PasswordParallelism != 1 {
		t.Fatalf("PasswordParallelism = %d, want 1", got.PasswordParallelism)
	}
	if got.LogLevel() != slog.LevelInfo {
		t.Fatalf("LogLevel = %s, want info", got.LogLevel())
	}
}

func TestLoadFallsBackForInvalidNumbers(t *testing.T) {
	t.Setenv("SHUTDOWN_TIMEOUT", "not-a-duration")
	t.Setenv("SESSION_TOKEN_BYTES", "many")

	got := Load()

	if got.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 10s", got.ShutdownTimeout)
	}
	if got.SessionTokenBytes != 32 {
		t.Fatalf("SessionTokenBytes = %d, want 32", got.SessionTokenBytes)
	}
}
