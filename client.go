package zongzi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/lni/dragonboat/v4/logger"
)

type Client interface {
	Propose(ctx context.Context, replicaID uint64, cmd []byte, linear bool) (value uint64, data []byte, err error)
	Query(ctx context.Context, replicaID uint64, query []byte, linear bool) (value uint64, data []byte, err error)
}

type HandlerFunc func(cmd string, args ...string) (res []string, err error)

type client struct {
	agent *agent
}

func newClient(a agent) *client {
	return &client{
		agent: a,
	}
}

func (c *client) Propose(ctx context.Context, replicaID uint64, cmd []byte, linear bool) (value uint64, data []byte, err error) {
	replica, err := c.agent.FindReplica(replicaID)
	if err != nil {
		return
	}
	if replica == nil {
		err = ErrReplicaNotFound
		return
	}
	if replica.ShardID < 1 {
		err = ErrReplicaNotAllowed
		return
	}
	if replica.Status != ReplicaStatus_Active {
		err = ErrReplicaNotActive
		return
	}
	if !linear && !c.agent.hostConfig.NotifyCommit {
		c.agent.log.Warningf(ErrNotifyCommitDisabled)
	}
	var res *internal.Response
	if replica.HostID == c.agent.GetHostID() {
		res, err = c.grpcServer.Propose(ctx, &internal.Request{
			ShardID: replica.ShardID,
			Linear:  linear,
			Data:    cmd,
		})
	} else {
		conn, err := c.agent.grpcClient.get(replica.Host.ApiAddress)
		if err != nil {
			return
		}
		res, err = conn.Propose(ctx, &internal.Request{
			ShardID: replica.ShardID,
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

func (c *client) Query(ctx context.Context, replicaID uint64, query []byte, linear bool) (value uint64, data []byte, err error) {
	replica, err := c.agent.FindReplica(replicaID)
	if err != nil {
		return
	}
	if replica == nil {
		err = ErrReplicaNotFound
		return
	}
	if replica.ShardID < 1 {
		err = ErrReplicaNotAllowed
		return
	}
	if replica.Status != ReplicaStatus_Active {
		err = ErrReplicaNotActive
		return
	}
	var res *internal.Response
	if replica.HostID == c.agent.GetHostID() {
		res, err = c.grpcServer.Query(ctx, &internal.Request{
			ShardID: replica.ShardID,
			Linear:  linear,
			Data:    query,
		})
	} else {
		conn, err := c.agent.grpcClient.get(replica.Host.ApiAddress)
		if err != nil {
			return
		}
		res, err = conn.Query(ctx, &internal.Request{
			ShardID: replica.ShardID,
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
