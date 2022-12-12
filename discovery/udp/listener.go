package zongzi

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lni/dragonboat/v4/logger"

	"github.com/logbn/zongzi/udp"
)

type udpListener struct {
	log    logger.ILogger
	agent  *agent
	client udp.Client
	peers  map[string]string // gossipAddr: discoveryAddr
	probes map[string]bool   // gossipAddr: true
	cancel context.CancelFunc
	mutex  sync.RWMutex
}

func newUDPListener(agent *agent, multicastAddr string) *udpListener {
	return &udpListener{
		agent:  agent,
		log:    logger.GetLogger(magicPrefix + ": disco: udp"),
		client: udp.NewClient(magicPrefix, multicastAddr, agent.GetClusterName()),
		peers:  map[string]string{},
		probes: map[string]bool{},
	}
}

func (a *udpListener) Peers() map[string]string {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return strMapCopy(a.peers)
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
	// Some out of band system wants this node to begin the initialization process
	case INIT:
		if a.agent.GetStatus() == AgentStatus_Active {
			return []string{INIT_CONFLICT}, fmt.Errorf("Already initialized")
		}
		if err = a.agent.init(a.Peers()); err != nil {
			return []string{INIT_ERROR}, err
		}
		res = []string{INIT_SUCCESS, a.agent.hostID()}

	// Another node wants this node to participate in cluster initialization
	case INIT_HOST:
		if res, err = a.validate(cmd, args, 1); err != nil {
			return
		}
		seedList := args[0]
		seeds := strings.Split(seedList, ",")
		var replicaID uint64
		for i, gossipAddr := range seeds {
			if gossipAddr == a.agent.GetHostConfig().Gossip.AdvertiseAddress {
				replicaID = uint64(i + 1)
			}
		}
		if replicaID == 0 {
			return
		}
		err = a.agent.startHost(seeds)
		if err != nil {
			return []string{INIT_HOST_ERROR}, err
		}
		res = []string{INIT_HOST_SUCCESS, strconv.Itoa(int(replicaID)), a.agent.hostID()}

	// Another node wants this node to participate in prime shard initialization
	case INIT_SHARD:
		if res, err = a.validate(cmd, args, 1); err != nil {
			return
		}
		var replicaID uint64
		initialMembers := map[uint64]string{}
		err := json.Unmarshal([]byte(args[0]), &initialMembers)
		if err != nil {
			return []string{INIT_SHARD_ERROR}, err
		}
		for i, v := range initialMembers {
			if v == a.agent.hostID() {
				replicaID = i
			}
		}
		if replicaID < 1 {
			return []string{INIT_SHARD_ERROR}, fmt.Errorf("Node not in initial members")
		}
		a.agent.primeConfig.ReplicaID = replicaID
		err = a.agent.startReplica(initialMembers, false, a.agent.primeConfig)
		if err != nil {
			return []string{INIT_SHARD_ERROR}, fmt.Errorf("Failed to start prime shard during init: %w", err)
		}
		a.agent.setStatus(AgentStatus_Active)
		res = []string{INIT_SHARD_SUCCESS, fmt.Sprintf("%d", replicaID)}

	// New node wants to join the cluster
	case PROBE:
		if res, err = a.validate(cmd, args, 3); err != nil {
			return
		}
		nhid, gossipAddr, discoveryAddr := args[0], args[1], args[2]
		switch a.agent.GetStatus() {
		case AgentStatus_Active:
			return []string{PROBE_RESPONSE, a.agent.hostConfig.Gossip.AdvertiseAddress}, nil
		case AgentStatus_Rejoining:
			a.peers[gossipAddr] = discoveryAddr
			if len(a.peers) >= minReplicas {
				return []string{PROBE_RESPONSE, a.agent.hostConfig.Gossip.AdvertiseAddress}, nil
			}
		default:
			a.peers[gossipAddr] = discoveryAddr
		}

	default:
		err = fmt.Errorf("Unrecognized command: %s %s", cmd, strings.Join(args, " "))
	}
	return
}

func (a *udpListener) validate(cmd string, args []string, n int) (res []string, err error) {
	ok := len(args) == n
	if !ok {
		res = []string{fmt.Sprintf("%s_INVALID", cmd)}
		err = fmt.Errorf("%s requires %d arguments: %v", cmd, n, args)
	}
	return
}
