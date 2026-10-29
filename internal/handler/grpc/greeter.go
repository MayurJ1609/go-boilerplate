package grpc

import (
	"context"

	"github.com/example/go-boilerplate/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const greeterFullMethodName = "/greeter.v1.Greeter/SayHello"

type GreeterServer interface {
	SayHello(context.Context, *wrapperspb.StringValue) (*wrapperspb.StringValue, error)
}

type GreeterClient interface {
	SayHello(ctx context.Context, in *wrapperspb.StringValue, opts ...grpc.CallOption) (*wrapperspb.StringValue, error)
}

type greeterClient struct {
	cc grpc.ClientConnInterface
}

func NewGreeterClient(cc grpc.ClientConnInterface) GreeterClient {
	return &greeterClient{cc: cc}
}

func (c *greeterClient) SayHello(ctx context.Context, in *wrapperspb.StringValue, opts ...grpc.CallOption) (*wrapperspb.StringValue, error) {
	out := new(wrapperspb.StringValue)
	if err := c.cc.Invoke(ctx, greeterFullMethodName, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

type server struct {
	GreeterServer
}

func RegisterGreeterServer(registrar grpc.ServiceRegistrar, implementation GreeterServer) {
	registrar.RegisterService(&greeterServiceDesc, &server{GreeterServer: implementation})
}

func (s *server) SayHello(ctx context.Context, in *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return s.GreeterServer.SayHello(ctx, in)
}

func sayHelloHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(wrapperspb.StringValue)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GreeterServer).SayHello(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: greeterFullMethodName}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(GreeterServer).SayHello(ctx, req.(*wrapperspb.StringValue))
	}
	return interceptor(ctx, in, info, handler)
}

var greeterServiceDesc = grpc.ServiceDesc{
	ServiceName: "greeter.v1.Greeter",
	HandlerType: (*GreeterServer)(nil),
	Methods: []grpc.MethodDesc{{
		MethodName: "SayHello",
		Handler:    sayHelloHandler,
	}},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/proto/greeter.proto",
}

type GreeterService struct {
	greeter domain.Greeter
}

func NewGreeterService(greeter domain.Greeter) GreeterServer {
	return &GreeterService{greeter: greeter}
}

func (s *GreeterService) SayHello(_ context.Context, in *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	return wrapperspb.String(s.greeter.Greet(in.Value)), nil
}
