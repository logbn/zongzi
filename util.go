package zongzi

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/lni/dragonboat/v4/config"
	"github.com/martinlindhe/base36"
)

const (
	magicPrefix  = "zongzi"
	probeTimeout = 3 * time.Second
	probePause   = 5 * time.Second
	joinTimeout  = 5 * time.Second
	raftTimeout  = time.Second
	metaShardID  = 1
)

type AgentStatus string

var (
	errNodeNotFound = fmt.Errorf("Node not found")

	AgentStatus_Unknown      = AgentStatus("Unknown")
	AgentStatus_Pending      = AgentStatus("Pending")
	AgentStatus_Joining      = AgentStatus("Joining")
	AgentStatus_Rejoining    = AgentStatus("Rejoining")
	AgentStatus_Ready        = AgentStatus("Ready")
	AgentStatus_Initializing = AgentStatus("Initializing")
	AgentStatus_Active       = AgentStatus("Active")

	ReplicaStatus_New = "new"
)

var DefaultRaftNodeConfig = config.Config{
	CheckQuorum:         true,
	ShardID:             metaShardID,
	CompactionOverhead:  1000,
	ElectionRTT:         10,
	HeartbeatRTT:        2,
	OrderedConfigChange: true,
	Quiesce:             false,
	SnapshotEntries:     10,
}

func MustBase36Decode(name string) uint64 {
	id, err := strconv.ParseUint(name, 36, 64)
	if err != nil {
		panic(err)
	}
	return id
}

func base36Encode(id uint64) string {
	return base36.Encode(id)
}

func raftCtx() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), raftTimeout)
	return ctx
}
