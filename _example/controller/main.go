package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/logbn/zongzi"
	"github.com/logbn/zongzi/discovery/udp"
)

var (
	name        = flag.String("n", "test001", "Cluster name (base36 maxlen 12)")
	dataDir     = flag.String("d", "/var/lib/zongzi/node", "Base data directory")
	raftAddr    = flag.String("r", "127.0.0.1:10801", "Raft address")
	gossipAddr  = flag.String("g", "127.0.0.1:10802", "Gossip address")
	discoListen = flag.String("dl", "239.108.0.1:10803", "Multicast address")
	discoMulti  = flag.String("dm", "239.108.0.1:10803", "UDP multicast discovery addresses (csv)")
	zone        = flag.String("z", "us-west-1a", "Zone")
	secret      = flag.String("s", "", "Shared secrets (csv)")
	shardType   = "banana"
)

func main() {
	flag.Parse()
	zongzi.SetLogLevelSaneDebug()
	meta, _ := json.Marshal(map[string]any{"zone": *zone})
	c := &controller{}
	agent, err := zongzi.NewAgent(zongzi.AgentConfig{
		ClusterName: *name,
		Discovery: udp.NewOracle(udp.Config{
			Multicast:       strings.Split(*discoMulti, ","),
			MulticastListen: *discoListen,
			Secrets:         strings.Split(*secret, ","),
		}),
		HostConfig: zongzi.HostConfig{
			WALDir:      *dataDir + "/wal",
			NodeHostDir: *dataDir + "/raft",
			RaftAddress: *raftAddr,
			Gossip: zongzi.GossipConfig{
				BindAddress:      fmt.Sprintf("0.0.0.0:%s", strings.Split(*gossipAddr, ":")[1]),
				AdvertiseAddress: *gossipAddr,
				Meta:             meta,
			},
			RaftEventListener: c,
			RTTMillisecond:    10,
		},
		Secrets: strings.Split(*secret, ","),
	})
	if err != nil {
		panic(err)
	}
	agent.RegisterShardType(shardType, bananaFactory(), nil)
	c.agent = agent
	if err = agent.Start(); err != nil {
		panic(err)
	}

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop
	agent.Stop()
}
