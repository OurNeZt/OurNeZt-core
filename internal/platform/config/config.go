package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                    string
	GRPCAddr                  string
	DatabaseURL               string
	BootstrapAdminEmail       string
	BootstrapAdminPassword    string
	BootstrapAdminDisplayName string
	ShutdownTimeout           time.Duration
	SessionTokenBytes         int
	PasswordMemoryKB          uint32
	PasswordIterations        uint32
	PasswordParallelism       uint8
}

func Load() Config {
	return Config{
		AppEnv:                    env("APP_ENV", "development"),
		GRPCAddr:                  env("GRPC_ADDR", ":50051"),
		DatabaseURL:               env("DATABASE_URL", ""),
		BootstrapAdminEmail:       env("BOOTSTRAP_ADMIN_EMAIL", ""),
		BootstrapAdminPassword:    env("BOOTSTRAP_ADMIN_PASSWORD", ""),
		BootstrapAdminDisplayName: env("BOOTSTRAP_ADMIN_DISPLAY_NAME", "Bootstrap Admin"),
		ShutdownTimeout:           envDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		SessionTokenBytes:         envInt("SESSION_TOKEN_BYTES", 32),
		PasswordMemoryKB:          uint32(envInt("PASSWORD_MEMORY_KB", 64*1024)),
		PasswordIterations:        uint32(envInt("PASSWORD_ITERATIONS", 3)),
		PasswordParallelism:       uint8(envInt("PASSWORD_PARALLELISM", 2)),
	}
}

func (c Config) LogLevel() slog.Level {
	if strings.EqualFold(c.AppEnv, "development") {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
