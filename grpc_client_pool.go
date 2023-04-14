package zongzi

import (
	"github.com/hashicorp/golang-lru/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/logbn/zongzi/internal"
)

type grpcClientPool struct {
	clients  *lru.Cache[string, grpcClientPoolEntry]
	dialOpts []grpc.DialOption
}

type grpcClientPoolEntry struct {
	client internal.ZongziClient
	conn   *grpc.ClientConn
}

func grpcClientPoolEvictFunc(addr string, e grpcClientPoolEntry) {
	e.conn.Close()
}

func newGrpcClientPool(size int, dialOpts ...grpc.DialOption) *grpcClientPool {
	clients, _ := lru.NewWithEvict[string, grpcClientPoolEntry](size, grpcClientPoolEvictFunc)
	return &grpcClientPool{
		clients:  clients,
		dialOpts: dialOpts,
	}
}

func (c *grpcClientPool) get(addr string) (client internal.ZongziClient) {
	e, ok := c.clients.Get(addr)
	if ok {
		return e.client
	}
	// https://github.com/grpc/grpc-go/tree/master/examples/features/authentication
	// opts = append(opts, grpc.WithPerRPCCredentials(perRPC))
	conn, err := grpc.Dial(addr, append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, c.dialOpts...)...)
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
