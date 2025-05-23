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
	Internal_Ping_FullMethodName    = "/zongzi.Internal/Ping"
	Internal_Probe_FullMethodName   = "/zongzi.Internal/Probe"
	Internal_Info_FullMethodName    = "/zongzi.Internal/Info"
	Internal_Members_FullMethodName = "/zongzi.Internal/Members"
	Internal_Join_FullMethodName    = "/zongzi.Internal/Join"
	Internal_Add_FullMethodName     = "/zongzi.Internal/Add"
	Internal_Index_FullMethodName   = "/zongzi.Internal/Index"
	Internal_Apply_FullMethodName   = "/zongzi.Internal/Apply"
	Internal_Commit_FullMethodName  = "/zongzi.Internal/Commit"
	Internal_Read_FullMethodName    = "/zongzi.Internal/Read"
	Internal_Watch_FullMethodName   = "/zongzi.Internal/Watch"
)

// InternalClient is the client API for Internal service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type InternalClient interface {
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
	// Add takes a replica ID, shard ID and host ID and returns success
	// Used during replica creation to request replica's addition to the shard
	Add(ctx context.Context, in *AddRequest, opts ...grpc.CallOption) (*AddResponse, error)
	// Index reads the index of a shard
	Index(ctx context.Context, in *IndexRequest, opts ...grpc.CallOption) (*IndexResponse, error)
	// Apply provides unary request/response command forwarding
	Apply(ctx context.Context, in *ApplyRequest, opts ...grpc.CallOption) (*ApplyResponse, error)
	// Commit provides unary request/response command forwarding
	Commit(ctx context.Context, in *CommitRequest, opts ...grpc.CallOption) (*CommitResponse, error)
	// Read provides unary request/response query forwarding
	Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (*ReadResponse, error)
	// Watch provides streaming query response forwarding
	Watch(ctx context.Context, in *WatchRequest, opts ...grpc.CallOption) (Internal_WatchClient, error)
}

type internalClient struct {
	cc grpc.ClientConnInterface
}

func NewInternalClient(cc grpc.ClientConnInterface) InternalClient {
	return &internalClient{cc}
}

