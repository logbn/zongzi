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
	addrs, err := c.getAddrs(replicaID, true)
	if err != nil {
		return
	}
	conn, err := c.agent.grpcClient.get(addrs)
	if err != nil {
		return
	}
	res, err := conn.Propose(ctx, &internal.Request{
		ReplicaID: replicaID,
		Linear:    linear,
		Data:      cmd,
	})
	if err != nil {
		return
	}
	value = res.Value
	data = res.Data
	return
}

func (c *client) Query(ctx context.Context, replicaID uint64, query []byte, linear bool) (value uint64, data []byte, err error) {
	addrs, err := c.getAddrs(replicaID, true)
	if err != nil {
		return
	}
	res, err := c.service(addr).Query(ctx, &internal.Request{
		ReplicaID: replicaID,
		Linear:    linear,
		Data:      query,
	})
	if err != nil {
		return
	}
	data = res.Data
	if len(res.Error) > 0 {
		err = fmt.Errorf(res.Error)
	}
	return
}
