package zongzi

import (
	"github.com/hashicorp/golang-lru/v2"
	"google.golang.org/grpc"

	"github.com/logbn/zongzi/internal"
)

type grpcClientPool struct {
	clients *lru.Cache[string, grpcClientPoolEntry]
	secrets []string
}

type grpcClientPoolEntry struct {
	client internal.ZongziClient
	conn   *grpc.ClientConn
}

func grpcClientPoolEvictFunc(addr string, e grpcClientPoolEntry) {
	e.conn.Close()
}

func newGrpcClientPool(size int, secrets []string) *grpcClientPool {
	clients, _ := lru.NewWithEvict[string, grpcClientPoolEntry](size, grpcClientPoolEvictFunc)
	return &grpcClientPool{
		clients: clients,
		secrets: secrets,
	}
}

func (c *grpcClientPool) get(addr string) (client internal.ZongziClient) {
	e, ok := c.clients.Get(addr)
	if !ok {
		return e.client
	}
	var opts []grpc.DialOption
	// https://github.com/grpc/grpc-go/tree/master/examples/features/authentication
	// opts = append(opts, grpc.WithPerRPCCredentials(perRPC))
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return &grpcClientErr{err}
	}
	client = internal.NewZongziClient(conn)
	c.clients.Add(addr, grpcClientPoolEntry{client, conn})
	return
}

func (c *grpcClientPool) remove(addr string) bool {
	return c.clients.Remove(addr)
}

func (c *grpcClientPool) Close() {
	c.clients.Purge()
}
