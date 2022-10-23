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
	minReplicas  = 3
	magicPrefix  = "zongzi"
	probeTimeout = 3 * time.Second
	probePause   = 5 * time.Second
	joinTimeout  = 5 * time.Second
	raftTimeout  = time.Second
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

const (
	PROBE_JOIN         = "PROBE_JOIN"
	INIT               = "INIT"
	INIT_ERROR         = "INIT_ERROR"
	INIT_CONFLICT      = "INIT_CONFLICT"
	INIT_SUCCESS       = "INIT_SUCCESS"
	INIT_HOST          = "INIT_HOST"
	INIT_HOST_ERROR    = "INIT_HOST_ERROR"
	INIT_HOST_SUCCESS  = "INIT_HOST_SUCCESS"
	INIT_SHARD         = "INIT_SHARD"
	INIT_SHARD_ERROR   = "INIT_SHARD_ERROR"
	INIT_SHARD_SUCCESS = "INIT_SHARD_SUCCESS"
	JOIN_HOST          = "JOIN_HOST"
	JOIN_ERROR         = "JOIN_ERROR"
	JOIN_SHARD         = "JOIN_SHARD"

	PROBE_REJOIN = "PROBE_REJOIN"
	REJOIN_PEER  = "REJOIN_PEER"
)

var DefaultRaftNodeConfig = config.Config{
	CheckQuorum:         true,
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

func strMapCopy(m map[string]string) map[string]string {
	c := map[string]string{}
	for k, v := range m {
		c[k] = v
	}
	return c
}
