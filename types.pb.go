// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.6.1
// source: types.proto

package zongzi

import (
	any1 "github.com/golang/protobuf/ptypes/any"
	field_mask "google.golang.org/genproto/protobuf/field_mask"
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

type Cmd_Method int32

const (
	Cmd_METHOD_UNSPECIFIED Cmd_Method = 0
	Cmd_METHOD_GET         Cmd_Method = 1
	Cmd_METHOD_PUT         Cmd_Method = 2
	Cmd_METHOD_PATCH       Cmd_Method = 3
	Cmd_METHOD_DELETE      Cmd_Method = 4
)

// Enum value maps for Cmd_Method.
var (
	Cmd_Method_name = map[int32]string{
		0: "METHOD_UNSPECIFIED",
		1: "METHOD_GET",
		2: "METHOD_PUT",
		3: "METHOD_PATCH",
		4: "METHOD_DELETE",
	}
	Cmd_Method_value = map[string]int32{
		"METHOD_UNSPECIFIED": 0,
		"METHOD_GET":         1,
		"METHOD_PUT":         2,
		"METHOD_PATCH":       3,
		"METHOD_DELETE":      4,
	}
)

func (x Cmd_Method) Enum() *Cmd_Method {
	p := new(Cmd_Method)
	*p = x
	return p
}

func (x Cmd_Method) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Cmd_Method) Descriptor() protoreflect.EnumDescriptor {
	return file_types_proto_enumTypes[0].Descriptor()
}

func (Cmd_Method) Type() protoreflect.EnumType {
	return &file_types_proto_enumTypes[0]
}

func (x Cmd_Method) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Cmd_Method.Descriptor instead.
func (Cmd_Method) EnumDescriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{0, 0}
}

type Cmd_Type int32

const (
	Cmd_TYPE_UNSPECIFIED Cmd_Type = 0
	Cmd_TYPE_HOST        Cmd_Type = 1
	Cmd_TYPE_SHARD       Cmd_Type = 2
	Cmd_TYPE_REPLICA     Cmd_Type = 3
	Cmd_TYPE_SNAPSHOT    Cmd_Type = 4
)

// Enum value maps for Cmd_Type.
var (
	Cmd_Type_name = map[int32]string{
		0: "TYPE_UNSPECIFIED",
		1: "TYPE_HOST",
		2: "TYPE_SHARD",
		3: "TYPE_REPLICA",
		4: "TYPE_SNAPSHOT",
	}
	Cmd_Type_value = map[string]int32{
		"TYPE_UNSPECIFIED": 0,
		"TYPE_HOST":        1,
		"TYPE_SHARD":       2,
		"TYPE_REPLICA":     3,
		"TYPE_SNAPSHOT":    4,
	}
)

func (x Cmd_Type) Enum() *Cmd_Type {
	p := new(Cmd_Type)
	*p = x
	return p
}

func (x Cmd_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Cmd_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_types_proto_enumTypes[1].Descriptor()
}

func (Cmd_Type) Type() protoreflect.EnumType {
	return &file_types_proto_enumTypes[1]
}

func (x Cmd_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Cmd_Type.Descriptor instead.
func (Cmd_Type) EnumDescriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{0, 1}
}

type Host_Status int32

const (
	Host_Status_UNSPECIFIED Host_Status = 0
	Host_Status_NEW         Host_Status = 1
	Host_Status_ACTIVE      Host_Status = 2
	Host_Status_MISSING     Host_Status = 3
	Host_Status_RECOVERING  Host_Status = 4
	Host_Status_GONE        Host_Status = 5
)

// Enum value maps for Host_Status.
var (
	Host_Status_name = map[int32]string{
		0: "Status_UNSPECIFIED",
		1: "Status_NEW",
		2: "Status_ACTIVE",
		3: "Status_MISSING",
		4: "Status_RECOVERING",
		5: "Status_GONE",
	}
	Host_Status_value = map[string]int32{
		"Status_UNSPECIFIED": 0,
		"Status_NEW":         1,
		"Status_ACTIVE":      2,
		"Status_MISSING":     3,
		"Status_RECOVERING":  4,
		"Status_GONE":        5,
	}
)

