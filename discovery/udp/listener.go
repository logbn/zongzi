package udp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lni/dragonboat/v4/logger"

	"github.com/logbn/zongzi"
	"github.com/logbn/zongzi/udp"
)

type udpListener struct {
	log         logger.ILogger
	agent       zongzi.Agent
	client      udp.Client
	minReplicas int
	peers       map[string]string // gossipAddr: discoveryAddr
	cancel      context.CancelFunc
	mutex       sync.RWMutex
}

func newUDPListener(agent zongzi.Agent, multicastAddr string, minReplicas int) *udpListener {
	return &udpListener{
		agent:       agent,
		log:         logger.GetLogger(magicPrefix + ": disco: udp"),
		client:      udp.NewClient(magicPrefix, multicastAddr, agent.GetClusterName()),
		peers:       map[string]string{},
		minReplicas: minReplicas,
	}
}

func (a *udpListener) Peers() map[string]string {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return mapCopy(a.peers)
}

func (a *udpListener) Start() {
	var ctx context.Context
	ctx, a.cancel = context.WithCancel(context.Background())
	go func() {
		for {
			err := a.client.Listen(ctx, udp.HandlerFunc(a.handle))
			a.log.Errorf("Error reading UDP: %s", err.Error())
			select {
			case <-ctx.Done():
				a.log.Debugf("Stopped Listener")
				return
			case <-time.After(time.Second):
			}
		}
	}()
}

func (a *udpListener) Stop() {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
}

func (a *udpListener) handle(cmd string, args ...string) (res []string, err error) {
	switch cmd {
	// New node wants to join the cluster
	case PROBE:
		if res, err = a.client.Validate(cmd, args, 3); err != nil {
			return
		}
		_, gossipAddr, discoveryAddr := args[0], args[1], args[2]
		switch a.agent.GetStatus() {
		case zongzi.AgentStatus_Active:
			return []string{PROBE_RESPONSE, a.agent.GetHostConfig().Gossip.AdvertiseAddress}, nil
		case zongzi.AgentStatus_Rejoining:
			a.peers[gossipAddr] = discoveryAddr
			if len(a.peers) >= a.minReplicas {
				return []string{PROBE_RESPONSE, a.agent.GetHostConfig().Gossip.AdvertiseAddress}, nil
			}
		default:
			a.peers[gossipAddr] = discoveryAddr
		}

	default:
		err = fmt.Errorf("Unrecognized command: %s %s", cmd, strings.Join(args, " "))
	}
	return
}
