//
// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.20.1
// 	protoc        v3.6.1
// source: proto/tunnel/tunnel.proto

// Package grpctunnel defines a service specification for a TCP over gRPC proxy
// interface. This interface is set so that the tunnel can act as a transparent
// TCP proxy between a pair of gRPC client and server endpoints.

package grpctunnel

import (
	context "context"
	reflect "reflect"
	sync "sync"

	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type Data struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tag   int32  `protobuf:"varint,1,opt,name=tag,proto3" json:"tag,omitempty"`     // Tag associated with the initial TCP stream setup.
	Data  []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`    // Bytes received from client connection.
	Close bool   `protobuf:"varint,3,opt,name=close,proto3" json:"close,omitempty"` // Connection has reached EOF.
}

func (x *Data) Reset() {
	*x = Data{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_tunnel_tunnel_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Data) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Data) ProtoMessage() {}

func (x *Data) ProtoReflect() protoreflect.Message {
	mi := &file_proto_tunnel_tunnel_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Data.ProtoReflect.Descriptor instead.
func (*Data) Descriptor() ([]byte, []int) {
	return file_proto_tunnel_tunnel_proto_rawDescGZIP(), []int{0}
}

func (x *Data) GetTag() int32 {
	if x != nil {
		return x.Tag
	}
	return 0
}

func (x *Data) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *Data) GetClose() bool {
	if x != nil {
		return x.Close
	}
	return false
}

type Session struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The tag associated with the initial TCP stream setup.
	Tag int32 `protobuf:"varint,1,opt,name=tag,proto3" json:"tag,omitempty"`
	// Used to ack the connection tag.
	Accept bool `protobuf:"varint,2,opt,name=accept,proto3" json:"accept,omitempty"`
	// Capabilities of the tunnel client and server.
	Capabilities *Capabilities `protobuf:"bytes,3,opt,name=capabilities,proto3" json:"capabilities,omitempty"`
	// Target id identifies which handler to use for a tunnel stream.
	TargetId string `protobuf:"bytes,4,opt,name=target_id,json=targetId,proto3" json:"target_id,omitempty"`
	// Error allows the register stream to return an error without breaking the
	// stream.
	Error string `protobuf:"bytes,5,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *Session) Reset() {
	*x = Session{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_tunnel_tunnel_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Session) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session) ProtoMessage() {}

func (x *Session) ProtoReflect() protoreflect.Message {
	mi := &file_proto_tunnel_tunnel_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session.ProtoReflect.Descriptor instead.
func (*Session) Descriptor() ([]byte, []int) {
	return file_proto_tunnel_tunnel_proto_rawDescGZIP(), []int{1}
}

func (x *Session) GetTag() int32 {
	if x != nil {
		return x.Tag
	}
	return 0
}

func (x *Session) GetAccept() bool {
	if x != nil {
		return x.Accept
	}
	return false
}

func (x *Session) GetCapabilities() *Capabilities {
	if x != nil {
		return x.Capabilities
	}
	return nil
}

func (x *Session) GetTargetId() string {
	if x != nil {
		return x.TargetId
	}
	return ""
}

func (x *Session) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type Capabilities struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Handler bool     `protobuf:"varint,1,opt,name=handler,proto3" json:"handler,omitempty"` // Whether a handler is installed on the endpoint.
	Target  []string `protobuf:"bytes,2,rep,name=target,proto3" json:"target,omitempty"`
}

func (x *Capabilities) Reset() {
	*x = Capabilities{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_tunnel_tunnel_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Capabilities) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Capabilities) ProtoMessage() {}

