package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/config"
	"github.com/OurNeZt/ournezt-core/internal/platform/database"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/repository/postgres"
	"github.com/OurNeZt/ournezt-core/internal/server"
	"github.com/OurNeZt/ournezt-core/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel(),
	}))

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Error("listen grpc", "addr", cfg.GRPCAddr, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	userRepo := postgres.NewUserRepository(pool)
	familyRepo := postgres.NewFamilyRepository(pool)
	personRepo := postgres.NewPersonRepository(pool)
	housingRepo := postgres.NewHousingRepository(pool)

	authService := service.NewAuthService(userRepo, security.Argon2Params{
		MemoryKB:    cfg.PasswordMemoryKB,
		Iterations:  cfg.PasswordIterations,
		Parallelism: cfg.PasswordParallelism,
		SaltLength:  16,
		KeyLength:   32,
	})

	authServer := server.NewAuthServer(authService, userRepo, cfg.SessionTokenBytes, 24*time.Hour, nil)
	familyServer := server.NewFamilyServer(familyRepo, 7*24*time.Hour, nil, authServer)
	personServer := server.NewPersonServer(personRepo, authServer)
	housingServer := server.NewHousingServer(housingRepo, authServer)
	financeServer := server.NewFinanceServer()
	dashboardServer := server.NewDashboardServer(personRepo, housingRepo, authServer)

	ourneztv1.RegisterAuthServiceServer(grpcServer, authServer)
	ourneztv1.RegisterFamilyServiceServer(grpcServer, familyServer)
	ourneztv1.RegisterPersonServiceServer(grpcServer, personServer)
	ourneztv1.RegisterHousingServiceServer(grpcServer, housingServer)
	ourneztv1.RegisterIncomeServiceServer(grpcServer, financeServer)
	ourneztv1.RegisterCPFServiceServer(grpcServer, financeServer)
	ourneztv1.RegisterDashboardServiceServer(grpcServer, dashboardServer)

	go func() {
		logger.Info("ournezt core started", "grpc_addr", cfg.GRPCAddr, "env", cfg.AppEnv)
		if serveErr := grpcServer.Serve(listener); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			logger.Error("grpc server stopped unexpectedly", "error", serveErr)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("ournezt core shutting down")
	grpcServer.GracefulStop()
}
