package main

import (
	"context"
	"flag"
	"fmt"
	// "log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/logbn/zongzi"
)

var (
	name       = flag.String("n", `kv1`, "Cluster name (base36 maxlen 12)")
	peers      = flag.String("p", `kv1-usc1-a-0,kv1-usc1-c-0,kv1-usc1-f-0`, "Peer nodes")
	region     = flag.String("re", `us-central1`, "Region")
	zone       = flag.String("z", `us-central1-a`, "Zone")
	shards     = flag.Int("s", 4, "Shard count")
	apiAddr    = flag.String("a", "10.0.0.1:17001", "Internal gRPC api address")
	raftAddr   = flag.String("r", "10.0.0.1:17002", "Dragonboat raft address")
	gossipAddr = flag.String("g", "10.0.0.1:17003", "Memberlist gossip address")
	dataDir    = flag.String("d", "/var/lib/zongzi", "Base data directory")
)

func main() {
	flag.Parse()
	zongzi.SetLogLevelDebug()
	ctx := context.Background()
	ctrl := newController()
	agent, err := zongzi.NewAgent(*name, strings.Split(*peers, ","),
		zongzi.WithHostTags(
			fmt.Sprintf(`geo:region=%s`, *region),
			fmt.Sprintf(`geo:zone=%s`, *zone)),
		zongzi.WithApiAddress(*apiAddr),
		zongzi.WithGossipAddress(*gossipAddr),
		zongzi.WithHostConfig(zongzi.HostConfig{
			NodeHostDir:    *dataDir + "/raft",
			NotifyCommit:   true,
			RaftAddress:    *raftAddr,
			RTTMillisecond: 2,
			WALDir:         *dataDir + "/wal",
		}),
		zongzi.WithRaftEventListener(ctrl))
	if err != nil {
		panic(err)
	}
	agent.RegisterStateMachine(uri, factory)
	if err = agent.Start(); err != nil {
		panic(err)
	}
	// var clients = make([]zongzi.ShardClient, *shards)
	for i := 0; i < *shards; i++ {
		_, _, err := agent.RegisterShard(ctx, uri,
			zongzi.WithName(fmt.Sprintf(`%s-%05d`, name, i)),
			zongzi.WithPlacementVary(`geo:zone`),
			zongzi.WithPlacementMembers(3, `geo:region=`+*region),
			zongzi.WithPlacementReplicas(*region, 3, `geo:region=`+*region)) // Place 3 read replicas in this region
		if err != nil {
			panic(err)
		}
		// clients[i] = agent.ShardClient(shard.ID)
	}
	if err = ctrl.Start(agent); err != nil {
		panic(err)
	}

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop
	ctrl.Stop()
	agent.Stop()
}