func (x *Capabilities) ProtoReflect() protoreflect.Message {
	mi := &file_proto_tunnel_tunnel_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Capabilities.ProtoReflect.Descriptor instead.
func (*Capabilities) Descriptor() ([]byte, []int) {
	return file_proto_tunnel_tunnel_proto_rawDescGZIP(), []int{2}
}

func (x *Capabilities) GetHandler() bool {
	if x != nil {
		return x.Handler
	}
	return false
}

func (x *Capabilities) GetTarget() []string {
	if x != nil {
		return x.Target
	}
	return nil
}

var File_proto_tunnel_tunnel_proto protoreflect.FileDescriptor

var file_proto_tunnel_tunnel_proto_rawDesc = []byte{
	0x0a, 0x19, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2f, 0x74,
	0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x67, 0x72, 0x70,
	0x63, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x22, 0x42, 0x0a, 0x04, 0x44, 0x61, 0x74, 0x61, 0x12,
	0x10, 0x0a, 0x03, 0x74, 0x61, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x74, 0x61,
	0x67, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x6c, 0x6f, 0x73, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x63, 0x6c, 0x6f, 0x73, 0x65, 0x22, 0xa4, 0x01, 0x0a, 0x07,
	0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x10, 0x0a, 0x03, 0x74, 0x61, 0x67, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x74, 0x61, 0x67, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x63, 0x63,
	0x65, 0x70, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x61, 0x63, 0x63, 0x65, 0x70,
	0x74, 0x12, 0x3c, 0x0a, 0x0c, 0x63, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65,
	0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x74, 0x75,
	0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65,
	0x73, 0x52, 0x0c, 0x63, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69, 0x65, 0x73, 0x12,
	0x1b, 0x0a, 0x09, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x08, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05,
	0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72,
	0x6f, 0x72, 0x22, 0x28, 0x0a, 0x0c, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x69,
	0x65, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x07, 0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x32, 0x74, 0x0a, 0x06,
	0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x12, 0x38, 0x0a, 0x08, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x65, 0x72, 0x12, 0x13, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2e,
	0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x1a, 0x13, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x74, 0x75,
	0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x28, 0x01, 0x30, 0x01,
	0x12, 0x30, 0x0a, 0x06, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x12, 0x10, 0x2e, 0x67, 0x72, 0x70,
	0x63, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x1a, 0x10, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x28, 0x01,
	0x30, 0x01, 0x42, 0x19, 0x5a, 0x17, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x75, 0x6e, 0x6e,
	0x65, 0x6c, 0x3b, 0x67, 0x72, 0x70, 0x63, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_tunnel_tunnel_proto_rawDescOnce sync.Once
	file_proto_tunnel_tunnel_proto_rawDescData = file_proto_tunnel_tunnel_proto_rawDesc
)

func file_proto_tunnel_tunnel_proto_rawDescGZIP() []byte {
	file_proto_tunnel_tunnel_proto_rawDescOnce.Do(func() {
		file_proto_tunnel_tunnel_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_tunnel_tunnel_proto_rawDescData)
	})
	return file_proto_tunnel_tunnel_proto_rawDescData
}

var file_proto_tunnel_tunnel_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_proto_tunnel_tunnel_proto_goTypes = []interface{}{
	(*Data)(nil),         // 0: grpctunnel.Data
	(*Session)(nil),      // 1: grpctunnel.Session
	(*Capabilities)(nil), // 2: grpctunnel.Capabilities
}
var file_proto_tunnel_tunnel_proto_depIdxs = []int32{
	2, // 0: grpctunnel.Session.capabilities:type_name -> grpctunnel.Capabilities
	1, // 1: grpctunnel.Tunnel.Register:input_type -> grpctunnel.Session
	0, // 2: grpctunnel.Tunnel.Tunnel:input_type -> grpctunnel.Data
	1, // 3: grpctunnel.Tunnel.Register:output_type -> grpctunnel.Session
	0, // 4: grpctunnel.Tunnel.Tunnel:output_type -> grpctunnel.Data
	3, // [3:5] is the sub-list for method output_type
	1, // [1:3] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proto_tunnel_tunnel_proto_init() }
func file_proto_tunnel_tunnel_proto_init() {
	if File_proto_tunnel_tunnel_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_tunnel_tunnel_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Data); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_tunnel_tunnel_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Session); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_tunnel_tunnel_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Capabilities); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_tunnel_tunnel_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_tunnel_tunnel_proto_goTypes,
		DependencyIndexes: file_proto_tunnel_tunnel_proto_depIdxs,
		MessageInfos:      file_proto_tunnel_tunnel_proto_msgTypes,
	}.Build()
	File_proto_tunnel_tunnel_proto = out.File
	file_proto_tunnel_tunnel_proto_rawDesc = nil
	file_proto_tunnel_tunnel_proto_goTypes = nil
	file_proto_tunnel_tunnel_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// TunnelClient is the client API for Tunnel service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type TunnelClient interface {
	// Register is used to request new Tunnel gRPCs.
	Register(ctx context.Context, opts ...grpc.CallOption) (Tunnel_RegisterClient, error)
	// Tunnel allows the tunnel client and server to create a bidirectional stream
	// in which data can be forwarded.
	Tunnel(ctx context.Context, opts ...grpc.CallOption) (Tunnel_TunnelClient, error)
}

