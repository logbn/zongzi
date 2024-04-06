// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.12.4
// source: zongzi.proto

package internal

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Zongzi_Ping_FullMethodName      = "/zongzi.Zongzi/Ping"
	Zongzi_Probe_FullMethodName     = "/zongzi.Zongzi/Probe"
	Zongzi_Info_FullMethodName      = "/zongzi.Zongzi/Info"
	Zongzi_Members_FullMethodName   = "/zongzi.Zongzi/Members"
	Zongzi_Join_FullMethodName      = "/zongzi.Zongzi/Join"
	Zongzi_ShardJoin_FullMethodName = "/zongzi.Zongzi/ShardJoin"
	Zongzi_Apply_FullMethodName     = "/zongzi.Zongzi/Apply"
	Zongzi_Commit_FullMethodName    = "/zongzi.Zongzi/Commit"
	Zongzi_Read_FullMethodName      = "/zongzi.Zongzi/Read"
	Zongzi_Watch_FullMethodName     = "/zongzi.Zongzi/Watch"
)

// ZongziClient is the client API for Zongzi service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ZongziClient interface {
	// Ping is a noop for timing purposes
	Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error)
	// Probe returns Gossip.AdvertiseAddress
	// Used by new hosts to start dragonboat
	Probe(ctx context.Context, in *ProbeRequest, opts ...grpc.CallOption) (*ProbeResponse, error)
	// Info returns zero shard replicaID and hostID
	// Used by new hosts to discover zero shard replicas
	Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error)
	// Members returns json marshaled map of member zero shard replicaID to hostID
	// Used by new hosts joining a bootstrapped zero shard
	Members(ctx context.Context, in *MembersRequest, opts ...grpc.CallOption) (*MembersResponse, error)
	// Join takes a host ID and returns success
	// Used by joining hosts to request their own addition to the zero shard
	Join(ctx context.Context, in *JoinRequest, opts ...grpc.CallOption) (*JoinResponse, error)
	// ShardJoin takes a replica ID and host ID and returns success
	// Used during replica creation to request replica's addition to the shard
	ShardJoin(ctx context.Context, in *ShardJoinRequest, opts ...grpc.CallOption) (*ShardJoinResponse, error)
	// Apply provides unary request/response command forwarding
	Apply(ctx context.Context, in *ApplyRequest, opts ...grpc.CallOption) (*ApplyResponse, error)
	// Commit provides unary request/response command forwarding
	Commit(ctx context.Context, in *CommitRequest, opts ...grpc.CallOption) (*CommitResponse, error)
	// Read provides unary request/response query forwarding
	Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (*ReadResponse, error)
	// Watch provides streaming query response forwarding
	Watch(ctx context.Context, in *WatchRequest, opts ...grpc.CallOption) (Zongzi_WatchClient, error)
}

type zongziClient struct {
	cc grpc.ClientConnInterface
}

func NewZongziClient(cc grpc.ClientConnInterface) ZongziClient {
	return &zongziClient{cc}
}