func (x Host_Status) Enum() *Host_Status {
	p := new(Host_Status)
	*p = x
	return p
}

func (x Host_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Host_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_types_proto_enumTypes[2].Descriptor()
}

func (Host_Status) Type() protoreflect.EnumType {
	return &file_types_proto_enumTypes[2]
}

func (x Host_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Host_Status.Descriptor instead.
func (Host_Status) EnumDescriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{1, 0}
}

type Shard_Status int32

const (
	Shard_Status_UNSPECIFIED Shard_Status = 0
	Shard_Status_NEW         Shard_Status = 1
	Shard_Status_ACTIVE      Shard_Status = 2
	Shard_Status_CLOSED      Shard_Status = 3
	Shard_Status_STARTING    Shard_Status = 4
	Shard_Status_UNAVAILABLE Shard_Status = 5
)

// Enum value maps for Shard_Status.
var (
	Shard_Status_name = map[int32]string{
		0: "Status_UNSPECIFIED",
		1: "Status_NEW",
		2: "Status_ACTIVE",
		3: "Status_CLOSED",
		4: "Status_STARTING",
		5: "Status_UNAVAILABLE",
	}
	Shard_Status_value = map[string]int32{
		"Status_UNSPECIFIED": 0,
		"Status_NEW":         1,
		"Status_ACTIVE":      2,
		"Status_CLOSED":      3,
		"Status_STARTING":    4,
		"Status_UNAVAILABLE": 5,
	}
)

func (x Shard_Status) Enum() *Shard_Status {
	p := new(Shard_Status)
	*p = x
	return p
}

func (x Shard_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Shard_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_types_proto_enumTypes[3].Descriptor()
}

func (Shard_Status) Type() protoreflect.EnumType {
	return &file_types_proto_enumTypes[3]
}

func (x Shard_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Shard_Status.Descriptor instead.
func (Shard_Status) EnumDescriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{2, 0}
}

type Replica_Status int32

const (
	Replica_Status_UNSPECIFIED Replica_Status = 0
	Replica_Status_NEW         Replica_Status = 1
	Replica_Status_ACTIVE      Replica_Status = 2
	Replica_Status_INACTIVE    Replica_Status = 3
	Replica_Status_DONE        Replica_Status = 4
)

// Enum value maps for Replica_Status.
var (
	Replica_Status_name = map[int32]string{
		0: "Status_UNSPECIFIED",
		1: "Status_NEW",
		2: "Status_ACTIVE",
		3: "Status_INACTIVE",
		4: "Status_DONE",
	}
	Replica_Status_value = map[string]int32{
		"Status_UNSPECIFIED": 0,
		"Status_NEW":         1,
		"Status_ACTIVE":      2,
		"Status_INACTIVE":    3,
		"Status_DONE":        4,
	}
)

func (x Replica_Status) Enum() *Replica_Status {
	p := new(Replica_Status)
	*p = x
	return p
}

func (x Replica_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Replica_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_types_proto_enumTypes[4].Descriptor()
}

func (Replica_Status) Type() protoreflect.EnumType {
	return &file_types_proto_enumTypes[4]
}

func (x Replica_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Replica_Status.Descriptor instead.
func (Replica_Status) EnumDescriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{3, 0}
}

