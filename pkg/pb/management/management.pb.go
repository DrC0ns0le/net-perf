// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.0
// 	protoc        v3.12.4
// source: proto/management/management.proto

package management

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Route struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Address       string                 `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Next          string                 `protobuf:"bytes,2,opt,name=next,proto3" json:"next,omitempty"`
	Interface     string                 `protobuf:"bytes,3,opt,name=interface,proto3" json:"interface,omitempty"`
	Metric        int32                  `protobuf:"varint,4,opt,name=metric,proto3" json:"metric,omitempty"`
	Protocol      int32                  `protobuf:"varint,5,opt,name=protocol,proto3" json:"protocol,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Route) Reset() {
	*x = Route{}
	mi := &file_proto_management_management_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Route) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Route) ProtoMessage() {}

func (x *Route) ProtoReflect() protoreflect.Message {
	mi := &file_proto_management_management_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Route.ProtoReflect.Descriptor instead.
func (*Route) Descriptor() ([]byte, []int) {
	return file_proto_management_management_proto_rawDescGZIP(), []int{0}
}

func (x *Route) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *Route) GetNext() string {
	if x != nil {
		return x.Next
	}
	return ""
}

func (x *Route) GetInterface() string {
	if x != nil {
		return x.Interface
	}
	return ""
}

func (x *Route) GetMetric() int32 {
	if x != nil {
		return x.Metric
	}
	return 0
}

func (x *Route) GetProtocol() int32 {
	if x != nil {
		return x.Protocol
	}
	return 0
}

type RouteTable struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Routes        []*Route               `protobuf:"bytes,1,rep,name=routes,proto3" json:"routes,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RouteTable) Reset() {
	*x = RouteTable{}
	mi := &file_proto_management_management_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RouteTable) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RouteTable) ProtoMessage() {}

func (x *RouteTable) ProtoReflect() protoreflect.Message {
	mi := &file_proto_management_management_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RouteTable.ProtoReflect.Descriptor instead.
func (*RouteTable) Descriptor() ([]byte, []int) {
	return file_proto_management_management_proto_rawDescGZIP(), []int{1}
}

func (x *RouteTable) GetRoutes() []*Route {
	if x != nil {
		return x.Routes
	}
	return nil
}

type GetRouteTableRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetRouteTableRequest) Reset() {
	*x = GetRouteTableRequest{}
	mi := &file_proto_management_management_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetRouteTableRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRouteTableRequest) ProtoMessage() {}

func (x *GetRouteTableRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_management_management_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRouteTableRequest.ProtoReflect.Descriptor instead.
func (*GetRouteTableRequest) Descriptor() ([]byte, []int) {
	return file_proto_management_management_proto_rawDescGZIP(), []int{2}
}

type GetRouteTableResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Routes        *RouteTable            `protobuf:"bytes,1,opt,name=routes,proto3" json:"routes,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetRouteTableResponse) Reset() {
	*x = GetRouteTableResponse{}
	mi := &file_proto_management_management_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetRouteTableResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRouteTableResponse) ProtoMessage() {}

