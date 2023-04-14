package zongzi

import (
	"context"

	"github.com/logbn/zongzi/internal"
)

type Client struct {
	agent          *Agent
	hostApiAddress string
	hostID         string
}

func newClient(a *Agent, host Host) *Client {
	return &Client{
		agent:          a,
		hostApiAddress: host.ApiAddress,
		hostID:         host.ID,
	}
}

func (c *Client) Ping(ctx context.Context) (err error) {
	if c.hostID == c.agent.HostID() {
		return
	}
	_, err = c.agent.grpcClientPool.get(c.hostApiAddress).Ping(ctx, &internal.PingRequest{})
	return
}

func (c *Client) Apply(ctx context.Context, shardID uint64, cmd []byte) (value uint64, data []byte, err error) {
	var res *internal.ApplyResponse
	if c.hostID == c.agent.HostID() {
		c.agent.log.Debugf(`gRPC Client Apply Local: %s`, string(cmd))
		res, err = c.agent.grpcServer.Apply(ctx, &internal.ApplyRequest{
			ShardId: shardID,
			Data:    cmd,
		})
	} else {
		c.agent.log.Debugf(`gRPC Client Apply Remote: %s`, string(cmd))
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

func (c *Client) Commit(ctx context.Context, shardID uint64, cmd []byte) (err error) {
	if c.hostID == c.agent.HostID() {
		c.agent.log.Debugf(`gRPC Client Commit Local: %s`, string(cmd))
		_, err = c.agent.grpcServer.Commit(ctx, &internal.CommitRequest{
			ShardId: shardID,
			Data:    cmd,
		})
	} else {
		c.agent.log.Debugf(`gRPC Client Commit Remote: %s`, string(cmd))
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

func (c *Client) Query(ctx context.Context, shardID uint64, query []byte, stale ...bool) (value uint64, data []byte, err error) {
	var res *internal.QueryResponse
	if c.hostID == c.agent.HostID() {
		res, err = c.agent.grpcServer.Query(ctx, &internal.QueryRequest{
			ShardId: shardID,
			Stale:   len(stale) > 0 && stale[0],
			Data:    query,
		})
	} else {
		res, err = c.agent.grpcClientPool.get(c.hostApiAddress).Query(ctx, &internal.QueryRequest{
			ShardId: shardID,
			Stale:   len(stale) > 0 && stale[0],
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
