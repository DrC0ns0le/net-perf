// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.12.4
// source: proto/distributed/distributed.proto

package distributed

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	RouteService_GetRoute_FullMethodName    = "/distributed.RouteService/GetRoute"
	RouteService_UpdateRoute_FullMethodName = "/distributed.RouteService/UpdateRoute"
)

// RouteServiceClient is the client API for RouteService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RouteServiceClient interface {
	GetRoute(ctx context.Context, in *GetRouteRequest, opts ...grpc.CallOption) (*SiteRoute, error)
	UpdateRoute(ctx context.Context, in *SiteRoute, opts ...grpc.CallOption) (*UpdateRouteResponse, error)
}

type routeServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRouteServiceClient(cc grpc.ClientConnInterface) RouteServiceClient {
	return &routeServiceClient{cc}
}

func (c *routeServiceClient) GetRoute(ctx context.Context, in *GetRouteRequest, opts ...grpc.CallOption) (*SiteRoute, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SiteRoute)
	err := c.cc.Invoke(ctx, RouteService_GetRoute_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *routeServiceClient) UpdateRoute(ctx context.Context, in *SiteRoute, opts ...grpc.CallOption) (*UpdateRouteResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdateRouteResponse)
	err := c.cc.Invoke(ctx, RouteService_UpdateRoute_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RouteServiceServer is the server API for RouteService service.
// All implementations must embed UnimplementedRouteServiceServer
// for forward compatibility.
type RouteServiceServer interface {
	GetRoute(context.Context, *GetRouteRequest) (*SiteRoute, error)
	UpdateRoute(context.Context, *SiteRoute) (*UpdateRouteResponse, error)
	mustEmbedUnimplementedRouteServiceServer()
}

// UnimplementedRouteServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRouteServiceServer struct{}

func (UnimplementedRouteServiceServer) GetRoute(context.Context, *GetRouteRequest) (*SiteRoute, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRoute not implemented")
}
func (UnimplementedRouteServiceServer) UpdateRoute(context.Context, *SiteRoute) (*UpdateRouteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRoute not implemented")
}
func (UnimplementedRouteServiceServer) mustEmbedUnimplementedRouteServiceServer() {}
func (UnimplementedRouteServiceServer) testEmbeddedByValue()                      {}

// UnsafeRouteServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RouteServiceServer will
// result in compilation errors.
type UnsafeRouteServiceServer interface {
	mustEmbedUnimplementedRouteServiceServer()
}

func RegisterRouteServiceServer(s grpc.ServiceRegistrar, srv RouteServiceServer) {
	// If the following call pancis, it indicates UnimplementedRouteServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&RouteService_ServiceDesc, srv)
}

func _RouteService_GetRoute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRouteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RouteServiceServer).GetRoute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RouteService_GetRoute_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RouteServiceServer).GetRoute(ctx, req.(*GetRouteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RouteService_UpdateRoute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SiteRoute)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RouteServiceServer).UpdateRoute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RouteService_UpdateRoute_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RouteServiceServer).UpdateRoute(ctx, req.(*SiteRoute))
	}
	return interceptor(ctx, in, info, handler)
}

// RouteService_ServiceDesc is the grpc.ServiceDesc for RouteService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RouteService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "distributed.RouteService",
	HandlerType: (*RouteServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetRoute",
			Handler:    _RouteService_GetRoute_Handler,
		},
		{
			MethodName: "UpdateRoute",
			Handler:    _RouteService_UpdateRoute_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/distributed/distributed.proto",
}
