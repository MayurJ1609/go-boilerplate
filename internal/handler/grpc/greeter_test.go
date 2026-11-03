package grpc

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/example/go-boilerplate/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestGreeterClientAndServer(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	RegisterGreeterServer(server, NewGreeterService(domain.Greeter{}))
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

	response, err := NewGreeterClient(conn).SayHello(ctx, wrapperspb.String("Ada"))
	if err != nil {
		t.Fatalf("SayHello() error = %v", err)
	}
	if got, want := response.Value, "Hello, Ada!"; got != want {
		t.Fatalf("response = %q, want %q", got, want)
	}
}

func TestGreeterServiceRejectsNilRequest(t *testing.T) {
	_, err := NewGreeterService(domain.Greeter{}).SayHello(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
}

func TestSayHelloHandlerDecodeFailure(t *testing.T) {
	_, err := sayHelloHandler(
		NewGreeterService(domain.Greeter{}),
		context.Background(),
		func(any) error { return errors.New("invalid request") },
		nil,
	)
	if err == nil || err.Error() != "invalid request" {
		t.Fatalf("handler error = %v, want decode error", err)
	}
}

func TestSayHelloHandlerUsesInterceptor(t *testing.T) {
	interceptorCalled := false
	response, err := sayHelloHandler(
		NewGreeterService(domain.Greeter{}),
		context.Background(),
		func(value any) error {
			value.(*wrapperspb.StringValue).Value = "Ada"
			return nil
		},
		func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			interceptorCalled = true
			if info.FullMethod != greeterFullMethodName {
				t.Errorf("full method = %q, want %q", info.FullMethod, greeterFullMethodName)
			}
			return handler(ctx, req)
		},
	)
	if err != nil {
		t.Fatalf("sayHelloHandler() error = %v", err)
	}
	if !interceptorCalled {
		t.Fatal("unary interceptor was not called")
	}
	if got, want := response.(*wrapperspb.StringValue).Value, "Hello, Ada!"; got != want {
		t.Fatalf("response = %q, want %q", got, want)
	}
}

type testClientConn struct {
	err error
}

func (c testClientConn) Invoke(_ context.Context, _ string, _, reply any, _ ...grpc.CallOption) error {
	if c.err != nil {
		return c.err
	}
	reply.(*wrapperspb.StringValue).Value = "Hello, test!"
	return nil
}

func (testClientConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	panic("NewStream should not be called by unary RPC")
}

func TestGreeterClientInvocationError(t *testing.T) {
	wantErr := errors.New("connection failed")
	_, err := NewGreeterClient(testClientConn{err: wantErr}).SayHello(
		context.Background(),
		wrapperspb.String("test"),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("SayHello() error = %v, want %v", err, wantErr)
	}
}
