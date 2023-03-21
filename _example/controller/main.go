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
)

func main() {
	var (
		clusterName = flag.String("n", "test001", "Cluster name (base36 maxlen 12)")
		dataDir     = flag.String("d", "/var/lib/zongzi", "Base data directory")
		peers       = flag.String("p", "127.0.0.1:10801", "Peer nodes")
		listenAddr  = flag.String("l", "127.0.0.1:10801", "Listen address")
		gossipAddr  = flag.String("g", "127.0.0.1:10802", "Gossip address")
		raftAddr    = flag.String("r", "127.0.0.1:10803", "Raft address")
		zone        = flag.String("z", "us-west-1a", "Zone")
		secret      = flag.String("s", "", "Shared secrets (csv)")
	)
	flag.Parse()
	zongzi.SetLogLevelDebug()
	ctrl := newController()
	meta, _ := json.Marshal(map[string]any{"zone": *zone})
	agent, err := zongzi.NewAgent(
		*clusterName,
		strings.Split(*peers, ","),
		zongzi.AgentOptionApiAddress(*listenAddr),
		zongzi.AgentOptionGossipAddress(*gossipAddr),
		zongzi.AgentOptionHostConfig(zongzi.HostConfig{
			NodeHostDir:       *dataDir + "/raft",
			NotifyCommit:      true,
			RaftAddress:       *raftAddr,
			RaftEventListener: ctrl,
			RTTMillisecond:    100,
			WALDir:            *dataDir + "/wal",
		}),
		zongzi.AgentOptionMeta(meta),
		zongzi.AgentOptionSecrets(strings.Split(*secret, ",")),
	)
	if err != nil {
		panic(err)
	}
	agent.RegisterStateMachine(
		StateMachineUri,
		StateMachineVersion,
		StateMachineFactory(),
	)
	ctrl.agent = agent
	if err = agent.Start(); err != nil {
		panic(err)
	}

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop
	agent.Stop()
}
