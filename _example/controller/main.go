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
	name       = flag.String("n", "test001", "Cluster name (base36 maxlen 12)")
	dataDir    = flag.String("d", "/var/lib/zongzi/node", "Base data directory")
	raftAddr   = flag.String("r", "127.0.0.1:10801", "Raft address")
	gossipAddr = flag.String("g", "127.0.0.1:10802", "Gossip address")
	discovery  = flag.String("u", "239.108.0.1:10801", "UDP multicast discovery addresses (csv)")
	zone       = flag.String("z", "us-west-1a", "Zone")
	secret     = flag.String("s", "", "Shared secret")
	shardType  = "banana"
)

func main() {
	flag.Parse()
	zongzi.SetLogLevelDebug()
	meta, _ := json.Marshal(map[string]any{"zone": *zone})
	c := &controller{}
	agent, err := zongzi.NewAgent(zongzi.AgentConfig{
		ClusterName: *name,
		Secret:      *secret,
		Discovery: udp.NewOracle(udp.Config{
			Multicast:       strings.Split(*discovery, ","),
			MulticastListen: fmt.Sprintf("239.108.0.1:%s", strings.Split(*raftAddr, ":")[1]),
			Secret:          *secret,
		}),
		HostConfig: zongzi.HostConfig{
			RTTMillisecond: 10,
			WALDir:         *dataDir + "/wal",
			NodeHostDir:    *dataDir + "/raft",
			RaftAddress:    *raftAddr,
			Gossip: zongzi.GossipConfig{
				BindAddress:      fmt.Sprintf("0.0.0.0:%s", strings.Split(*gossipAddr, ":")[1]),
				AdvertiseAddress: *gossipAddr,
				Meta:             meta,
			},
			RaftEventListener: c,
		},
	})
	if err != nil {
		panic(err)
	}
	agent.RegisterShardType(shardType, raftNodeFactory(), nil)
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
