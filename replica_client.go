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

func newReplicaClient(replica Replica, host Host, a *Agent) *ReplicaClient {
	return &ReplicaClient{
		agent:          a,
		hostApiAddress: host.ApiAddress,
		hostID:         replica.HostID,
		shardID:        replica.ShardID,
	}
}

// Propose a command to a specific replica.
//
// # cmd
//
// The command to run.
//
// # linear
//
// Indicates whether to return early after command is committed rather than waiting for it to be applied.
// NotifyCommit must be set to true in HostConfig in order for non-linear writes to work.
//
//	false - if you want blind high-volume writes. Returned value and data will always be empty.
//	true  - if you need a response.
func (c *ReplicaClient) Propose(ctx context.Context, cmd []byte, linear bool) (value uint64, data []byte, err error) {
	var res *internal.Response
	if c.hostID == c.agent.HostID() {
		c.agent.log.Debugf(`gRPC Client Propose Local: %s, %v`, string(cmd), linear)
		res, err = c.agent.grpcServer.Propose(ctx, &internal.Request{
			ShardId: c.shardID,
			Linear:  linear,
			Data:    cmd,
		})
	} else {
		c.agent.log.Debugf(`gRPC Client Propose Remote: %s, %v`, string(cmd), linear)
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

// Query a specific replica.
//
// # query
//
// The query forwarded to the Lookup method of the replica's state machine.
//
// # linear
//
// Indicates whether to read index before performing the query.
//
//	false - if you want low latency reads with eventual consistency.
//	true  - if you are willing to pay the extra latency for a highly consistent read.
func (c *ReplicaClient) Query(ctx context.Context, query []byte, linear bool) (value uint64, data []byte, err error) {
	var res *internal.Response
	if c.hostID == c.agent.HostID() {
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