func (x *GetRouteTableResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_management_management_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRouteTableResponse.ProtoReflect.Descriptor instead.
func (*GetRouteTableResponse) Descriptor() ([]byte, []int) {
	return file_proto_management_management_proto_rawDescGZIP(), []int{3}
}

func (x *GetRouteTableResponse) GetRoutes() *RouteTable {
	if x != nil {
		return x.Routes
	}
	return nil
}

type GetRouteTableMapResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Routes        map[int32]int32        `protobuf:"bytes,1,rep,name=routes,proto3" json:"routes,omitempty" protobuf_key:"varint,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetRouteTableMapResponse) Reset() {
	*x = GetRouteTableMapResponse{}
	mi := &file_proto_management_management_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetRouteTableMapResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRouteTableMapResponse) ProtoMessage() {}

func (x *GetRouteTableMapResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_management_management_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRouteTableMapResponse.ProtoReflect.Descriptor instead.
func (*GetRouteTableMapResponse) Descriptor() ([]byte, []int) {
	return file_proto_management_management_proto_rawDescGZIP(), []int{4}
}

func (x *GetRouteTableMapResponse) GetRoutes() map[int32]int32 {
	if x != nil {
		return x.Routes
	}
	return nil
}

type GetStateRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           string                 `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Namespace     string                 `protobuf:"bytes,2,opt,name=namespace,proto3" json:"namespace,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetStateRequest) Reset() {
	*x = GetStateRequest{}
	mi := &file_proto_management_management_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetStateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetStateRequest) ProtoMessage() {}

func (x *GetStateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_management_management_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetStateRequest.ProtoReflect.Descriptor instead.
func (*GetStateRequest) Descriptor() ([]byte, []int) {
	return file_proto_management_management_proto_rawDescGZIP(), []int{5}
}

func (x *GetStateRequest) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *GetStateRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

type GetStateResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         string                 `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp     int64                  `protobuf:"varint,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetStateResponse) Reset() {
	*x = GetStateResponse{}
	mi := &file_proto_management_management_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetStateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetStateResponse) ProtoMessage() {}

func (x *GetStateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_management_management_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetStateResponse.ProtoReflect.Descriptor instead.
func (*GetStateResponse) Descriptor() ([]byte, []int) {
	return file_proto_management_management_proto_rawDescGZIP(), []int{6}
}

func (x *GetStateResponse) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *GetStateResponse) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

type GetConsensusStateRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetConsensusStateRequest) Reset() {
	*x = GetConsensusStateRequest{}
	mi := &file_proto_management_management_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetConsensusStateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetConsensusStateRequest) ProtoMessage() {}

func (x *GetConsensusStateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_management_management_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetConsensusStateRequest.ProtoReflect.Descriptor instead.
func (*GetConsensusStateRequest) Descriptor() ([]byte, []int) {
	return file_proto_management_management_proto_rawDescGZIP(), []int{7}
}

type GetConsensusStateResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	State         string                 `protobuf:"bytes,1,opt,name=state,proto3" json:"state,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetConsensusStateResponse) Reset() {
	*x = GetConsensusStateResponse{}
	mi := &file_proto_management_management_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetConsensusStateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetConsensusStateResponse) ProtoMessage() {}

func (x *GetConsensusStateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_management_management_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetConsensusStateResponse.ProtoReflect.Descriptor instead.
func (*GetConsensusStateResponse) Descriptor() ([]byte, []int) {
	return file_proto_management_management_proto_rawDescGZIP(), []int{8}
}

func (x *GetConsensusStateResponse) GetState() string {
	if x != nil {
		return x.State
	}
	return ""
}

var File_proto_management_management_proto protoreflect.FileDescriptor

var file_proto_management_management_proto_rawDesc = []byte{
	0x0a, 0x21, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65,
	0x6e, 0x74, 0x2f, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x22,
	0x87, 0x01, 0x0a, 0x05, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72,
	0x65, 0x73, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x65, 0x78, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x6e, 0x65, 0x78, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x66, 0x61, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x66, 0x61, 0x63, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x12, 0x1a, 0x0a,
	0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x22, 0x37, 0x0a, 0x0a, 0x52, 0x6f, 0x75,
	0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x29, 0x0a, 0x06, 0x72, 0x6f, 0x75, 0x74, 0x65,
	0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65,
	0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x52, 0x06, 0x72, 0x6f, 0x75, 0x74,
	0x65, 0x73, 0x22, 0x16, 0x0a, 0x14, 0x47, 0x65, 0x74, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61,
	0x62, 0x6c, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x47, 0x0a, 0x15, 0x47, 0x65,
	0x74, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x2e, 0x0a, 0x06, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x73, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74,
	0x2e, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x52, 0x06, 0x72, 0x6f, 0x75,
	0x74, 0x65, 0x73, 0x22, 0x9f, 0x01, 0x0a, 0x18, 0x47, 0x65, 0x74, 0x52, 0x6f, 0x75, 0x74, 0x65,
	0x54, 0x61, 0x62, 0x6c, 0x65, 0x4d, 0x61, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x48, 0x0a, 0x06, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x30, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x47, 0x65,
	0x74, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x4d, 0x61, 0x70, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x52, 0x06, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x73, 0x1a, 0x39, 0x0a, 0x0b, 0x52, 0x6f,
	0x75, 0x74, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x41, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74,
	0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61,
	0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e,
	0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x22, 0x46, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x53,
	0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x22, 0x1a, 0x0a, 0x18, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73, 0x75, 0x73,
	0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x31, 0x0a, 0x19,
	0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73, 0x75, 0x73, 0x53, 0x74, 0x61, 0x74,
	0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x74, 0x61,
	0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x32,
	0xd7, 0x03, 0x0a, 0x0a, 0x4d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x56,
	0x0a, 0x0d, 0x47, 0x65, 0x74, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x12,
	0x20, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x47, 0x65, 0x74,
	0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x21, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x47,
	0x65, 0x74, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x64, 0x0a, 0x18, 0x47, 0x65, 0x74, 0x43, 0x65, 0x6e,
	0x74, 0x72, 0x61, 0x6c, 0x69, 0x73, 0x65, 0x64, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62,
	0x6c, 0x65, 0x12, 0x20, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e,
	0x47, 0x65, 0x74, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x24, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e,
	0x74, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x4d,
	0x61, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x5e, 0x0a, 0x12,
	0x47, 0x65, 0x74, 0x47, 0x72, 0x61, 0x70, 0x68, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62,
	0x6c, 0x65, 0x12, 0x20, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e,
	0x47, 0x65, 0x74, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x24, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e,
	0x74, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x4d,
	0x61, 0x70, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x47, 0x0a, 0x08,
	0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x1b, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67,
	0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1c, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65,
	0x6e, 0x74, 0x2e, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x62, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x73,
	0x65, 0x6e, 0x73, 0x75, 0x73, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x24, 0x2e, 0x6d, 0x61, 0x6e,
	0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x73, 0x65,
	0x6e, 0x73, 0x75, 0x73, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x25, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x47, 0x65,
	0x74, 0x43, 0x6f, 0x6e, 0x73, 0x65, 0x6e, 0x73, 0x75, 0x73, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x3c, 0x5a, 0x3a, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x44, 0x72, 0x43, 0x30, 0x6e, 0x73, 0x30, 0x6c,
	0x65, 0x2f, 0x6e, 0x65, 0x74, 0x2d, 0x70, 0x65, 0x72, 0x66, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x70,
	0x62, 0x2f, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x3b, 0x6d, 0x61, 0x6e,
	0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_management_management_proto_rawDescOnce sync.Once
	file_proto_management_management_proto_rawDescData = file_proto_management_management_proto_rawDesc
)

func file_proto_management_management_proto_rawDescGZIP() []byte {
	file_proto_management_management_proto_rawDescOnce.Do(func() {
		file_proto_management_management_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_management_management_proto_rawDescData)
	})
	return file_proto_management_management_proto_rawDescData
}

