package zongzi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/lni/dragonboat/v4/logger"
)

type UDPClient interface {
	Send(d time.Duration, addr string, cmd string, args ...string) (res string, data []string, err error)
	Listen(ctx context.Context, handler UDPHandlerFunc) error
	Close()
}

type UDPHandlerFunc func(cmd string, args ...string) (level logger.LogLevel, res string, data []string)

type udpClient struct {
	connections map[string]net.PacketConn
	clusterName string
	magicPrefix string
	listenAddr  string
}

func newUDPClient(magicPrefix, listenAddr, clusterName string) *udpClient {
	return &udpClient{
		connections: map[string]net.PacketConn{},
		magicPrefix: magicPrefix,
		clusterName: clusterName,
		listenAddr:  fmt.Sprintf("239.108.0.1:%s", strings.Split(listenAddr, ":")[1]),
	}
}

func (c *udpClient) Send(d time.Duration, addr string, cmd string, args ...string) (res string, data []string, err error) {
	raddr, _ := net.ResolveUDPAddr("udp", addr)
	conn, _ := net.ListenPacket("udp", ":0")
	conn.SetDeadline(time.Now().Add(d))
	argStr := strings.Join(args, " ")
	packet := fmt.Sprintf("%s %s %s %s", c.magicPrefix, cmd, c.clusterName, argStr)
	// log["agent"].Debugf("send(%s) %s", addr, packet)
	conn.WriteTo([]byte(packet), raddr)
	buf := make([]byte, 1024)
	i, _, err := conn.ReadFrom(buf)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			err = nil
		}
		return
	}
	msg := strings.Trim(string(buf[:i]), "\n")
	parts := strings.Split(msg, " ")
	if len(parts) < 3 {
		return "", nil, fmt.Errorf("Invalid response (%s - %s %s) / (%s)", addr, cmd, argStr, msg)
	}
	magic, res, clusterName, data := parts[0], parts[1], parts[2], parts[3:]
	if magic != c.magicPrefix || clusterName != c.clusterName {
		return "", nil, fmt.Errorf("Invalid response (%s - %s %s) / (%s)", addr, cmd, argStr, msg)
	}
	return
}

func (c *udpClient) Listen(ctx context.Context, handler UDPHandlerFunc) (err error) {
	if _, ok := c.connections[c.listenAddr]; ok {
		return
	}
	addr, err := net.ResolveUDPAddr("udp4", c.listenAddr)
	if err != nil {
		return
	}
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return
	}
	// conn.SetReadBuffer(8192)
	c.connections[c.listenAddr] = conn
	var done = make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if c, ok := c.connections[c.listenAddr]; ok {
				c.Close()
			}
			delete(c.connections, c.listenAddr)
		case <-done:
			delete(c.connections, c.listenAddr)
		}
	}()
	buf := make([]byte, 4096)
	for {
		i, dst, err := conn.ReadFromUDP(buf)
		if err != nil {
			close(done)
			return err
		}
		// log["agent"].Debugf("receive")
		msg := string(buf[:i])
		parts := strings.Split(strings.Trim(msg, "\n"), " ")
		if len(parts) < 3 {
			continue
		}
		magic, cmd, clusterName, args := parts[0], parts[1], parts[2], parts[3:]
		if magic != c.magicPrefix || clusterName != c.clusterName {
			continue
		}
		level, res, args := handler(cmd, args...)
		data := strings.Join(args, " ")
		if len(res) > 0 || len(data) > 0 {
			switch level {
			case logger.DEBUG:
				log["agent"].Debugf("%s %s", res, data)
			case logger.INFO:
				log["agent"].Infof("%s %s", res, data)
			case logger.WARNING:
				log["agent"].Warningf("%s %s", res, data)
			case logger.ERROR:
				log["agent"].Errorf("%s %s", res, data)
			case logger.CRITICAL:
				log["agent"].Panicf("%s %s", res, data)
			}
		}
		if len(res) > 0 {
			_, err := conn.WriteTo([]byte(fmt.Sprintf("%s %s %s %s", c.magicPrefix, res, c.clusterName, data)), dst)
			if err != nil {
				close(done)
				return fmt.Errorf("Error replying to %s (%s): %s", cmd, res, err.Error())
			}
		}
	}
}

func (c *udpClient) Close() {
	for k := range c.connections {
		c.connections[k].Close()
	}
}
