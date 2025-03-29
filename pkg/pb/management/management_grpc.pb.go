// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.12.4
// source: proto/management/management.proto

package management

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
	Management_GetRouteTable_FullMethodName            = "/management.Management/GetRouteTable"
	Management_GetCentralisedRouteTable_FullMethodName = "/management.Management/GetCentralisedRouteTable"
	Management_GetGraphRouteTable_FullMethodName       = "/management.Management/GetGraphRouteTable"
	Management_GetState_FullMethodName                 = "/management.Management/GetState"
	Management_GetConsensusState_FullMethodName        = "/management.Management/GetConsensusState"
)

// ManagementClient is the client API for Management service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ManagementClient interface {
	GetRouteTable(ctx context.Context, in *GetRouteTableRequest, opts ...grpc.CallOption) (*GetRouteTableResponse, error)
	GetCentralisedRouteTable(ctx context.Context, in *GetRouteTableRequest, opts ...grpc.CallOption) (*GetRouteTableMapResponse, error)
	GetGraphRouteTable(ctx context.Context, in *GetRouteTableRequest, opts ...grpc.CallOption) (*GetRouteTableMapResponse, error)
	GetState(ctx context.Context, in *GetStateRequest, opts ...grpc.CallOption) (*GetStateResponse, error)
	GetConsensusState(ctx context.Context, in *GetConsensusStateRequest, opts ...grpc.CallOption) (*GetConsensusStateResponse, error)
}

type managementClient struct {
	cc grpc.ClientConnInterface
}

func NewManagementClient(cc grpc.ClientConnInterface) ManagementClient {
	return &managementClient{cc}
}

func (c *managementClient) GetRouteTable(ctx context.Context, in *GetRouteTableRequest, opts ...grpc.CallOption) (*GetRouteTableResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetRouteTableResponse)
	err := c.cc.Invoke(ctx, Management_GetRouteTable_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *managementClient) GetCentralisedRouteTable(ctx context.Context, in *GetRouteTableRequest, opts ...grpc.CallOption) (*GetRouteTableMapResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetRouteTableMapResponse)
	err := c.cc.Invoke(ctx, Management_GetCentralisedRouteTable_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *managementClient) GetGraphRouteTable(ctx context.Context, in *GetRouteTableRequest, opts ...grpc.CallOption) (*GetRouteTableMapResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetRouteTableMapResponse)
	err := c.cc.Invoke(ctx, Management_GetGraphRouteTable_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *managementClient) GetState(ctx context.Context, in *GetStateRequest, opts ...grpc.CallOption) (*GetStateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetStateResponse)
	err := c.cc.Invoke(ctx, Management_GetState_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *managementClient) GetConsensusState(ctx context.Context, in *GetConsensusStateRequest, opts ...grpc.CallOption) (*GetConsensusStateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetConsensusStateResponse)
	err := c.cc.Invoke(ctx, Management_GetConsensusState_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ManagementServer is the server API for Management service.
// All implementations must embed UnimplementedManagementServer
// for forward compatibility.
type ManagementServer interface {
	GetRouteTable(context.Context, *GetRouteTableRequest) (*GetRouteTableResponse, error)
	GetCentralisedRouteTable(context.Context, *GetRouteTableRequest) (*GetRouteTableMapResponse, error)
	GetGraphRouteTable(context.Context, *GetRouteTableRequest) (*GetRouteTableMapResponse, error)
	GetState(context.Context, *GetStateRequest) (*GetStateResponse, error)
	GetConsensusState(context.Context, *GetConsensusStateRequest) (*GetConsensusStateResponse, error)
	mustEmbedUnimplementedManagementServer()
}

// UnimplementedManagementServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedManagementServer struct{}

func (UnimplementedManagementServer) GetRouteTable(context.Context, *GetRouteTableRequest) (*GetRouteTableResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRouteTable not implemented")
}
func (UnimplementedManagementServer) GetCentralisedRouteTable(context.Context, *GetRouteTableRequest) (*GetRouteTableMapResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCentralisedRouteTable not implemented")
}
func (UnimplementedManagementServer) GetGraphRouteTable(context.Context, *GetRouteTableRequest) (*GetRouteTableMapResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetGraphRouteTable not implemented")
}
func (UnimplementedManagementServer) GetState(context.Context, *GetStateRequest) (*GetStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetState not implemented")
}
func (UnimplementedManagementServer) GetConsensusState(context.Context, *GetConsensusStateRequest) (*GetConsensusStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConsensusState not implemented")
}
func (UnimplementedManagementServer) mustEmbedUnimplementedManagementServer() {}
func (UnimplementedManagementServer) testEmbeddedByValue()                    {}

// UnsafeManagementServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ManagementServer will
// result in compilation errors.
type UnsafeManagementServer interface {
	mustEmbedUnimplementedManagementServer()
}

func RegisterManagementServer(s grpc.ServiceRegistrar, srv ManagementServer) {
	// If the following call pancis, it indicates UnimplementedManagementServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Management_ServiceDesc, srv)
}

func _Management_GetRouteTable_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRouteTableRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ManagementServer).GetRouteTable(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Management_GetRouteTable_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ManagementServer).GetRouteTable(ctx, req.(*GetRouteTableRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Management_GetCentralisedRouteTable_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRouteTableRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ManagementServer).GetCentralisedRouteTable(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Management_GetCentralisedRouteTable_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ManagementServer).GetCentralisedRouteTable(ctx, req.(*GetRouteTableRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Management_GetGraphRouteTable_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRouteTableRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ManagementServer).GetGraphRouteTable(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Management_GetGraphRouteTable_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ManagementServer).GetGraphRouteTable(ctx, req.(*GetRouteTableRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Management_GetState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ManagementServer).GetState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Management_GetState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ManagementServer).GetState(ctx, req.(*GetStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Management_GetConsensusState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetConsensusStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ManagementServer).GetConsensusState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Management_GetConsensusState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ManagementServer).GetConsensusState(ctx, req.(*GetConsensusStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Management_ServiceDesc is the grpc.ServiceDesc for Management service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Management_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "management.Management",
	HandlerType: (*ManagementServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetRouteTable",
			Handler:    _Management_GetRouteTable_Handler,
		},
		{
			MethodName: "GetCentralisedRouteTable",
			Handler:    _Management_GetCentralisedRouteTable_Handler,
		},
		{
			MethodName: "GetGraphRouteTable",
			Handler:    _Management_GetGraphRouteTable_Handler,
		},
		{
			MethodName: "GetState",
			Handler:    _Management_GetState_Handler,
		},
		{
			MethodName: "GetConsensusState",
			Handler:    _Management_GetConsensusState_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/management/management.proto",
}
