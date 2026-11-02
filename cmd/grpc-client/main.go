package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/MayurJ1609/common-lib/logger"
	greetergrpc "github.com/example/go-boilerplate/internal/handler/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {
	log := logger.New(logger.Options{Service: "go-boilerplate", Level: envOrDefault("LOG_LEVEL", "info")})
	if err := run(log); err != nil {
		log.Error("gRPC client failed", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	target := envOrDefault("GRPC_TARGET", "localhost:9090")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", target, err)
	}
	defer conn.Close()

	response, err := greetergrpc.NewGreeterClient(conn).SayHello(ctx, wrapperspb.String("gRPC client"))
	if err != nil {
		return fmt.Errorf("call SayHello: %w", err)
	}
	log.Info("gRPC response", "message", response.Value)
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