type tunnelClient struct {
	cc grpc.ClientConnInterface
}

func NewTunnelClient(cc grpc.ClientConnInterface) TunnelClient {
	return &tunnelClient{cc}
}

func (c *tunnelClient) Register(ctx context.Context, opts ...grpc.CallOption) (Tunnel_RegisterClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Tunnel_serviceDesc.Streams[0], "/grpctunnel.Tunnel/Register", opts...)
	if err != nil {
		return nil, err
	}
	x := &tunnelRegisterClient{stream}
	return x, nil
}

type Tunnel_RegisterClient interface {
	Send(*Session) error
	Recv() (*Session, error)
	grpc.ClientStream
}

type tunnelRegisterClient struct {
	grpc.ClientStream
}

func (x *tunnelRegisterClient) Send(m *Session) error {
	return x.ClientStream.SendMsg(m)
}

func (x *tunnelRegisterClient) Recv() (*Session, error) {
	m := new(Session)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *tunnelClient) Tunnel(ctx context.Context, opts ...grpc.CallOption) (Tunnel_TunnelClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Tunnel_serviceDesc.Streams[1], "/grpctunnel.Tunnel/Tunnel", opts...)
	if err != nil {
		return nil, err
	}
	x := &tunnelTunnelClient{stream}
	return x, nil
}

type Tunnel_TunnelClient interface {
	Send(*Data) error
	Recv() (*Data, error)
	grpc.ClientStream
}

type tunnelTunnelClient struct {
	grpc.ClientStream
}

func (x *tunnelTunnelClient) Send(m *Data) error {
	return x.ClientStream.SendMsg(m)
}

func (x *tunnelTunnelClient) Recv() (*Data, error) {
	m := new(Data)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// TunnelServer is the server API for Tunnel service.
type TunnelServer interface {
	// Register is used to request new Tunnel gRPCs.
	Register(Tunnel_RegisterServer) error
	// Tunnel allows the tunnel client and server to create a bidirectional stream
	// in which data can be forwarded.
	Tunnel(Tunnel_TunnelServer) error
}

// UnimplementedTunnelServer can be embedded to have forward compatible implementations.
type UnimplementedTunnelServer struct {
}

func (*UnimplementedTunnelServer) Register(Tunnel_RegisterServer) error {
	return status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (*UnimplementedTunnelServer) Tunnel(Tunnel_TunnelServer) error {
	return status.Errorf(codes.Unimplemented, "method Tunnel not implemented")
}

func RegisterTunnelServer(s *grpc.Server, srv TunnelServer) {
	s.RegisterService(&_Tunnel_serviceDesc, srv)
}

func _Tunnel_Register_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(TunnelServer).Register(&tunnelRegisterServer{stream})
}

type Tunnel_RegisterServer interface {
	Send(*Session) error
	Recv() (*Session, error)
	grpc.ServerStream
}

type tunnelRegisterServer struct {
	grpc.ServerStream
}

func (x *tunnelRegisterServer) Send(m *Session) error {
	return x.ServerStream.SendMsg(m)
}

func (x *tunnelRegisterServer) Recv() (*Session, error) {
	m := new(Session)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Tunnel_Tunnel_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(TunnelServer).Tunnel(&tunnelTunnelServer{stream})
}

type Tunnel_TunnelServer interface {
	Send(*Data) error
	Recv() (*Data, error)
	grpc.ServerStream
}

type tunnelTunnelServer struct {
	grpc.ServerStream
}

func (x *tunnelTunnelServer) Send(m *Data) error {
	return x.ServerStream.SendMsg(m)
}

func (x *tunnelTunnelServer) Recv() (*Data, error) {
	m := new(Data)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _Tunnel_serviceDesc = grpc.ServiceDesc{
	ServiceName: "grpctunnel.Tunnel",
	HandlerType: (*TunnelServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Register",
			Handler:       _Tunnel_Register_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "Tunnel",
			Handler:       _Tunnel_Tunnel_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "proto/tunnel/tunnel.proto",
}
