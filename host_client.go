package zongzi

import (
	"context"
	"time"

	"github.com/benbjohnson/clock"
	"google.golang.org/grpc"

	"github.com/logbn/zongzi/internal"
)

type HostClient interface {
	Ping(ctx context.Context) (t time.Duration, err error)
	Apply(ctx context.Context, shardID uint64, cmd []byte) (value uint64, data []byte, err error)
	Commit(ctx context.Context, shardID uint64, cmd []byte) (err error)
	Read(ctx context.Context, shardID uint64, query []byte, stale bool) (value uint64, data []byte, err error)
	Watch(ctx context.Context, shardID uint64, query []byte, results chan<- *Result, stale bool) (err error)
}

type hostclient struct {
	agent          *Agent
	clock          clock.Clock
	hostApiAddress string
	hostID         string
}

func newHostClient(a *Agent, host Host) *hostclient {
	return &hostclient{
		agent:          a,
		clock:          clock.New(),
		hostApiAddress: host.ApiAddress,
		hostID:         host.ID,
	}
}

func (c *hostclient) Ping(ctx context.Context) (t time.Duration, err error) {
	if c.hostID == c.agent.HostID() {
		return
	}
	start := c.clock.Now()
	_, err = c.agent.grpcClientPool.get(c.hostApiAddress).Ping(ctx, &internal.PingRequest{})
	t = c.clock.Since(start)
	return
}

func (c *hostclient) Apply(ctx context.Context, shardID uint64, cmd []byte) (value uint64, data []byte, err error) {
	var res *internal.ApplyResponse
	if c.hostID == c.agent.HostID() {
		c.agent.log.Debugf(`gRPC HostClient Apply Local: %s`, string(cmd))
		res, err = c.agent.grpcServer.Apply(ctx, &internal.ApplyRequest{
			ShardId: shardID,
			Data:    cmd,
		})
	} else {
		c.agent.log.Debugf(`gRPC HostClient Apply Remote: %s`, string(cmd))
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

func (c *hostclient) Commit(ctx context.Context, shardID uint64, cmd []byte) (err error) {
	if c.hostID == c.agent.HostID() {
		c.agent.log.Debugf(`gRPC HostClient Commit Local: %s`, string(cmd))
		_, err = c.agent.grpcServer.Commit(ctx, &internal.CommitRequest{
			ShardId: shardID,
			Data:    cmd,
		})
	} else {
		c.agent.log.Debugf(`gRPC HostClient Commit Remote: %s`, string(cmd))
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

func (c *hostclient) Read(ctx context.Context, shardID uint64, query []byte, stale bool) (value uint64, data []byte, err error) {
	var res *internal.ReadResponse
	if c.hostID == c.agent.HostID() {
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

func (c *hostclient) Watch(ctx context.Context, shardID uint64, query []byte, results chan<- *Result, stale bool) (err error) {
	var client internal.Internal_WatchClient
	if c.hostID == c.agent.HostID() {
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
