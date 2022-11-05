package zongzi

import (
	"context"
	"strconv"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/lni/dragonboat/v4/raftio"
	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/martinlindhe/base36"
)

type (
	AgentStatus string

	ReadonlyLogReader = dragonboat.ReadonlyLogReader

	Config         = config.Config
	GossipConfig   = config.GossipConfig
	NodeHostConfig = config.NodeHostConfig

	LeaderInfo     = raftio.LeaderInfo
	NodeInfo       = raftio.NodeInfo
	ConnectionInfo = raftio.ConnectionInfo
	SnapshotInfo   = raftio.SnapshotInfo
	EntryInfo      = raftio.EntryInfo

	CreateStateMachineFunc  = statemachine.CreateStateMachineFunc
	IStateMachine           = statemachine.IStateMachine
	Result                  = statemachine.Result
	Entry                   = statemachine.Entry
	ISnapshotFileCollection = statemachine.ISnapshotFileCollection
	SnapshotFile            = statemachine.SnapshotFile
)

const (
	minReplicas  = 3
	magicPrefix  = "zongzi"
	probeTimeout = 3 * time.Second
	probePause   = 5 * time.Second
	joinTimeout  = 5 * time.Second
	raftTimeout  = time.Second

	AgentStatus_Unknown      = AgentStatus("Unknown")
	AgentStatus_Pending      = AgentStatus("Pending")
	AgentStatus_Joining      = AgentStatus("Joining")
	AgentStatus_Rejoining    = AgentStatus("Rejoining")
	AgentStatus_Ready        = AgentStatus("Ready")
	AgentStatus_Initializing = AgentStatus("Initializing")
	AgentStatus_Active       = AgentStatus("Active")

	ReplicaStatus_New    = "new"
	ReplicaStatus_Ready  = "ready"
	ReplicaStatus_Active = "active"
	ReplicaStatus_Gone   = "gone"

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

func SetLogLevel(level logger.LogLevel) {
	logger.GetLogger("dragonboat").SetLevel(level)
	logger.GetLogger("gossip").SetLevel(level)
	logger.GetLogger("grpc").SetLevel(level)
	logger.GetLogger("logdb").SetLevel(level)
	logger.GetLogger("raft").SetLevel(level)
	logger.GetLogger("rsm").SetLevel(level)
	logger.GetLogger("transport").SetLevel(level)
	logger.GetLogger("zongzi").SetLevel(level)
}

func SetLogLevelDebug() {
	SetLogLevel(logger.DEBUG)
	logger.GetLogger("gossip").SetLevel(logger.ERROR)
	logger.GetLogger("dragonboat").SetLevel(logger.WARNING)
	logger.GetLogger("raft").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)
}

type compositeRaftEventListener struct {
	listeners []raftio.IRaftEventListener
}

func newCompositeRaftEventListener(listeners ...raftio.IRaftEventListener) raftio.IRaftEventListener {
	return &compositeRaftEventListener{listeners}
}

func (c *compositeRaftEventListener) LeaderUpdated(info LeaderInfo) {
	for _, listener := range c.listeners {
		if listener != nil {
			listener.LeaderUpdated(info)
		}
	}
}