func (c *internalClient) Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error) {
	out := new(PingResponse)
	err := c.cc.Invoke(ctx, Internal_Ping_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Probe(ctx context.Context, in *ProbeRequest, opts ...grpc.CallOption) (*ProbeResponse, error) {
	out := new(ProbeResponse)
	err := c.cc.Invoke(ctx, Internal_Probe_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoResponse, error) {
	out := new(InfoResponse)
	err := c.cc.Invoke(ctx, Internal_Info_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Members(ctx context.Context, in *MembersRequest, opts ...grpc.CallOption) (*MembersResponse, error) {
	out := new(MembersResponse)
	err := c.cc.Invoke(ctx, Internal_Members_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Join(ctx context.Context, in *JoinRequest, opts ...grpc.CallOption) (*JoinResponse, error) {
	out := new(JoinResponse)
	err := c.cc.Invoke(ctx, Internal_Join_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Add(ctx context.Context, in *AddRequest, opts ...grpc.CallOption) (*AddResponse, error) {
	out := new(AddResponse)
	err := c.cc.Invoke(ctx, Internal_Add_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Index(ctx context.Context, in *IndexRequest, opts ...grpc.CallOption) (*IndexResponse, error) {
	out := new(IndexResponse)
	err := c.cc.Invoke(ctx, Internal_Index_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Apply(ctx context.Context, in *ApplyRequest, opts ...grpc.CallOption) (*ApplyResponse, error) {
	out := new(ApplyResponse)
	err := c.cc.Invoke(ctx, Internal_Apply_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Commit(ctx context.Context, in *CommitRequest, opts ...grpc.CallOption) (*CommitResponse, error) {
	out := new(CommitResponse)
	err := c.cc.Invoke(ctx, Internal_Commit_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (*ReadResponse, error) {
	out := new(ReadResponse)
	err := c.cc.Invoke(ctx, Internal_Read_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *internalClient) Watch(ctx context.Context, in *WatchRequest, opts ...grpc.CallOption) (Internal_WatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &Internal_ServiceDesc.Streams[0], Internal_Watch_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &internalWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Internal_WatchClient interface {
	Recv() (*WatchResponse, error)
	grpc.ClientStream
}

type internalWatchClient struct {
	grpc.ClientStream
}

func (x *internalWatchClient) Recv() (*WatchResponse, error) {
	m := new(WatchResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// InternalServer is the server API for Internal service.
// All implementations must embed UnimplementedInternalServer
// for forward compatibility
type InternalServer interface {
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
	// Add takes a replica ID, shard ID and host ID and returns success
	// Used during replica creation to request replica's addition to the shard
	Add(context.Context, *AddRequest) (*AddResponse, error)
	// Index reads the index of a shard
	Index(context.Context, *IndexRequest) (*IndexResponse, error)
	// Apply provides unary request/response command forwarding
	Apply(context.Context, *ApplyRequest) (*ApplyResponse, error)
	// Commit provides unary request/response command forwarding
	Commit(context.Context, *CommitRequest) (*CommitResponse, error)
	// Read provides unary request/response query forwarding
	Read(context.Context, *ReadRequest) (*ReadResponse, error)
	// Watch provides streaming query response forwarding
	Watch(*WatchRequest, Internal_WatchServer) error
	mustEmbedUnimplementedInternalServer()
}

// UnimplementedInternalServer must be embedded to have forward compatible implementations.
type UnimplementedInternalServer struct {
}

func (UnimplementedInternalServer) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedInternalServer) Probe(context.Context, *ProbeRequest) (*ProbeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Probe not implemented")
}
func (UnimplementedInternalServer) Info(context.Context, *InfoRequest) (*InfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Info not implemented")
}
func (UnimplementedInternalServer) Members(context.Context, *MembersRequest) (*MembersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Members not implemented")
}
func (UnimplementedInternalServer) Join(context.Context, *JoinRequest) (*JoinResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Join not implemented")
}
func (UnimplementedInternalServer) Add(context.Context, *AddRequest) (*AddResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Add not implemented")
}
func (UnimplementedInternalServer) Index(context.Context, *IndexRequest) (*IndexResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Index not implemented")
}
func (UnimplementedInternalServer) Apply(context.Context, *ApplyRequest) (*ApplyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Apply not implemented")
}
func (UnimplementedInternalServer) Commit(context.Context, *CommitRequest) (*CommitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Commit not implemented")
}
func (UnimplementedInternalServer) Read(context.Context, *ReadRequest) (*ReadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Read not implemented")
}
func (UnimplementedInternalServer) Watch(*WatchRequest, Internal_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}
func (UnimplementedInternalServer) mustEmbedUnimplementedInternalServer() {}

// UnsafeInternalServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to InternalServer will
// result in compilation errors.
type UnsafeInternalServer interface {
	mustEmbedUnimplementedInternalServer()
}

func RegisterInternalServer(s grpc.ServiceRegistrar, srv InternalServer) {
	s.RegisterService(&Internal_ServiceDesc, srv)
}

func _Internal_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Ping_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Ping(ctx, req.(*PingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Probe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProbeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Probe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Probe_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Probe(ctx, req.(*ProbeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Info_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Info(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Info_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Info(ctx, req.(*InfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Members_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MembersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Members(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Members_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Members(ctx, req.(*MembersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Join_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(JoinRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Join(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Join_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Join(ctx, req.(*JoinRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Add_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Add(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Add_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Add(ctx, req.(*AddRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Index_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(IndexRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Index(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Index_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Index(ctx, req.(*IndexRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Apply_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ApplyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Apply(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Apply_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Apply(ctx, req.(*ApplyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Commit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Commit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Commit_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Commit(ctx, req.(*CommitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Read_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InternalServer).Read(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Internal_Read_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InternalServer).Read(ctx, req.(*ReadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Internal_Watch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(InternalServer).Watch(m, &internalWatchServer{stream})
}

type Internal_WatchServer interface {
	Send(*WatchResponse) error
	grpc.ServerStream
}

type internalWatchServer struct {
	grpc.ServerStream
}

func (x *internalWatchServer) Send(m *WatchResponse) error {
	return x.ServerStream.SendMsg(m)
}

// Internal_ServiceDesc is the grpc.ServiceDesc for Internal service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Internal_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "zongzi.Internal",
	HandlerType: (*InternalServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _Internal_Ping_Handler,
		},
		{
			MethodName: "Probe",
			Handler:    _Internal_Probe_Handler,
		},
		{
			MethodName: "Info",
			Handler:    _Internal_Info_Handler,
		},
		{
			MethodName: "Members",
			Handler:    _Internal_Members_Handler,
		},
		{
			MethodName: "Join",
			Handler:    _Internal_Join_Handler,
		},
		{
			MethodName: "Add",
			Handler:    _Internal_Add_Handler,
		},
		{
			MethodName: "Index",
			Handler:    _Internal_Index_Handler,
		},
		{
			MethodName: "Apply",
			Handler:    _Internal_Apply_Handler,
		},
		{
			MethodName: "Commit",
			Handler:    _Internal_Commit_Handler,
		},
		{
			MethodName: "Read",
			Handler:    _Internal_Read_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Watch",
			Handler:       _Internal_Watch_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "zongzi.proto",
}
