package main

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"testing"

	"github.com/example/go-boilerplate/internal/domain"
	greetergrpc "github.com/example/go-boilerplate/internal/handler/grpc"
	"google.golang.org/grpc"
)

func TestRunCallsGreeterServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	greetergrpc.RegisterGreeterServer(server, greetergrpc.NewGreeterService(domain.Greeter{}))
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve gRPC: %v", err)
		}
	}()
	t.Cleanup(server.Stop)
	t.Setenv("GRPC_TARGET", listener.Addr().String())

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := run(log); err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

func TestRunReturnsErrorWhenServerIsUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve address: %v", err)
	}
	target := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	t.Setenv("GRPC_TARGET", target)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := run(log); err == nil {
		t.Fatal("run() error = nil, want connection error")
	}
}
