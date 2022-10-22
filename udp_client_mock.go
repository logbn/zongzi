package zongzi

import (
	"context"
	"net"
	"time"

	"github.com/lni/dragonboat/v4/logger"
)

type udpClientMock struct {
	log         logger.ILogger
	connections map[string]net.PacketConn
	clusterName string
	magicPrefix string
	listenAddr  string
}

func newUDPClientMock(magicPrefix, listenAddr, clusterName string) *udpClientMock {
	return &udpClientMock{
		connections: map[string]net.PacketConn{},
		magicPrefix: magicPrefix,
		clusterName: clusterName,
		listenAddr:  listenAddr,
	}
}

func (c *udpClientMock) Send(d time.Duration, addr string, cmd string, args ...string) (res string, data []string, err error) {
	return
}

func (c *udpClientMock) Listen(ctx context.Context, handler UDPHandlerFunc) (err error) {
	return
}

func (c *udpClientMock) Close() {
	return
}
