package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-boilerplate/internal/config"
	"github.com/example/go-boilerplate/internal/database"
	"github.com/example/go-boilerplate/internal/domain"
	greetergrpc "github.com/example/go-boilerplate/internal/handler/grpc"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load dependency configuration: %v", err)
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 30*time.Second)
	clients, err := database.Connect(connectCtx, cfg)
	connectCancel()
	if err != nil {
		log.Fatalf("connect dependencies: %v", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := clients.Close(closeCtx); err != nil {
			log.Printf("close dependencies: %v", err)
		}
	}()

	addr := envOrDefault("GRPC_ADDR", ":9090")
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen on %s: %v", addr, err)
	}

	server := grpc.NewServer()
	greetergrpc.RegisterGreeterServer(server, greetergrpc.NewGreeterService(domain.Greeter{}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	log.Printf("gRPC server listening on %s", addr)
	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		log.Fatalf("gRPC server: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
