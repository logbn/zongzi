package udp

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
	Send(d time.Duration, addr string, cmd string, args ...string) (res string, data []string, err error)
	Listen(ctx context.Context, handler HandlerFunc) error
	Validate(cmd string, args []string, n int) (res []string, err error)
}

type HandlerFunc func(cmd string, args ...string) (res []string, err error)

type client struct {
	log         logger.ILogger
	clusterName string
	magicPrefix string
	listenAddr  string
	secrets     []string
}

func NewClient(log logger.ILogger, magicPrefix, listenAddr, clusterName string, secrets []string) *client {
	return &client{
		log:         log,
		magicPrefix: magicPrefix,
		clusterName: strings.ToLower(clusterName),
		listenAddr:  listenAddr,
		secrets:     secrets,
	}
}

func (c *client) Send(d time.Duration, addr string, cmd string, args ...string) (res string, data []string, err error) {
	raddr, _ := net.ResolveUDPAddr("udp", addr)
	conn, _ := net.ListenPacket("udp", ":0")
	conn.SetDeadline(time.Now().Add(d))
	argStr := strings.Join(args, " ")
	packet := fmt.Sprintf("%s %s %s %s %s", c.magicPrefix, c.clusterName, c.sig(cmd, argStr), cmd, argStr)
	conn.WriteTo([]byte(packet), raddr)
	var buf = make([]byte, 1024)
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

func (c *client) Listen(ctx context.Context, handler HandlerFunc) (err error) {
	addr, err := net.ResolveUDPAddr("udp4", c.listenAddr)
	if err != nil {
		return
	}
	var conn *net.UDPConn
	if addr.IP.IsMulticast() {
		conn, err = net.ListenMulticastUDP("udp4", nil, addr)
	} else {
		conn, err = net.ListenUDP("udp4", addr)
	}
	if err != nil {
		return
	}
	defer conn.Close()
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		}
	}()
	c.log.Infof("UDP Listening on %s", c.listenAddr)
	var buf = make([]byte, 4096)
	var i int
	var dst *net.UDPAddr
	var res []string
	for {
		i, dst, err = conn.ReadFromUDP(buf)
		if err != nil {
			err = fmt.Errorf("Error reading from UDP: %w", err)
			break
		}
		msg := string(buf[:i])
		parts := strings.Split(strings.Trim(msg, "\n"), " ")
		if len(parts) < 4 {
			c.log.Warningf("Not enough arguments for %s", msg)
			continue
		}
		magic, clusterName, sig, cmd, args := parts[0], strings.ToLower(parts[1]), parts[2], parts[3], parts[4:]
		if magic != c.magicPrefix || clusterName != c.clusterName {
			continue
		}
		argStr := strings.Join(args, " ")
		if !c.sigVerify(sig, cmd, argStr) {
			c.log.Warningf("Ignoring signature mismatch %s %s %s", sig, cmd, argStr)
			continue
		}
		c.log.Debugf("Received %s %s", cmd, argStr)
		res, err = handler(cmd, args...)
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
			err = fmt.Errorf("Error replying to %s (%s): %w", cmd, strings.Join(res, " "), err)
			break
		}
	}
	return
}

func (c *client) Validate(cmd string, args []string, n int) (res []string, err error) {
	ok := len(args) == n
	if !ok {
		res = []string{fmt.Sprintf("%s_INVALID", cmd)}
		err = fmt.Errorf("%s requires %d arguments: %v", cmd, n, args)
	}
	return
}

func (c *client) sig(cmd, args string) string {
	if len(c.secrets) < 1 {
		return "-"
	}
	mac := hmac.New(sha256.New, []byte(c.secrets[0]))
	mac.Write([]byte(fmt.Sprintf("%s %s", cmd, args)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (c *client) sigVerify(sig, cmd, args string) bool {
	if len(c.secrets) < 1 && sig == "-" {
		return true
	}
	b, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	for _, s := range c.secrets {
		mac := hmac.New(sha256.New, []byte(s))
		mac.Write([]byte(fmt.Sprintf("%s %s", cmd, args)))
		if hmac.Equal(b, mac.Sum(nil)) {
			return true
		}
	}
	return false
}