type Cmd struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Method    Cmd_Method            `protobuf:"varint,1,opt,name=method,proto3,enum=Cmd_Method" json:"method,omitempty"`
	Type      Cmd_Type              `protobuf:"varint,2,opt,name=type,proto3,enum=Cmd_Type" json:"type,omitempty"`
	FieldMask *field_mask.FieldMask `protobuf:"bytes,14,opt,name=field_mask,json=fieldMask,proto3" json:"field_mask,omitempty"`
	Data      *any1.Any             `protobuf:"bytes,15,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *Cmd) Reset() {
	*x = Cmd{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Cmd) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Cmd) ProtoMessage() {}

func (x *Cmd) ProtoReflect() protoreflect.Message {
	mi := &file_types_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Cmd.ProtoReflect.Descriptor instead.
func (*Cmd) Descriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{0}
}

func (x *Cmd) GetMethod() Cmd_Method {
	if x != nil {
		return x.Method
	}
	return Cmd_METHOD_UNSPECIFIED
}

func (x *Cmd) GetType() Cmd_Type {
	if x != nil {
		return x.Type
	}
	return Cmd_TYPE_UNSPECIFIED
}

func (x *Cmd) GetFieldMask() *field_mask.FieldMask {
	if x != nil {
		return x.FieldMask
	}
	return nil
}

func (x *Cmd) GetData() *any1.Any {
	if x != nil {
		return x.Data
	}
	return nil
}

type Host struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id         uint64      `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Status     Host_Status `protobuf:"varint,2,opt,name=status,proto3,enum=Host_Status" json:"status,omitempty"`
	Address    string      `protobuf:"bytes,3,opt,name=address,proto3" json:"address,omitempty"`
	ShardTypes []string    `protobuf:"bytes,4,rep,name=shard_types,json=shardTypes,proto3" json:"shard_types,omitempty"`
	Meta       []byte      `protobuf:"bytes,5,opt,name=meta,proto3" json:"meta,omitempty"`
}

func (x *Host) Reset() {
	*x = Host{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Host) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Host) ProtoMessage() {}

func (x *Host) ProtoReflect() protoreflect.Message {
	mi := &file_types_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Host.ProtoReflect.Descriptor instead.
func (*Host) Descriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{1}
}

func (x *Host) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Host) GetStatus() Host_Status {
	if x != nil {
		return x.Status
	}
	return Host_Status_UNSPECIFIED
}

func (x *Host) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *Host) GetShardTypes() []string {
	if x != nil {
		return x.ShardTypes
	}
	return nil
}

func (x *Host) GetMeta() []byte {
	if x != nil {
		return x.Meta
	}
	return nil
}

type Shard struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      uint64       `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Status  Shard_Status `protobuf:"varint,2,opt,name=status,proto3,enum=Shard_Status" json:"status,omitempty"`
	Type    string       `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
	Version uint64       `protobuf:"varint,4,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *Shard) Reset() {
	*x = Shard{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Shard) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Shard) ProtoMessage() {}

func (x *Shard) ProtoReflect() protoreflect.Message {
	mi := &file_types_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Shard.ProtoReflect.Descriptor instead.
func (*Shard) Descriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{2}
}

func (x *Shard) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Shard) GetStatus() Shard_Status {
	if x != nil {
		return x.Status
	}
	return Shard_Status_UNSPECIFIED
}

func (x *Shard) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *Shard) GetVersion() uint64 {
	if x != nil {
		return x.Version
	}
	return 0
}

type Replica struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id          uint64         `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Status      Replica_Status `protobuf:"varint,2,opt,name=status,proto3,enum=Replica_Status" json:"status,omitempty"`
	ShardId     uint64         `protobuf:"varint,3,opt,name=shard_id,json=shardId,proto3" json:"shard_id,omitempty"`
	IsNonVoting bool           `protobuf:"varint,4,opt,name=is_non_voting,json=isNonVoting,proto3" json:"is_non_voting,omitempty"`
	IsWitness   bool           `protobuf:"varint,5,opt,name=is_witness,json=isWitness,proto3" json:"is_witness,omitempty"`
}

func (x *Replica) Reset() {
	*x = Replica{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Replica) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Replica) ProtoMessage() {}

func (x *Replica) ProtoReflect() protoreflect.Message {
	mi := &file_types_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Replica.ProtoReflect.Descriptor instead.
func (*Replica) Descriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{3}
}

func (x *Replica) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Replica) GetStatus() Replica_Status {
	if x != nil {
		return x.Status
	}
	return Replica_Status_UNSPECIFIED
}

func (x *Replica) GetShardId() uint64 {
	if x != nil {
		return x.ShardId
	}
	return 0
}

func (x *Replica) GetIsNonVoting() bool {
	if x != nil {
		return x.IsNonVoting
	}
	return false
}

func (x *Replica) GetIsWitness() bool {
	if x != nil {
		return x.IsWitness
	}
	return false
}

