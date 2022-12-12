package udp

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

type Client interface {
	Send(d time.Duration, addr string, cmd string, args ...string) (res string, data []string, err error)
	Listen(ctx context.Context, handler HandlerFunc) error
	Close()
}

type HandlerFunc func(cmd string, args ...string) (res []string, err error)

type client struct {
	log         logger.ILogger
	connections map[string]net.PacketConn
	clusterName string
	magicPrefix string
	listenAddr  string
}

func NewClient(magicPrefix, listenAddr, clusterName string) *client {
	return &client{
		log:         logger.GetLogger(magicPrefix),
		connections: map[string]net.PacketConn{},
		magicPrefix: magicPrefix,
		clusterName: strings.ToLower(clusterName),
		listenAddr:  listenAddr,
	}
}

func (c *client) Send(d time.Duration, addr string, cmd string, args ...string) (res string, data []string, err error) {
	raddr, _ := net.ResolveUDPAddr("udp", addr)
	conn, _ := net.ListenPacket("udp", ":0")
	conn.SetDeadline(time.Now().Add(d))
	argStr := strings.Join(args, " ")
	packet := fmt.Sprintf("%s %s %s %s", c.magicPrefix, c.clusterName, cmd, argStr)
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
	magic, clusterName, res, data := parts[0], strings.ToLower(parts[1]), parts[2], parts[3:]
	if magic != c.magicPrefix || clusterName != c.clusterName {
		return "", nil, fmt.Errorf("Invalid response (%s - %s %s) / (%s)", addr, cmd, argStr, msg)
	}
	return
}

func (c *client) Listen(ctx context.Context, handler func(cmd string, args ...string) (res []string, err error)) (err error) {
	if _, ok := c.connections[c.listenAddr]; ok {
		return
	}
	addr, err := net.ResolveUDPAddr("udp4", c.listenAddr)
	if err != nil {
		return
	}
	var conn *net.UDPConn
	if net.ParseIP(addr.IP).IsMulticast() {
		conn, err = net.ListenMulticastUDP("udp4", nil, addr)
		if err != nil {
			return
		}
	} else {
		conn, err = net.ListenUDP("udp4", nil, addr)
		if err != nil {
			return
		}
	}
	c.connections[c.listenAddr] = conn
	var done = make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if conn, ok := c.connections[c.listenAddr]; ok {
				conn.Close()
			}
		case <-done:
		}
		delete(c.connections, c.listenAddr)
	}()
	buf := make([]byte, 4096)
	c.log.Infof("UDP Listening on %s", c.listenAddr)
	for {
		i, dst, err := conn.ReadFromUDP(buf)
		if err != nil {
			close(done)
			return err
		}
		msg := string(buf[:i])
		parts := strings.Split(strings.Trim(msg, "\n"), " ")
		if len(parts) < 3 {
			continue
		}
		magic, clusterName, cmd, args := parts[0], strings.ToLower(parts[1]), parts[2], parts[3:]
		if magic != c.magicPrefix || clusterName != c.clusterName {
			continue
		}
		c.log.Debugf("Received %s %s", cmd, strings.Join(args, " "))
		res, err := handler(cmd, args...)
		if err != nil {
			c.log.Errorf("Error %s %s", strings.Join(res, " "), err.Error())
		} else if res != nil {
			c.log.Infof("Replying %s", strings.Join(res, " "))
		}
		if res == nil {
			continue
		}
		resData := strings.Join(res, " ")
		_, err = conn.WriteTo([]byte(fmt.Sprintf("%s %s %s", c.magicPrefix, c.clusterName, resData)), dst)
		if err != nil {
			close(done)
			return fmt.Errorf("Error replying to %s (%s): %s", cmd, strings.Join(res, " "), err.Error())
		}
	}
}

func (c *client) Close() {
	for k := range c.connections {
		c.connections[k].Close()
	}
}
