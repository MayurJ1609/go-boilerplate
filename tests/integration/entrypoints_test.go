//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/go-boilerplate/internal/service"
	"github.com/example/go-boilerplate/internal/transport/grpcapi"
	"github.com/example/go-boilerplate/internal/transport/rest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRESTIntegration(t *testing.T) {
	server := httptest.NewServer(rest.NewHandler(service.Greeter{}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/greet?name=Integration")
	if err != nil {
		t.Fatalf("GET greeting: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode greeting: %v", err)
	}
	if got, want := body["message"], "Hello, Integration!"; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestGRPCIntegration(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	grpcapi.RegisterGreeterServer(server, grpcapi.NewGreeterService(service.Greeter{}))
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve gRPC: %v", err)
		}
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatalf("create gRPC client: %v", err)
	}
	defer conn.Close()

	response, err := grpcapi.NewGreeterClient(conn).SayHello(ctx, wrapperspb.String("Integration"))
	if err != nil {
		t.Fatalf("SayHello: %v", err)
	}
	if got, want := response.Value, "Hello, Integration!"; got != want {
		t.Fatalf("response = %q, want %q", got, want)
	}
}