var file_proto_management_management_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_proto_management_management_proto_goTypes = []any{
	(*Route)(nil),                     // 0: management.Route
	(*RouteTable)(nil),                // 1: management.RouteTable
	(*GetRouteTableRequest)(nil),      // 2: management.GetRouteTableRequest
	(*GetRouteTableResponse)(nil),     // 3: management.GetRouteTableResponse
	(*GetRouteTableMapResponse)(nil),  // 4: management.GetRouteTableMapResponse
	(*GetStateRequest)(nil),           // 5: management.GetStateRequest
	(*GetStateResponse)(nil),          // 6: management.GetStateResponse
	(*GetConsensusStateRequest)(nil),  // 7: management.GetConsensusStateRequest
	(*GetConsensusStateResponse)(nil), // 8: management.GetConsensusStateResponse
	nil,                               // 9: management.GetRouteTableMapResponse.RoutesEntry
}
var file_proto_management_management_proto_depIdxs = []int32{
	0, // 0: management.RouteTable.routes:type_name -> management.Route
	1, // 1: management.GetRouteTableResponse.routes:type_name -> management.RouteTable
	9, // 2: management.GetRouteTableMapResponse.routes:type_name -> management.GetRouteTableMapResponse.RoutesEntry
	2, // 3: management.Management.GetRouteTable:input_type -> management.GetRouteTableRequest
	2, // 4: management.Management.GetCentralisedRouteTable:input_type -> management.GetRouteTableRequest
	2, // 5: management.Management.GetGraphRouteTable:input_type -> management.GetRouteTableRequest
	5, // 6: management.Management.GetState:input_type -> management.GetStateRequest
	7, // 7: management.Management.GetConsensusState:input_type -> management.GetConsensusStateRequest
	3, // 8: management.Management.GetRouteTable:output_type -> management.GetRouteTableResponse
	4, // 9: management.Management.GetCentralisedRouteTable:output_type -> management.GetRouteTableMapResponse
	4, // 10: management.Management.GetGraphRouteTable:output_type -> management.GetRouteTableMapResponse
	6, // 11: management.Management.GetState:output_type -> management.GetStateResponse
	8, // 12: management.Management.GetConsensusState:output_type -> management.GetConsensusStateResponse
	8, // [8:13] is the sub-list for method output_type
	3, // [3:8] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_proto_management_management_proto_init() }
func file_proto_management_management_proto_init() {
	if File_proto_management_management_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_management_management_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_management_management_proto_goTypes,
		DependencyIndexes: file_proto_management_management_proto_depIdxs,
		MessageInfos:      file_proto_management_management_proto_msgTypes,
	}.Build()
	File_proto_management_management_proto = out.File
	file_proto_management_management_proto_rawDesc = nil
	file_proto_management_management_proto_goTypes = nil
	file_proto_management_management_proto_depIdxs = nil
}
