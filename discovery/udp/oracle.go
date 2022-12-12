package zongzi

import (
	"strings"
	"time"

	"github.com/logbn/zongzi"
	"github.com/logbn/zongzi/udp"
)

const (
	magicPrefix  = "zongzi"
	probeTimeout = 3 * time.Second
	probePause   = 5 * time.Second
	joinTimeout  = 5 * time.Second
)

const (
	PROBE_JOIN       = "PROBE_JOIN"
	PROBE_JOIN_HOST  = "PROBE_JOIN_HOST"
	PROBE_JOIN_SHARD = "PROBE_JOIN_SHARD"
	PROBE_JOIN_ERROR = "PROBE_JOIN_ERROR"

	PROBE_REJOIN      = "PROBE_REJOIN"
	PROBE_REJOIN_PEER = "PROBE_REJOIN_PEER"

	INIT               = "INIT"
	INIT_ERROR         = "INIT_ERROR"
	INIT_CONFLICT      = "INIT_CONFLICT"
	INIT_SUCCESS       = "INIT_SUCCESS"
	INIT_HOST          = "INIT_HOST"
	INIT_HOST_ERROR    = "INIT_HOST_ERROR"
	INIT_HOST_SUCCESS  = "INIT_HOST_SUCCESS"
	INIT_SHARD         = "INIT_SHARD"
	INIT_SHARD_ERROR   = "INIT_SHARD_ERROR"
	INIT_SHARD_SUCCESS = "INIT_SHARD_SUCCESS"
)

type oracle struct {
	cfg      Config
	client   udp.Client
	listener *udpListener
	peers    map[string]string // gossipAddr: discoveryAddr
	probes   map[string]bool   // gossipAddr: true
	seedChan chan []string
}

type Config struct {
	Multicast       []string
	MulticastListen string
	Secret          string
}

func NewOracle(cfg Config) *oracle {
	return &oracle{
		cfg:      cfg,
		peers:    map[string]string{},
		probes:   map[string]bool{},
		seedChan: make(chan []string),
	}
}

func (o *oracle) GetSeedList(agent zongzi.Agent) (seeds []string, err error) {
	var (
		hostID     = agent.hostID()
		gossipAddr = agent.GetHostConfig().Gossip.AdvertiseAddress
		raftAddr   = agent.GetHostConfig().RaftAddress
		client     = udp.NewClient(magicPrefix, o.cfg.MulticastListen, agent.GetClusterName())
	)
	o.listener = newUDPListener(a, o.cfg.MulticastListen)
	o.listener.Start()
	for {
		for _, addr := range o.cfg.Multicast {
			res, args, err := client.Send(time.Second, addr, PROBE, hostID, gossipAddr, raftAddr)
			if err != nil {
				continue
			}
			if res == PROBE_RESPONSE && len(args) == 1 {
				seeds = append(seeds, strings.Split(args[0], ",")...)
			}
		}
		if len(seeds) > 0 {
			break
		}
		a.clock.Sleep(time.Second)
	}
	return
}
