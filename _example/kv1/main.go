package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/logbn/zongzi"
)

var (
	gossipAddr = flag.String("g", "10.0.0.1:17001", "Memberlist gossip address")
	raftAddr   = flag.String("r", "10.0.0.1:17002", "Dragonboat raft address")
	zongziAddr = flag.String("a", "10.0.0.1:17003", "Zongzi gRPC api address")
	httpAddr   = flag.String("h", "10.0.0.1:8000", "HTTP address")

	name    = flag.String("n", `kv1`, "Cluster name (base36 maxlen 12)")
	peers   = flag.String("p", `kv1-usc1-a-0,kv1-usc1-c-0,kv1-usc1-f-0`, "Peer nodes")
	region  = flag.String("re", `us-central1`, "Region")
	zone    = flag.String("z", `us-central1-a`, "Zone")
	dataDir = flag.String("d", "/var/lib/zongzi", "Base data directory")
	shards  = flag.Int("s", 4, "Shard count")
)

func main() {
	flag.Parse()
	zongzi.SetLogLevelDebug()
	ctx := context.Background()
	agent, err := zongzi.NewAgent(*name, strings.Split(*peers, ","),
		zongzi.WithRaftDir(*dataDir+"/raft"),
		zongzi.WithWALDir(*dataDir+"/wal"),
		zongzi.WithGossipAddress(*gossipAddr),
		zongzi.WithRaftAddress(*raftAddr),
		zongzi.WithApiAddress(*zongziAddr),
		zongzi.WithHostTags(
			fmt.Sprintf(`geo:region=%s`, *region),
			fmt.Sprintf(`geo:zone=%s`, *zone)))
	if err != nil {
		panic(err)
	}
	agent.StateMachineRegister(uri, factory)
	if err = agent.Start(ctx); err != nil {
		panic(err)
	}
	var clients = make([]zongzi.ShardClient, *shards)
	for i := 0; i < *shards; i++ {
		shard, _, err := agent.ShardCreate(ctx, uri,
			zongzi.WithName(fmt.Sprintf(`%s-%05d`, *name, i)),
			zongzi.WithPlacementVary(`geo:zone`),
			zongzi.WithPlacementMembers(3, `geo:region=`+*region))
		// zongzi.WithPlacementReplicas(*region, 3, `geo:region=`+*region)) // Place 3 read replicas in this region
		if err != nil {
			panic(err)
		}
		clients[i] = agent.GetClient(shard.ID)
	}
	// Start HTTP API
	go func(s *http.Server) {
		log.Fatal(s.ListenAndServe())
	}(&http.Server{
		Addr:    *httpAddr,
		Handler: &handler{clients},
	})

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop

	agent.Stop()
}
