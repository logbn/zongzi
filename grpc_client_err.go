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

func (c *grpcClientErr) Propose(ctx context.Context, in *internal.Request, opts ...grpc.CallOption) (*internal.Response, error) {
	return nil, c.err
}

func (c *grpcClientErr) Query(ctx context.Context, in *internal.Request, opts ...grpc.CallOption) (*internal.Response, error) {
	return nil, c.err
}
