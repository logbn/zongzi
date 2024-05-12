package zongzi

import (
	"context"

	"google.golang.org/grpc"

	"github.com/logbn/zongzi/internal"
)

type grpcClientErr struct {
	err error
}

func (c *grpcClientErr) Ping(ctx context.Context, in *internal.PingRequest, opts ...grpc.CallOption) (*internal.PingResponse, error) {
	return nil, c.err
}

func (c *grpcClientErr) Probe(ctx context.Context, in *internal.ProbeRequest, opts ...grpc.CallOption) (*internal.ProbeResponse, error) {
	return nil, c.err
}

func (c *grpcClientErr) Info(ctx context.Context, in *internal.InfoRequest, opts ...grpc.CallOption) (*internal.InfoResponse, error) {
	return nil, c.err
}

func (c *grpcClientErr) Members(ctx context.Context, in *internal.MembersRequest, opts ...grpc.CallOption) (*internal.MembersResponse, error) {
	return nil, c.err
}

func (c *grpcClientErr) Join(ctx context.Context, in *internal.JoinRequest, opts ...grpc.CallOption) (*internal.JoinResponse, error) {
	return nil, c.err
}

func (c *grpcClientErr) ShardJoin(ctx context.Context, in *internal.ShardJoinRequest, opts ...grpc.CallOption) (*internal.ShardJoinResponse, error) {
	return nil, c.err
}

func (c *grpcClientErr) Commit(ctx context.Context, in *internal.CommitRequest, opts ...grpc.CallOption) (*internal.CommitResponse, error) {
	return nil, c.err
}

func (c *grpcClientErr) Apply(ctx context.Context, in *internal.ApplyRequest, opts ...grpc.CallOption) (*internal.ApplyResponse, error) {
	return nil, c.err
}

func (c *grpcClientErr) Read(ctx context.Context, in *internal.ReadRequest, opts ...grpc.CallOption) (*internal.ReadResponse, error) {
	return nil, c.err
}

func (c *grpcClientErr) Watch(ctx context.Context, in *internal.WatchRequest, opts ...grpc.CallOption) (internal.Internal_WatchClient, error) {
	return nil, c.err
}
