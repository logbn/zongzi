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
	"logbin.com/zongzi"
)

var (
	autoinit   = flag.Int("i", 0, "Minimum peer count for autoinit (default 0 disabled)")
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
	go agent.Start()
	if err = unsafe_init(agent); err != nil {
		panic(err)
	}
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop
	agent.Stop()
}

// May cause split brain in real systems when used improperly.
// This type of implicit bootstrap method is for demo purposes only.
// Cluster should always be bootstrapped explicitly in production systems.
// See other examples.
func unsafe_init(agent zongzi.Agent) (err error) {
	var initialized bool
	for {
		switch agent.GetStatus() {
		case zongzi.AgentStatus_Ready:
			if !initialized && *autoinit > 0 && agent.PeerCount() >= *autoinit {
				log.Println("--- Autoinit ---")
				if err := agent.Init(); err != nil {
					log.Printf("%v", err)
					time.Sleep(time.Second)
					continue
				}
				initialized = true
			}
		case zongzi.AgentStatus_Active:
			log.Println("--- Active ---")
			go func() {
				nh := agent.GetNodeHost()
				reg, _ := nh.GetNodeHostRegistry()
				hostMeta, _ := reg.GetMeta(nh.ID())
				clusterMeta, _ := agent.GetMeta()
				b, _ := json.Marshal(clusterMeta)
				log.Printf("nodehost: %s", nh.ID())
				log.Printf("host meta: %s", hostMeta)
				log.Printf("cluster meta: %s", string(b))
			}()
			return
		}
		time.Sleep(time.Second)
	}
	return
}

func setLogLevel(level logger.LogLevel) {
	logger.GetLogger("dragonboat").SetLevel(level)
	logger.GetLogger("transport").SetLevel(level)
	logger.GetLogger("logdb").SetLevel(level)
	logger.GetLogger("raft").SetLevel(level)
	logger.GetLogger("grpc").SetLevel(level)
	logger.GetLogger("rsm").SetLevel(level)
	zongzi.GetLogger("agent").SetLevel(level)
}
