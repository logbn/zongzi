package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/logbn/zongzi"
)

var (
	cluster    = flag.String("c", "test001", "Cluster name (base36 maxlen 12)")
	dataDir    = flag.String("d", "/var/lib/zongzi/node", "Base data directory")
	raftDir    = flag.String("r", "127.0.0.1:10801", "Raft address")
	gossipAddr = flag.String("g", "127.0.0.1:10802", "Gossip address")
	discovery  = flag.String("u", "239.108.0.1:10801", "UDP multicast discovery addresses (csv)")
	zone       = flag.String("z", "us-west-1a", "Zone")
)

func main() {
	flag.Parse()
	setLogLevel(logger.DEBUG)
	discoveryAddrs := strings.Split(*discovery, ",")
	meta, err := json.Marshal(map[string]string{"zone": *zone})
	if err != nil {
		panic(err)
	}
	agent, err := zongzi.NewAgent(zongzi.AgentConfig{
		Multicast: discoveryAddrs,
		NodeHostConfig: config.NodeHostConfig{
			DeploymentID: zongzi.MustBase36Decode(*cluster),
			WALDir:       *dataDir + "/wal",
			NodeHostDir:  *dataDir + "/raft",
			RaftAddress:  *raftDir,
			Gossip: config.GossipConfig{
				BindAddress:      fmt.Sprintf("0.0.0.0:%s", strings.Split(*gossipAddr, ":")[1]),
				AdvertiseAddress: *gossipAddr,
				Meta:             meta,
			},
			RTTMillisecond: 10,
		},
	})
	if err != nil {
		panic(err)
	}
	if err = agent.Start(); err != nil {
		panic(err)
	}
	for {
		if agent.GetStatus() == zongzi.AgentStatus_Active {
			nh := agent.GetNodeHost()
			reg, _ := nh.GetNodeHostRegistry()
			hostMeta, _ := reg.GetMeta(nh.ID())
			snapshotJson, _ := agent.GetSnapshotJson()
			log.Printf("nodehost: %s", nh.ID())
			log.Printf("host meta: %s", hostMeta)
			log.Printf("snapshot: %s", string(snapshotJson))
			break
		}
		time.Sleep(time.Second)
	}
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop
	agent.Stop()
}

func setLogLevel(level logger.LogLevel) {
	logger.GetLogger("dragonboat").SetLevel(level)
	logger.GetLogger("transport").SetLevel(level)
	logger.GetLogger("logdb").SetLevel(level)
	logger.GetLogger("raft").SetLevel(level)
	logger.GetLogger("grpc").SetLevel(level)
	logger.GetLogger("rsm").SetLevel(level)
	logger.GetLogger("zongzi").SetLevel(level)
}
