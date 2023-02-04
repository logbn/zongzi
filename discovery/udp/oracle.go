package udp

import (
	"strings"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/lni/dragonboat/v4/logger"

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
	PROBE            = "PROBE"
	PROBE_JOIN       = "PROBE_JOIN"
	PROBE_JOIN_HOST  = "PROBE_JOIN_HOST"
	PROBE_JOIN_SHARD = "PROBE_JOIN_SHARD"
	PROBE_JOIN_ERROR = "PROBE_JOIN_ERROR"
	PROBE_RESPONSE   = "PROBE_RESPONSE"

	PROBE_REJOIN      = "PROBE_REJOIN"
	PROBE_REJOIN_PEER = "PROBE_REJOIN_PEER"
)

type Config struct {
	Multicast       []string
	MulticastListen string
	MinReplicas     int
	Secrets         []string
}

type oracle struct {
	cfg      Config
	log      logger.ILogger
	client   udp.Client
	listener *udpListener
	clock    clock.Clock
}

func NewOracle(cfg Config) *oracle {
	return &oracle{
		cfg:   cfg,
		log:   logger.GetLogger(magicPrefix),
		clock: clock.New(),
	}
}

func (o *oracle) GetSeedList(agent zongzi.Agent) (seeds []string, err error) {
	var (
		hostID     = agent.GetHostID()
		gossipAddr = agent.GetHostConfig().Gossip.AdvertiseAddress
		raftAddr   = agent.GetHostConfig().RaftAddress
		client     = udp.NewClient(o.log, magicPrefix, o.cfg.MulticastListen, agent.GetClusterName(), o.cfg.Secrets)
	)
	minReplicas := o.cfg.MinReplicas
	if minReplicas < 1 {
		minReplicas = 3
	}
	o.listener = newUDPListener(o.log, agent, o.cfg.MulticastListen, minReplicas, o.cfg.Secrets)
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
		o.clock.Sleep(time.Second)
	}
	return
}

func (o *oracle) Peers() map[string]string {
	return o.listener.Peers()
}
