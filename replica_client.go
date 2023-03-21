package zongzi

import (
	"context"

	"github.com/logbn/zongzi/internal"
)

type ReplicaClient struct {
	agent          *Agent
	hostApiAddress string
	hostID         string
	shardID        uint64
}

func newReplicaClient(replica *Replica, a *Agent) *ReplicaClient {
	return &ReplicaClient{
		agent:          a,
		hostApiAddress: replica.Host.ApiAddress,
		hostID:         replica.HostID,
		shardID:        replica.ShardID,
	}
}

func (c *ReplicaClient) Propose(ctx context.Context, cmd []byte, linear bool) (value uint64, data []byte, err error) {
	if !linear && !c.agent.configHost.NotifyCommit {
		c.agent.log.Warningf(`%v`, ErrNotifyCommitDisabled)
	}
	var res *internal.Response
	if c.hostID == c.agent.GetHostID() {
		res, err = c.agent.grpcServer.Propose(ctx, &internal.Request{
			ShardId: c.shardID,
			Linear:  linear,
			Data:    cmd,
		})
	} else {
		res, err = c.agent.grpcClientPool.get(c.hostApiAddress).Propose(ctx, &internal.Request{
			ShardId: c.shardID,
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

func (c *ReplicaClient) Query(ctx context.Context, query []byte, linear bool) (value uint64, data []byte, err error) {
	var res *internal.Response
	if c.hostID == c.agent.GetHostID() {
		res, err = c.agent.grpcServer.Query(ctx, &internal.Request{
			ShardId: c.shardID,
			Linear:  linear,
			Data:    query,
		})
	} else {
		res, err = c.agent.grpcClientPool.get(c.hostApiAddress).Query(ctx, &internal.Request{
			ShardId: c.shardID,
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
