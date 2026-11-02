package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MayurJ1609/common-lib/config"
	"github.com/MayurJ1609/common-lib/db"
	"github.com/MayurJ1609/common-lib/logger"
	"github.com/example/go-boilerplate/internal/domain"
	greetergrpc "github.com/example/go-boilerplate/internal/handler/grpc"
	"google.golang.org/grpc"
)

func main() {
	log := logger.New(logger.Options{Service: "go-boilerplate", Level: envOrDefault("LOG_LEVEL", "info")})
	if err := run(log); err != nil {
		log.Error("gRPC server stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) (result error) {
	cfg, err := config.LoadDependencyConfig()
	if err != nil {
		return fmt.Errorf("load dependency configuration: %w", err)
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 30*time.Second)
	clients, err := db.Connect(connectCtx, cfg)
	connectCancel()
	if err != nil {
		return fmt.Errorf("connect dependencies: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := clients.Close(closeCtx); err != nil {
			result = errors.Join(result, fmt.Errorf("close dependencies: %w", err))
		}
	}()

	addr := envOrDefault("GRPC_ADDR", ":9090")
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	grpcServer := grpc.NewServer()
	greetergrpc.RegisterGreeterServer(grpcServer, greetergrpc.NewGreeterService(domain.Greeter{}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	log.Info("gRPC server listening", "addr", addr)
	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("gRPC server: %w", err)
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
