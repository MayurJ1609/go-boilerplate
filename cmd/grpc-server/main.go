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

	"github.com/example/go-boilerplate/internal/platform/dependencies"
	"github.com/example/go-boilerplate/internal/service"
	"github.com/example/go-boilerplate/internal/transport/grpcapi"
	"google.golang.org/grpc"
)

func main() {
	config, err := dependencies.LoadConfig()
	if err != nil {
		log.Fatalf("load dependency configuration: %v", err)
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 30*time.Second)
	clients, err := dependencies.Connect(connectCtx, config)
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
	grpcapi.RegisterGreeterServer(server, grpcapi.NewGreeterService(service.Greeter{}))
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