func (c *zongziClient) Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error) {
	out := new(PingResponse)
	err := c.cc.Invoke(ctx, Zongzi_Ping_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zongziClient) Probe(ctx context.Context, in *ProbeRequest, opts ...grpc.CallOption) (*ProbeResponse, error) {
	out := new(ProbeResponse)
	err := c.cc.Invoke(ctx, Zongzi_Probe_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zongziClient) Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error) {
	out := new(InfoResponse)
	err := c.cc.Invoke(ctx, Zongzi_Info_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zongziClient) Members(ctx context.Context, in *MembersRequest, opts ...grpc.CallOption) (*MembersResponse, error) {
	out := new(MembersResponse)
	err := c.cc.Invoke(ctx, Zongzi_Members_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zongziClient) Join(ctx context.Context, in *JoinRequest, opts ...grpc.CallOption) (*JoinResponse, error) {
	out := new(JoinResponse)
	err := c.cc.Invoke(ctx, Zongzi_Join_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zongziClient) ShardJoin(ctx context.Context, in *ShardJoinRequest, opts ...grpc.CallOption) (*ShardJoinResponse, error) {
	out := new(ShardJoinResponse)
	err := c.cc.Invoke(ctx, Zongzi_ShardJoin_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zongziClient) Apply(ctx context.Context, in *ApplyRequest, opts ...grpc.CallOption) (*ApplyResponse, error) {
	out := new(ApplyResponse)
	err := c.cc.Invoke(ctx, Zongzi_Apply_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zongziClient) Commit(ctx context.Context, in *CommitRequest, opts ...grpc.CallOption) (*CommitResponse, error) {
	out := new(CommitResponse)
	err := c.cc.Invoke(ctx, Zongzi_Commit_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zongziClient) Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (*ReadResponse, error) {
	out := new(ReadResponse)
	err := c.cc.Invoke(ctx, Zongzi_Read_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *zongziClient) Watch(ctx context.Context, in *WatchRequest, opts ...grpc.CallOption) (Zongzi_WatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &Zongzi_ServiceDesc.Streams[0], Zongzi_Watch_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &zongziWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Zongzi_WatchClient interface {
	Recv() (*WatchResponse, error)
	grpc.ClientStream
}

type zongziWatchClient struct {
	grpc.ClientStream
}

func (x *zongziWatchClient) Recv() (*WatchResponse, error) {
	m := new(WatchResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ZongziServer is the server API for Zongzi service.
// All implementations must embed UnimplementedZongziServer
// for forward compatibility
type ZongziServer interface {
	// Ping is a noop for timing purposes
	Ping(context.Context, *PingRequest) (*PingResponse, error)
	// Probe returns Gossip.AdvertiseAddress
	// Used by new hosts to start dragonboat
	Probe(context.Context, *ProbeRequest) (*ProbeResponse, error)
	// Info returns zero shard replicaID and hostID
	// Used by new hosts to discover zero shard replicas
	Info(context.Context, *InfoRequest) (*InfoResponse, error)
	// Members returns json marshaled map of member zero shard replicaID to hostID
	// Used by new hosts joining a bootstrapped zero shard
	Members(context.Context, *MembersRequest) (*MembersResponse, error)
	// Join takes a host ID and returns success
	// Used by joining hosts to request their own addition to the zero shard
	Join(context.Context, *JoinRequest) (*JoinResponse, error)
	// ShardJoin takes a replica ID and host ID and returns success
	// Used during replica creation to request replica's addition to the shard
	ShardJoin(context.Context, *ShardJoinRequest) (*ShardJoinResponse, error)
	// Apply provides unary request/response command forwarding
	Apply(context.Context, *ApplyRequest) (*ApplyResponse, error)
	// Commit provides unary request/response command forwarding
	Commit(context.Context, *CommitRequest) (*CommitResponse, error)
	// Read provides unary request/response query forwarding
	Read(context.Context, *ReadRequest) (*ReadResponse, error)
	// Watch provides streaming query response forwarding
	Watch(*WatchRequest, Zongzi_WatchServer) error
	mustEmbedUnimplementedZongziServer()
}

// UnimplementedZongziServer must be embedded to have forward compatible implementations.
type UnimplementedZongziServer struct {
}

func (UnimplementedZongziServer) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedZongziServer) Probe(context.Context, *ProbeRequest) (*ProbeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Probe not implemented")
}
func (UnimplementedZongziServer) Info(context.Context, *InfoRequest) (*InfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Info not implemented")
}
func (UnimplementedZongziServer) Members(context.Context, *MembersRequest) (*MembersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Members not implemented")
}
func (UnimplementedZongziServer) Join(context.Context, *JoinRequest) (*JoinResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Join not implemented")
}
func (UnimplementedZongziServer) ShardJoin(context.Context, *ShardJoinRequest) (*ShardJoinResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ShardJoin not implemented")
}
func (UnimplementedZongziServer) Apply(context.Context, *ApplyRequest) (*ApplyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Apply not implemented")
}
func (UnimplementedZongziServer) Commit(context.Context, *CommitRequest) (*CommitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Commit not implemented")
}
func (UnimplementedZongziServer) Read(context.Context, *ReadRequest) (*ReadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Read not implemented")
}
func (UnimplementedZongziServer) Watch(*WatchRequest, Zongzi_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}
func (UnimplementedZongziServer) mustEmbedUnimplementedZongziServer() {}

// UnsafeZongziServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ZongziServer will
// result in compilation errors.
type UnsafeZongziServer interface {
	mustEmbedUnimplementedZongziServer()
}

func RegisterZongziServer(s grpc.ServiceRegistrar, srv ZongziServer) {
	s.RegisterService(&Zongzi_ServiceDesc, srv)
}

func _Zongzi_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZongziServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Zongzi_Ping_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZongziServer).Ping(ctx, req.(*PingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Zongzi_Probe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProbeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZongziServer).Probe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Zongzi_Probe_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZongziServer).Probe(ctx, req.(*ProbeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Zongzi_Info_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZongziServer).Info(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Zongzi_Info_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZongziServer).Info(ctx, req.(*InfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Zongzi_Members_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MembersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZongziServer).Members(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Zongzi_Members_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZongziServer).Members(ctx, req.(*MembersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Zongzi_Join_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(JoinRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZongziServer).Join(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Zongzi_Join_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZongziServer).Join(ctx, req.(*JoinRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Zongzi_ShardJoin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShardJoinRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZongziServer).ShardJoin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Zongzi_ShardJoin_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZongziServer).ShardJoin(ctx, req.(*ShardJoinRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Zongzi_Apply_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ApplyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZongziServer).Apply(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Zongzi_Apply_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZongziServer).Apply(ctx, req.(*ApplyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Zongzi_Commit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZongziServer).Commit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Zongzi_Commit_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZongziServer).Commit(ctx, req.(*CommitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Zongzi_Read_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ZongziServer).Read(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Zongzi_Read_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ZongziServer).Read(ctx, req.(*ReadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Zongzi_Watch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ZongziServer).Watch(m, &zongziWatchServer{stream})
}

type Zongzi_WatchServer interface {
	Send(*WatchResponse) error
	grpc.ServerStream
}

type zongziWatchServer struct {
	grpc.ServerStream
}

func (x *zongziWatchServer) Send(m *WatchResponse) error {
	return x.ServerStream.SendMsg(m)
}

// Zongzi_ServiceDesc is the grpc.ServiceDesc for Zongzi service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Zongzi_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "zongzi.Zongzi",
	HandlerType: (*ZongziServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _Zongzi_Ping_Handler,
		},
		{
			MethodName: "Probe",
			Handler:    _Zongzi_Probe_Handler,
		},
		{
			MethodName: "Info",
			Handler:    _Zongzi_Info_Handler,
		},
		{
			MethodName: "Members",
			Handler:    _Zongzi_Members_Handler,
		},
		{
			MethodName: "Join",
			Handler:    _Zongzi_Join_Handler,
		},
		{
			MethodName: "ShardJoin",
			Handler:    _Zongzi_ShardJoin_Handler,
		},
		{
			MethodName: "Apply",
			Handler:    _Zongzi_Apply_Handler,
		},
		{
			MethodName: "Commit",
			Handler:    _Zongzi_Commit_Handler,
		},
		{
			MethodName: "Read",
			Handler:    _Zongzi_Read_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Watch",
			Handler:       _Zongzi_Watch_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "zongzi.proto",
}
