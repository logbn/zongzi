package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/logbn/zongzi"
)

var (
	clusterName = flag.String("n", "test001", "Cluster name (base36 maxlen 12)")
	dataDir     = flag.String("d", "/var/lib/zongzi", "Base data directory")
	apiAddr     = flag.String("a", "10.0.0.1:17001", "Internal gRPC api address")
	raftAddr    = flag.String("r", "10.0.0.1:17002", "Dragonboat raft address")
	gossipAddr  = flag.String("g", "10.0.0.1:17003", "Memberlist gossip address")
	httpAddr    = flag.String("h", "10.0.0.1:8000", "HTTP address")
	zone        = flag.String("z", "us-west-1a", "Zone")
	peers       = flag.String("p", "10.0.0.1:17001, 10.0.0.2:17001, 10.0.0.3:17001", "Peer node api addresses")
	secret      = flag.String("s", "", "Shared secrets (csv)")
)

func main() {
	flag.Parse()
	zongzi.SetLogLevelDebug()
	ctrl := newController()
	agent, err := zongzi.NewAgent(
		*clusterName,
		strings.Split(*peers, ","),
		zongzi.WithApiAddress(*apiAddr),
		zongzi.WithGossipAddress(*gossipAddr),
		zongzi.WithHostConfig(zongzi.HostConfig{
			NodeHostDir:       *dataDir + "/raft",
			NotifyCommit:      true,
			RaftAddress:       *raftAddr,
			RaftEventListener: ctrl,
			RTTMillisecond:    2,
			WALDir:            *dataDir + "/wal",
		}),
		zongzi.WithHostTags("geo:zone="+*zone),
		zongzi.WithSecrets(strings.Split(*secret, ",")),
	)
	if err != nil {
		panic(err)
	}
	agent.ShardTypeRegister(
		stateMachineUri,
		stateMachineFactory(),
	)
	ctrl.agent = agent
	if err = agent.Start(); err != nil {
		panic(err)
	}
	if err = ctrl.Start(); err != nil {
		panic(err)
	}
	// Start HTTP API
	go func(s *http.Server) {
		log.Fatal(s.ListenAndServe())
	}(&http.Server{
		Addr:    *httpAddr,
		Handler: &handler{ctrl},
	})

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop
	ctrl.Stop()
	agent.Stop()
}