type Snapshot struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Index    uint64     `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
	Hosts    []*Host    `protobuf:"bytes,2,rep,name=hosts,proto3" json:"hosts,omitempty"`
	Shards   []*Shard   `protobuf:"bytes,3,rep,name=shards,proto3" json:"shards,omitempty"`
	Replicas []*Replica `protobuf:"bytes,4,rep,name=replicas,proto3" json:"replicas,omitempty"`
}

func (x *Snapshot) Reset() {
	*x = Snapshot{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Snapshot) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Snapshot) ProtoMessage() {}

func (x *Snapshot) ProtoReflect() protoreflect.Message {
	mi := &file_types_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Snapshot.ProtoReflect.Descriptor instead.
func (*Snapshot) Descriptor() ([]byte, []int) {
	return file_types_proto_rawDescGZIP(), []int{4}
}

func (x *Snapshot) GetIndex() uint64 {
	if x != nil {
		return x.Index
	}
	return 0
}

func (x *Snapshot) GetHosts() []*Host {
	if x != nil {
		return x.Hosts
	}
	return nil
}

func (x *Snapshot) GetShards() []*Shard {
	if x != nil {
		return x.Shards
	}
	return nil
}

func (x *Snapshot) GetReplicas() []*Replica {
	if x != nil {
		return x.Replicas
	}
	return nil
}

var File_types_proto protoreflect.FileDescriptor

var file_types_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x19, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x61,
	0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x5f,
	0x6d, 0x61, 0x73, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xf7, 0x02, 0x0a, 0x03, 0x63,
	0x6d, 0x64, 0x12, 0x23, 0x0a, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x0b, 0x2e, 0x63, 0x6d, 0x64, 0x2e, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x52,
	0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x1d, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x09, 0x2e, 0x63, 0x6d, 0x64, 0x2e, 0x54, 0x79, 0x70, 0x65,
	0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x39, 0x0a, 0x0a, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x5f,
	0x6d, 0x61, 0x73, 0x6b, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65,
	0x6c, 0x64, 0x4d, 0x61, 0x73, 0x6b, 0x52, 0x09, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x4d, 0x61, 0x73,
	0x6b, 0x12, 0x28, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x14, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x41, 0x6e, 0x79, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x65, 0x0a, 0x06, 0x4d,
	0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x16, 0x0a, 0x12, 0x4d, 0x45, 0x54, 0x48, 0x4f, 0x44, 0x5f,
	0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0e, 0x0a,
	0x0a, 0x4d, 0x45, 0x54, 0x48, 0x4f, 0x44, 0x5f, 0x47, 0x45, 0x54, 0x10, 0x01, 0x12, 0x0e, 0x0a,
	0x0a, 0x4d, 0x45, 0x54, 0x48, 0x4f, 0x44, 0x5f, 0x50, 0x55, 0x54, 0x10, 0x02, 0x12, 0x10, 0x0a,
	0x0c, 0x4d, 0x45, 0x54, 0x48, 0x4f, 0x44, 0x5f, 0x50, 0x41, 0x54, 0x43, 0x48, 0x10, 0x03, 0x12,
	0x11, 0x0a, 0x0d, 0x4d, 0x45, 0x54, 0x48, 0x4f, 0x44, 0x5f, 0x44, 0x45, 0x4c, 0x45, 0x54, 0x45,
	0x10, 0x04, 0x22, 0x60, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x14, 0x0a, 0x10, 0x54, 0x59,
	0x50, 0x45, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00,
	0x12, 0x0d, 0x0a, 0x09, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x48, 0x4f, 0x53, 0x54, 0x10, 0x01, 0x12,
	0x0e, 0x0a, 0x0a, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x53, 0x48, 0x41, 0x52, 0x44, 0x10, 0x02, 0x12,
	0x10, 0x0a, 0x0c, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x52, 0x45, 0x50, 0x4c, 0x49, 0x43, 0x41, 0x10,
	0x03, 0x12, 0x11, 0x0a, 0x0d, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x53, 0x4e, 0x41, 0x50, 0x53, 0x48,
	0x4f, 0x54, 0x10, 0x04, 0x22, 0x8c, 0x02, 0x0a, 0x04, 0x48, 0x6f, 0x73, 0x74, 0x12, 0x0e, 0x0a,
	0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x12, 0x24, 0x0a,
	0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0c, 0x2e,
	0x48, 0x6f, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x1f, 0x0a,
	0x0b, 0x73, 0x68, 0x61, 0x72, 0x64, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x18, 0x04, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x0a, 0x73, 0x68, 0x61, 0x72, 0x64, 0x54, 0x79, 0x70, 0x65, 0x73, 0x12, 0x12,
	0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x6d, 0x65,
	0x74, 0x61, 0x22, 0x7f, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x16, 0x0a, 0x12,
	0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49,
	0x45, 0x44, 0x10, 0x00, 0x12, 0x0e, 0x0a, 0x0a, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x4e,
	0x45, 0x57, 0x10, 0x01, 0x12, 0x11, 0x0a, 0x0d, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x41,
	0x43, 0x54, 0x49, 0x56, 0x45, 0x10, 0x02, 0x12, 0x12, 0x0a, 0x0e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x5f, 0x4d, 0x49, 0x53, 0x53, 0x49, 0x4e, 0x47, 0x10, 0x03, 0x12, 0x15, 0x0a, 0x11, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x52, 0x45, 0x43, 0x4f, 0x56, 0x45, 0x52, 0x49, 0x4e, 0x47,
	0x10, 0x04, 0x12, 0x0f, 0x0a, 0x0b, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x47, 0x4f, 0x4e,
	0x45, 0x10, 0x05, 0x22, 0xf2, 0x01, 0x0a, 0x05, 0x53, 0x68, 0x61, 0x72, 0x64, 0x12, 0x0e, 0x0a,
	0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x12, 0x25, 0x0a,
	0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0d, 0x2e,
	0x53, 0x68, 0x61, 0x72, 0x64, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x22, 0x83, 0x01, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x16, 0x0a,
	0x12, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46,
	0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0e, 0x0a, 0x0a, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f,
	0x4e, 0x45, 0x57, 0x10, 0x01, 0x12, 0x11, 0x0a, 0x0d, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f,
	0x41, 0x43, 0x54, 0x49, 0x56, 0x45, 0x10, 0x02, 0x12, 0x11, 0x0a, 0x0d, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x5f, 0x43, 0x4c, 0x4f, 0x53, 0x45, 0x44, 0x10, 0x03, 0x12, 0x13, 0x0a, 0x0f, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x53, 0x54, 0x41, 0x52, 0x54, 0x49, 0x4e, 0x47, 0x10, 0x04,
	0x12, 0x16, 0x0a, 0x12, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x55, 0x4e, 0x41, 0x56, 0x41,
	0x49, 0x4c, 0x41, 0x42, 0x4c, 0x45, 0x10, 0x05, 0x22, 0x8b, 0x02, 0x0a, 0x07, 0x52, 0x65, 0x70,
	0x6c, 0x69, 0x63, 0x61, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x02, 0x69, 0x64, 0x12, 0x27, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x0f, 0x2e, 0x52, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x2e, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x19, 0x0a,
	0x08, 0x73, 0x68, 0x61, 0x72, 0x64, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x07, 0x73, 0x68, 0x61, 0x72, 0x64, 0x49, 0x64, 0x12, 0x22, 0x0a, 0x0d, 0x69, 0x73, 0x5f, 0x6e,
	0x6f, 0x6e, 0x5f, 0x76, 0x6f, 0x74, 0x69, 0x6e, 0x67, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x0b, 0x69, 0x73, 0x4e, 0x6f, 0x6e, 0x56, 0x6f, 0x74, 0x69, 0x6e, 0x67, 0x12, 0x1d, 0x0a, 0x0a,
	0x69, 0x73, 0x5f, 0x77, 0x69, 0x74, 0x6e, 0x65, 0x73, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x09, 0x69, 0x73, 0x57, 0x69, 0x74, 0x6e, 0x65, 0x73, 0x73, 0x22, 0x69, 0x0a, 0x06, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x16, 0x0a, 0x12, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f,
	0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0e, 0x0a,
	0x0a, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x4e, 0x45, 0x57, 0x10, 0x01, 0x12, 0x11, 0x0a,
	0x0d, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x56, 0x45, 0x10, 0x02,
	0x12, 0x13, 0x0a, 0x0f, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f, 0x49, 0x4e, 0x41, 0x43, 0x54,
	0x49, 0x56, 0x45, 0x10, 0x03, 0x12, 0x0f, 0x0a, 0x0b, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5f,
	0x44, 0x4f, 0x4e, 0x45, 0x10, 0x04, 0x22, 0x83, 0x01, 0x0a, 0x08, 0x53, 0x6e, 0x61, 0x70, 0x73,
	0x68, 0x6f, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x1b, 0x0a, 0x05, 0x68, 0x6f, 0x73,
	0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x05, 0x2e, 0x48, 0x6f, 0x73, 0x74, 0x52,
	0x05, 0x68, 0x6f, 0x73, 0x74, 0x73, 0x12, 0x1e, 0x0a, 0x06, 0x73, 0x68, 0x61, 0x72, 0x64, 0x73,
	0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x06, 0x2e, 0x53, 0x68, 0x61, 0x72, 0x64, 0x52, 0x06,
	0x73, 0x68, 0x61, 0x72, 0x64, 0x73, 0x12, 0x24, 0x0a, 0x08, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63,
	0x61, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x08, 0x2e, 0x52, 0x65, 0x70, 0x6c, 0x69,
	0x63, 0x61, 0x52, 0x08, 0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x73, 0x42, 0x19, 0x5a, 0x17,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6c, 0x6f, 0x67, 0x62, 0x6e,
	0x2f, 0x7a, 0x6f, 0x6e, 0x67, 0x7a, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_types_proto_rawDescOnce sync.Once
	file_types_proto_rawDescData = file_types_proto_rawDesc
)

func file_types_proto_rawDescGZIP() []byte {
	file_types_proto_rawDescOnce.Do(func() {
		file_types_proto_rawDescData = protoimpl.X.CompressGZIP(file_types_proto_rawDescData)
	})
	return file_types_proto_rawDescData
}

var file_types_proto_enumTypes = make([]protoimpl.EnumInfo, 5)
var file_types_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_types_proto_goTypes = []interface{}{
	(Cmd_Method)(0),              // 0: cmd.Method
	(Cmd_Type)(0),                // 1: cmd.Type
	(Host_Status)(0),             // 2: Host.Status
	(Shard_Status)(0),            // 3: Shard.Status
	(Replica_Status)(0),          // 4: Replica.Status
	(*Cmd)(nil),                  // 5: cmd
	(*Host)(nil),                 // 6: Host
	(*Shard)(nil),                // 7: Shard
	(*Replica)(nil),              // 8: Replica
	(*Snapshot)(nil),             // 9: Snapshot
	(*field_mask.FieldMask)(nil), // 10: google.protobuf.FieldMask
	(*any1.Any)(nil),             // 11: google.protobuf.Any
}
var file_types_proto_depIdxs = []int32{
	0,  // 0: cmd.method:type_name -> cmd.Method
	1,  // 1: cmd.type:type_name -> cmd.Type
	10, // 2: cmd.field_mask:type_name -> google.protobuf.FieldMask
	11, // 3: cmd.data:type_name -> google.protobuf.Any
	2,  // 4: Host.status:type_name -> Host.Status
	3,  // 5: Shard.status:type_name -> Shard.Status
	4,  // 6: Replica.status:type_name -> Replica.Status
	6,  // 7: Snapshot.hosts:type_name -> Host
	7,  // 8: Snapshot.shards:type_name -> Shard
	8,  // 9: Snapshot.replicas:type_name -> Replica
	10, // [10:10] is the sub-list for method output_type
	10, // [10:10] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_types_proto_init() }
func file_types_proto_init() {
	if File_types_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_types_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Cmd); i {
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
		file_types_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Host); i {
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
		file_types_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Shard); i {
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
		file_types_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Replica); i {
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
		file_types_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Snapshot); i {
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
			RawDescriptor: file_types_proto_rawDesc,
			NumEnums:      5,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_types_proto_goTypes,
		DependencyIndexes: file_types_proto_depIdxs,
		EnumInfos:         file_types_proto_enumTypes,
		MessageInfos:      file_types_proto_msgTypes,
	}.Build()
	File_types_proto = out.File
	file_types_proto_rawDesc = nil
	file_types_proto_goTypes = nil
	file_types_proto_depIdxs = nil
}
