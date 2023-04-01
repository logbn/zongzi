package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
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
		peers       = flag.String("p", "127.0.0.1:17001", "Peer nodes")
		listenAddr  = flag.String("l", "127.0.0.1:17001", "Listen address")
		gossipAddr  = flag.String("g", "127.0.0.1:17002", "Gossip address")
		raftAddr    = flag.String("r", "127.0.0.1:17003", "Raft address")
		httpAddr    = flag.String("h", "127.0.0.1:8000", "HTTP address")
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
		zongzi.WithApiAddress(*listenAddr),
		zongzi.WithGossipAddress(*gossipAddr),
		zongzi.WithHostConfig(zongzi.HostConfig{
			NodeHostDir:       *dataDir + "/raft",
			NotifyCommit:      true,
			RaftAddress:       *raftAddr,
			RaftEventListener: ctrl,
			RTTMillisecond:    10,
			WALDir:            *dataDir + "/wal",
		}),
		zongzi.WithMeta(meta),
		zongzi.WithSecrets(strings.Split(*secret, ",")),
	)
	if err != nil {
		panic(err)
	}
	agent.RegisterStateMachine(
		stateMachineUri,
		stateMachineVersion,
		stateMachineFactory(),
	)
	if err = agent.Start(); err != nil {
		panic(err)
	}
	ctrl.agent = agent
	ctrl.Start()
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
