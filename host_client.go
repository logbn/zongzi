package zongzi

import (
	"context"
	"time"

	"github.com/benbjohnson/clock"
	"google.golang.org/grpc"

	"github.com/logbn/zongzi/internal"
)

type hostClient struct {
	agent          *Agent
	clock          clock.Clock
	hostApiAddress string
	hostID         string
}

func newhostClient(a *Agent, host Host) hostClient {
	return hostClient{
		agent:          a,
		clock:          clock.New(),
		hostApiAddress: host.ApiAddress,
		hostID:         host.ID,
	}
}

func (c *hostClient) Ping(ctx context.Context) (t time.Duration, err error) {
	if c.hostID == c.agent.hostID() {
		return
	}
	start := c.clock.Now()
	_, err = c.agent.grpcClientPool.get(c.hostApiAddress).Ping(ctx, &internal.PingRequest{})
	t = c.clock.Since(start)
	return
}

func (c *hostClient) ReadIndex(ctx context.Context, shardID uint64) (err error) {
	if c.hostID == c.agent.hostID() {
		return
	}
	_, err = c.agent.grpcClientPool.get(c.hostApiAddress).Index(ctx, &internal.IndexRequest{ShardId: shardID})
	return
}

func (c *hostClient) Apply(ctx context.Context, shardID uint64, cmd []byte) (value uint64, data []byte, err error) {
	var res *internal.ApplyResponse
	if c.hostID == c.agent.hostID() {
		c.agent.log.Debugf(`gRPC hostClient Apply Local: %s`, string(cmd))
		res, err = c.agent.grpcServer.Apply(ctx, &internal.ApplyRequest{
			ShardId: shardID,
			Data:    cmd,
		})
	} else {
		c.agent.log.Debugf(`gRPC hostClient Apply Remote: %s`, string(cmd))
		res, err = c.agent.grpcClientPool.get(c.hostApiAddress).Apply(ctx, &internal.ApplyRequest{
			ShardId: shardID,
			Data:    cmd,
		})
	}
	if err != nil {
		return
	}
	value = res.Value
	data = res.Data
	return
}

func (c *hostClient) Commit(ctx context.Context, shardID uint64, cmd []byte) (err error) {
	if c.hostID == c.agent.hostID() {
		c.agent.log.Debugf(`gRPC hostClient Commit Local: %s`, string(cmd))
		_, err = c.agent.grpcServer.Commit(ctx, &internal.CommitRequest{
			ShardId: shardID,
			Data:    cmd,
		})
	} else {
		c.agent.log.Debugf(`gRPC hostClient Commit Remote: %s`, string(cmd))
		_, err = c.agent.grpcClientPool.get(c.hostApiAddress).Commit(ctx, &internal.CommitRequest{
			ShardId: shardID,
			Data:    cmd,
		})
	}
	if err != nil {
		return
	}
	return
}

func (c *hostClient) Read(ctx context.Context, shardID uint64, query []byte, stale bool) (value uint64, data []byte, err error) {
	var res *internal.ReadResponse
	if c.hostID == c.agent.hostID() {
		res, err = c.agent.grpcServer.Read(ctx, &internal.ReadRequest{
			ShardId: shardID,
			Stale:   stale,
			Data:    query,
		})
	} else {
		res, err = c.agent.grpcClientPool.get(c.hostApiAddress).Read(ctx, &internal.ReadRequest{
			ShardId: shardID,
			Stale:   stale,
			Data:    query,
		})
	}
	if err != nil {
		return
	}
	value = res.Value
	data = res.Data
	return
}

type watchServer struct {
	grpc.ServerStream

	ctx     context.Context
	results chan<- *Result
}

func newWatchServer(ctx context.Context, results chan<- *Result) *watchServer {
	return &watchServer{
		ctx:     ctx,
		results: results,
	}
}

func (s *watchServer) Context() context.Context {
	return s.ctx
}

func (s *watchServer) Send(res *internal.WatchResponse) error {
	s.results <- &Result{
		Value: res.Value,
		Data:  res.Data,
	}
	return nil
}

func (c *hostClient) Watch(ctx context.Context, shardID uint64, query []byte, results chan<- *Result, stale bool) (err error) {
	var client internal.Internal_WatchClient
	if c.hostID == c.agent.hostID() {
		err = c.agent.grpcServer.Watch(&internal.WatchRequest{
			ShardId: shardID,
			Stale:   stale,
			Data:    query,
		}, newWatchServer(ctx, results))
	} else {
		client, err = c.agent.grpcClientPool.get(c.hostApiAddress).Watch(ctx, &internal.WatchRequest{
			ShardId: shardID,
			Stale:   stale,
			Data:    query,
		})
		for {
			res, err := client.Recv()
			if err != nil {
				break
			}
			results <- &Result{
				Value: res.Value,
				Data:  res.Data,
			}
		}
	}
	if err != nil {
		return
	}
	return
}
