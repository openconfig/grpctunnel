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

// Version: 0.1

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.12.4
// source: cmd/target/proto/config/target_config.proto

package targetconfig

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

type TLS struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// If both cert_file and key_file are provided, then mTLS will be used.
	// Otherwise, server side TLS will be used (ca_file used for client side for validation).
	CertFile string `protobuf:"bytes,1,opt,name=cert_file,json=certFile,proto3" json:"cert_file,omitempty"`
	KeyFile  string `protobuf:"bytes,2,opt,name=key_file,json=keyFile,proto3" json:"key_file,omitempty"`
	CaFile   string `protobuf:"bytes,3,opt,name=ca_file,json=caFile,proto3" json:"ca_file,omitempty"`
}

func (x *TLS) Reset() {
	*x = TLS{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TLS) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TLS) ProtoMessage() {}

func (x *TLS) ProtoReflect() protoreflect.Message {
	mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TLS.ProtoReflect.Descriptor instead.
func (*TLS) Descriptor() ([]byte, []int) {
	return file_cmd_target_proto_config_target_config_proto_rawDescGZIP(), []int{0}
}

func (x *TLS) GetCertFile() string {
	if x != nil {
		return x.CertFile
	}
	return ""
}

func (x *TLS) GetKeyFile() string {
	if x != nil {
		return x.KeyFile
	}
	return ""
}

func (x *TLS) GetCaFile() string {
	if x != nil {
		return x.CaFile
	}
	return ""
}

type Credentials struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tls *TLS `protobuf:"bytes,1,opt,name=tls,proto3" json:"tls,omitempty"`
}

func (x *Credentials) Reset() {
	*x = Credentials{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Credentials) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Credentials) ProtoMessage() {}

func (x *Credentials) ProtoReflect() protoreflect.Message {
	mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Credentials.ProtoReflect.Descriptor instead.
func (*Credentials) Descriptor() ([]byte, []int) {
	return file_cmd_target_proto_config_target_config_proto_rawDescGZIP(), []int{1}
}

func (x *Credentials) GetTls() *TLS {
	if x != nil {
		return x.Tls
	}
	return nil
}

type TunnelServer struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TunnelServerAddress string       `protobuf:"bytes,1,opt,name=tunnel_server_address,json=tunnelServerAddress,proto3" json:"tunnel_server_address,omitempty"`
	Credentials         *Credentials `protobuf:"bytes,2,opt,name=credentials,proto3" json:"credentials,omitempty"`
}

func (x *TunnelServer) Reset() {
	*x = TunnelServer{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TunnelServer) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TunnelServer) ProtoMessage() {}

