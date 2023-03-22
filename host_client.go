package zongzi

import (
	"context"

	"github.com/logbn/zongzi/internal"
)

type HostClient struct {
	agent          *Agent
	hostApiAddress string
	hostID         string
	shardID        uint64
}

func newHostClient(host *Host, a *Agent) *HostClient {
	return &HostClient{
		agent:          a,
		hostApiAddress: host.ApiAddress,
		hostID:         host.ID,
	}
}

func (c *HostClient) Ping(ctx context.Context) (err error) {
	if c.hostID == c.agent.GetHostID() {
		return
	}
	_, err = c.agent.grpcClientPool.get(c.hostApiAddress).Ping(ctx, &internal.PingRequest{})
	return
}
