package zongzi

import (
	"sync"

	"google.golang.org/grpc"
	"github.com/hashicorp/golang-lru"

	"github.com/logbn/zongzi/internal"
)

type grpcClient struct {
	agent *agent
	conns *lru.Cache
}

func newGrpcClient(a *agent, lruSize) *grpcClient {
	onEvict := func(addr string, conn internal.ZongziClient) {
		conn.Close()
	}
	return &grpcClient{
		agent: a,
		conns: lru.NewWithEvict[string, internal.ZongziClient](lruSize, onEvict),
	}
}

func (c *grpcClient) find(replicaID uint64) (conn internal.ZongziClient, err error) {
	replica, err := c.agent.FindReplica(replicaID)
	if err != nil {
		return
	}
	return c.get(replica.Host.Address)
}

func (c *grpcClient) get(addr string) (conn internal.ZongziClient, err error) {
	conn, ok := c.conns.get(addr)
	if ok {
		return
	}
	var opts []grpc.DialOption
	// https://github.com/grpc/grpc-go/tree/master/examples/features/authentication
	// opts = append(opts, grpc.WithPerRPCCredentials(perRPC))
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return
	}
	c.conns.Add(addr, conn)
	return
}

func (c *grpcClient) remove(addr string) bool {
	return c.conns.Remove(addr)
}

func (c *grpcClient) Close() {
	c.conns.Purge()
}