func (x *TunnelServer) ProtoReflect() protoreflect.Message {
	mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TunnelServer.ProtoReflect.Descriptor instead.
func (*TunnelServer) Descriptor() ([]byte, []int) {
	return file_cmd_target_proto_config_target_config_proto_rawDescGZIP(), []int{2}
}

func (x *TunnelServer) GetTunnelServerAddress() string {
	if x != nil {
		return x.TunnelServerAddress
	}
	return ""
}

func (x *TunnelServer) GetCredentials() *Credentials {
	if x != nil {
		return x.Credentials
	}
	return nil
}

type TunnelTarget struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TunnelServer []*TunnelServer `protobuf:"bytes,1,rep,name=tunnel_server,json=tunnelServer,proto3" json:"tunnel_server,omitempty"`
	Target       string          `protobuf:"bytes,2,opt,name=target,proto3" json:"target,omitempty"`
	Type         string          `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
	DialAddress  string          `protobuf:"bytes,4,opt,name=dial_address,json=dialAddress,proto3" json:"dial_address,omitempty"`
}

func (x *TunnelTarget) Reset() {
	*x = TunnelTarget{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TunnelTarget) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TunnelTarget) ProtoMessage() {}

func (x *TunnelTarget) ProtoReflect() protoreflect.Message {
	mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TunnelTarget.ProtoReflect.Descriptor instead.
func (*TunnelTarget) Descriptor() ([]byte, []int) {
	return file_cmd_target_proto_config_target_config_proto_rawDescGZIP(), []int{3}
}

func (x *TunnelTarget) GetTunnelServer() []*TunnelServer {
	if x != nil {
		return x.TunnelServer
	}
	return nil
}

func (x *TunnelTarget) GetTarget() string {
	if x != nil {
		return x.Target
	}
	return ""
}

func (x *TunnelTarget) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *TunnelTarget) GetDialAddress() string {
	if x != nil {
		return x.DialAddress
	}
	return ""
}

type TunnelTargetConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Tunnels dialed for targets if not specified within a given target.
	TunnelServerDefault []*TunnelServer `protobuf:"bytes,1,rep,name=tunnel_server_default,json=tunnelServerDefault,proto3" json:"tunnel_server_default,omitempty"`
	// Targets made available over remote tunnel servers.
	TunnelTarget []*TunnelTarget `protobuf:"bytes,2,rep,name=tunnel_target,json=tunnelTarget,proto3" json:"tunnel_target,omitempty"`
}

func (x *TunnelTargetConfig) Reset() {
	*x = TunnelTargetConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TunnelTargetConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TunnelTargetConfig) ProtoMessage() {}

func (x *TunnelTargetConfig) ProtoReflect() protoreflect.Message {
	mi := &file_cmd_target_proto_config_target_config_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TunnelTargetConfig.ProtoReflect.Descriptor instead.
func (*TunnelTargetConfig) Descriptor() ([]byte, []int) {
	return file_cmd_target_proto_config_target_config_proto_rawDescGZIP(), []int{4}
}

func (x *TunnelTargetConfig) GetTunnelServerDefault() []*TunnelServer {
	if x != nil {
		return x.TunnelServerDefault
	}
	return nil
}

func (x *TunnelTargetConfig) GetTunnelTarget() []*TunnelTarget {
	if x != nil {
		return x.TunnelTarget
	}
	return nil
}

var File_cmd_target_proto_config_target_config_proto protoreflect.FileDescriptor

var file_cmd_target_proto_config_target_config_proto_rawDesc = []byte{
	0x0a, 0x2b, 0x63, 0x6d, 0x64, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0c, 0x74,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22, 0x56, 0x0a, 0x03, 0x54,
	0x4c, 0x53, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x65, 0x72, 0x74, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x65, 0x72, 0x74, 0x46, 0x69, 0x6c, 0x65, 0x12,
	0x19, 0x0a, 0x08, 0x6b, 0x65, 0x79, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x6b, 0x65, 0x79, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x63, 0x61,
	0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x61, 0x46,
	0x69, 0x6c, 0x65, 0x22, 0x32, 0x0a, 0x0b, 0x43, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61,
	0x6c, 0x73, 0x12, 0x23, 0x0a, 0x03, 0x74, 0x6c, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x11, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x54,
	0x4c, 0x53, 0x52, 0x03, 0x74, 0x6c, 0x73, 0x22, 0x7f, 0x0a, 0x0c, 0x54, 0x75, 0x6e, 0x6e, 0x65,
	0x6c, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x32, 0x0a, 0x15, 0x74, 0x75, 0x6e, 0x6e, 0x65,
	0x6c, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x53, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x3b, 0x0a, 0x0b, 0x63,
	0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x19, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e,
	0x43, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x52, 0x0b, 0x63, 0x72, 0x65,
	0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x22, 0x9e, 0x01, 0x0a, 0x0c, 0x54, 0x75, 0x6e,
	0x6e, 0x65, 0x6c, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x12, 0x3f, 0x0a, 0x0d, 0x74, 0x75, 0x6e,
	0x6e, 0x65, 0x6c, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x1a, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e,
	0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x52, 0x0c, 0x74, 0x75,
	0x6e, 0x6e, 0x65, 0x6c, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x16, 0x0a, 0x06, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x21, 0x0a, 0x0c, 0x64, 0x69, 0x61, 0x6c, 0x5f, 0x61,
	0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x69,
	0x61, 0x6c, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0xa5, 0x01, 0x0a, 0x12, 0x54, 0x75,
	0x6e, 0x6e, 0x65, 0x6c, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x12, 0x4e, 0x0a, 0x15, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x5f, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x1a, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x54,
	0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x52, 0x13, 0x74, 0x75, 0x6e,
	0x6e, 0x65, 0x6c, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x44, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74,
	0x12, 0x3f, 0x0a, 0x0d, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x5f, 0x74, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x54, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x52, 0x0c, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x42, 0x40, 0x5a, 0x3e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x6f, 0x70, 0x65, 0x6e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x74,
	0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2f, 0x63, 0x6d, 0x64, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cmd_target_proto_config_target_config_proto_rawDescOnce sync.Once
	file_cmd_target_proto_config_target_config_proto_rawDescData = file_cmd_target_proto_config_target_config_proto_rawDesc
)

func file_cmd_target_proto_config_target_config_proto_rawDescGZIP() []byte {
	file_cmd_target_proto_config_target_config_proto_rawDescOnce.Do(func() {
		file_cmd_target_proto_config_target_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_cmd_target_proto_config_target_config_proto_rawDescData)
	})
	return file_cmd_target_proto_config_target_config_proto_rawDescData
}

var file_cmd_target_proto_config_target_config_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_cmd_target_proto_config_target_config_proto_goTypes = []interface{}{
	(*TLS)(nil),                // 0: targetconfig.TLS
	(*Credentials)(nil),        // 1: targetconfig.Credentials
	(*TunnelServer)(nil),       // 2: targetconfig.TunnelServer
	(*TunnelTarget)(nil),       // 3: targetconfig.TunnelTarget
	(*TunnelTargetConfig)(nil), // 4: targetconfig.TunnelTargetConfig
}
var file_cmd_target_proto_config_target_config_proto_depIdxs = []int32{
	0, // 0: targetconfig.Credentials.tls:type_name -> targetconfig.TLS
	1, // 1: targetconfig.TunnelServer.credentials:type_name -> targetconfig.Credentials
	2, // 2: targetconfig.TunnelTarget.tunnel_server:type_name -> targetconfig.TunnelServer
	2, // 3: targetconfig.TunnelTargetConfig.tunnel_server_default:type_name -> targetconfig.TunnelServer
	3, // 4: targetconfig.TunnelTargetConfig.tunnel_target:type_name -> targetconfig.TunnelTarget
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_cmd_target_proto_config_target_config_proto_init() }
func file_cmd_target_proto_config_target_config_proto_init() {
	if File_cmd_target_proto_config_target_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cmd_target_proto_config_target_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TLS); i {
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
		file_cmd_target_proto_config_target_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Credentials); i {
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
		file_cmd_target_proto_config_target_config_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TunnelServer); i {
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
		file_cmd_target_proto_config_target_config_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TunnelTarget); i {
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
		file_cmd_target_proto_config_target_config_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TunnelTargetConfig); i {
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
			RawDescriptor: file_cmd_target_proto_config_target_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cmd_target_proto_config_target_config_proto_goTypes,
		DependencyIndexes: file_cmd_target_proto_config_target_config_proto_depIdxs,
		MessageInfos:      file_cmd_target_proto_config_target_config_proto_msgTypes,
	}.Build()
	File_cmd_target_proto_config_target_config_proto = out.File
	file_cmd_target_proto_config_target_config_proto_rawDesc = nil
	file_cmd_target_proto_config_target_config_proto_goTypes = nil
	file_cmd_target_proto_config_target_config_proto_depIdxs = nil
}
