// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.12.4
// source: proto/measure/measure.proto

package measure

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
	Measure_PathLatency_FullMethodName        = "/measure.Measure/PathLatency"
	Measure_ProjectPathLatency_FullMethodName = "/measure.Measure/ProjectPathLatency"
)

// MeasureClient is the client API for Measure service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MeasureClient interface {
	PathLatency(ctx context.Context, in *PathLatencyRequest, opts ...grpc.CallOption) (*PathLatencyResponse, error)
	ProjectPathLatency(ctx context.Context, in *ProjectedPathLatencyRequest, opts ...grpc.CallOption) (*ProjectedPathLatencyResponse, error)
}

type measureClient struct {
	cc grpc.ClientConnInterface
}

func NewMeasureClient(cc grpc.ClientConnInterface) MeasureClient {
	return &measureClient{cc}
}

func (c *measureClient) PathLatency(ctx context.Context, in *PathLatencyRequest, opts ...grpc.CallOption) (*PathLatencyResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PathLatencyResponse)
	err := c.cc.Invoke(ctx, Measure_PathLatency_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *measureClient) ProjectPathLatency(ctx context.Context, in *ProjectedPathLatencyRequest, opts ...grpc.CallOption) (*ProjectedPathLatencyResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ProjectedPathLatencyResponse)
	err := c.cc.Invoke(ctx, Measure_ProjectPathLatency_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MeasureServer is the server API for Measure service.
// All implementations must embed UnimplementedMeasureServer
// for forward compatibility.
type MeasureServer interface {
	PathLatency(context.Context, *PathLatencyRequest) (*PathLatencyResponse, error)
	ProjectPathLatency(context.Context, *ProjectedPathLatencyRequest) (*ProjectedPathLatencyResponse, error)
	mustEmbedUnimplementedMeasureServer()
}

// UnimplementedMeasureServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedMeasureServer struct{}

func (UnimplementedMeasureServer) PathLatency(context.Context, *PathLatencyRequest) (*PathLatencyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PathLatency not implemented")
}
func (UnimplementedMeasureServer) ProjectPathLatency(context.Context, *ProjectedPathLatencyRequest) (*ProjectedPathLatencyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProjectPathLatency not implemented")
}
func (UnimplementedMeasureServer) mustEmbedUnimplementedMeasureServer() {}
func (UnimplementedMeasureServer) testEmbeddedByValue()                 {}

// UnsafeMeasureServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MeasureServer will
// result in compilation errors.
type UnsafeMeasureServer interface {
	mustEmbedUnimplementedMeasureServer()
}

func RegisterMeasureServer(s grpc.ServiceRegistrar, srv MeasureServer) {
	// If the following call pancis, it indicates UnimplementedMeasureServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Measure_ServiceDesc, srv)
}

func _Measure_PathLatency_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PathLatencyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MeasureServer).PathLatency(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Measure_PathLatency_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MeasureServer).PathLatency(ctx, req.(*PathLatencyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Measure_ProjectPathLatency_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProjectedPathLatencyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MeasureServer).ProjectPathLatency(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Measure_ProjectPathLatency_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MeasureServer).ProjectPathLatency(ctx, req.(*ProjectedPathLatencyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Measure_ServiceDesc is the grpc.ServiceDesc for Measure service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Measure_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "measure.Measure",
	HandlerType: (*MeasureServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PathLatency",
			Handler:    _Measure_PathLatency_Handler,
		},
		{
			MethodName: "ProjectPathLatency",
			Handler:    _Measure_ProjectPathLatency_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/measure/measure.proto",
}
