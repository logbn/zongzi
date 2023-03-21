package zongzi

import (
	"context"

	"github.com/logbn/zongzi/internal"
)

type ReplicaClient interface {
	Ping(ctx context.Context, host Host) (err error)
	Propose(ctx context.Context, replica Replica, cmd []byte, linear bool) (value uint64, data []byte, err error)
	Query(ctx context.Context, replica Replica, query []byte, linear bool) (value uint64, data []byte, err error)
}

type client struct {
	agent *Agent
}

func newClient(a *Agent) *client {
	return &client{
		agent: a,
	}
}

func (c *client) Ping(ctx context.Context, host Host) (err error) {
	if host.ID == c.agent.GetHostID() {
		return
	}
	_, err = c.agent.grpcClientPool.get(host.ApiAddress).Ping(ctx, nil)
	return
}

func (c *client) Propose(ctx context.Context, replica Replica, cmd []byte, linear bool) (value uint64, data []byte, err error) {
	if replica.ShardID < 1 {
		err = ErrReplicaNotAllowed
		return
	}
	if !linear && !c.agent.configHost.NotifyCommit {
		c.agent.log.Warningf(`%v`, ErrNotifyCommitDisabled)
	}
	return c.propose(ctx, replica, cmd, linear)
}

func (c *client) propose(ctx context.Context, replica Replica, cmd []byte, linear bool) (value uint64, data []byte, err error) {
	var res *internal.Response
	if replica.HostID == c.agent.GetHostID() {
		res, err = c.agent.grpcServer.Propose(ctx, &internal.Request{
			ShardId: replica.ShardID,
			Linear:  linear,
			Data:    cmd,
		})
	} else {
		res, err = c.agent.grpcClientPool.get(replica.Host.ApiAddress).Propose(ctx, &internal.Request{
			ShardId: replica.ShardID,
			Linear:  linear,
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

func (c *client) Query(ctx context.Context, replica Replica, query []byte, linear bool) (value uint64, data []byte, err error) {
	if replica.ShardID < 1 {
		err = ErrReplicaNotAllowed
		return
	}
	return c.query(ctx, replica, query, linear)
}

func (c *client) query(ctx context.Context, replica Replica, query []byte, linear bool) (value uint64, data []byte, err error) {
	var res *internal.Response
	if replica.HostID == c.agent.GetHostID() {
		res, err = c.agent.grpcServer.Query(ctx, &internal.Request{
			ShardId: replica.ShardID,
			Linear:  linear,
			Data:    query,
		})
	} else {
		res, err = c.agent.grpcClientPool.get(replica.Host.ApiAddress).Query(ctx, &internal.Request{
			ShardId: replica.ShardID,
			Linear:  linear,
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
