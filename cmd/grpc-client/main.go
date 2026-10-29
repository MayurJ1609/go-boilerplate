package main

import (
	"context"
	"log"
	"os"
	"time"

	greetergrpc "github.com/example/go-boilerplate/internal/handler/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {
	target := envOrDefault("GRPC_TARGET", "localhost:9090")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect to %s: %v", target, err)
	}
	defer conn.Close()

	response, err := greetergrpc.NewGreeterClient(conn).SayHello(ctx, wrapperspb.String("gRPC client"))
	if err != nil {
		log.Fatalf("call SayHello: %v", err)
	}
	log.Print(response.Value)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
