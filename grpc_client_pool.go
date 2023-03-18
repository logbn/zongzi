package zongzi

import (
	"sync"

	"github.com/hashicorp/golang-lru/v2"
	"google.golang.org/grpc"

	"github.com/logbn/zongzi/internal"
)

type grpcClientPool struct {
	conns   *lru.Cache
	secrets []string
}

func newGrpcClientPool(lruSize int, secrets []string) *grpcClientPool {
	onEvict := func(addr string, conn internal.ZongziClient) {
		conn.Close()
	}
	return &grpcClient{
		conns:   lru.NewWithEvict[string, internal.ZongziClient](lruSize, onEvict),
		secrets: secrets,
	}
}

func (c *grpcClientPool) get(addr string) (conn internal.ZongziClient, err error) {
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

func (c *grpcClientPool) remove(addr string) bool {
	return c.conns.Remove(addr)
}

func (c *grpcClientPool) Close() {
	c.conns.Purge()
}
