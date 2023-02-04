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

const (
	minReplicas = 3
	raftTimeout = time.Second
	magicPrefix = "zongzi"
	joinTimeout = 5 * time.Second
)

type (
	ReadonlyLogReader = dragonboat.ReadonlyLogReader
	ShardView         = dragonboat.ShardView
	ShardInfo         = dragonboat.ShardInfo

	ReplicaConfig = config.Config
	GossipConfig  = config.GossipConfig
	HostConfig    = config.NodeHostConfig

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

	SHARD_JOIN         = "SHARD_JOIN"
	SHARD_JOIN_SUCCESS = "SHARD_JOIN_SUCCESS"
	SHARD_JOIN_REFUSED = "SHARD_JOIN_REFUSED"
	SHARD_JOIN_ERROR   = "SHARD_JOIN_ERROR"
)

// GossipOracle identifies gossip peers
type GossipOracle interface {
	// GetSeedList returns a list of gossip peers
	GetSeedList(Agent) ([]string, error)

	// Peers returns list of peers before cluster init
	Peers() map[string]string
}

type (
	AgentStatus   string
	HostStatus    string
	ShardStatus   string
	ReplicaStatus string
)

const (
	AgentStatus_Pending      = AgentStatus("pending")
	AgentStatus_Joining      = AgentStatus("joining")
	AgentStatus_Rejoining    = AgentStatus("rejoining")
	AgentStatus_Ready        = AgentStatus("ready")
	AgentStatus_Initializing = AgentStatus("initializing")
	AgentStatus_Active       = AgentStatus("active")

	HostStatus_New        = HostStatus("new")
	HostStatus_Active     = HostStatus("active")
	HostStatus_Missing    = HostStatus("missing")
	HostStatus_Recovering = HostStatus("recovering")
	HostStatus_Gone       = HostStatus("gone")

	ShardStatus_New         = ShardStatus("new")
	ShardStatus_Starting    = ShardStatus("starting")
	ShardStatus_Active      = ShardStatus("active")
	ShardStatus_Unavailable = ShardStatus("unavailable")
	ShardStatus_Closed      = ShardStatus("closed")

	ReplicaStatus_New      = ReplicaStatus("new")
	ReplicaStatus_Active   = ReplicaStatus("active")
	ReplicaStatus_Inactive = ReplicaStatus("inactive")
	ReplicaStatus_Done     = ReplicaStatus("done")
)

var DefaultReplicaConfig = ReplicaConfig{
	CheckQuorum:         true,
	CompactionOverhead:  1000,
	ElectionRTT:         10,
	HeartbeatRTT:        2,
	OrderedConfigChange: true,
	Quiesce:             false,
	SnapshotEntries:     10,
}

func MustBase36Decode(name string) uint64 {
	id, err := base36Decode(name)
	if err != nil {
		panic(err)
	}
	return id
}

func base36Decode(name string) (uint64, error) {
	return strconv.ParseUint(name, 36, 64)
}

func base36Encode(id uint64) string {
	return base36.Encode(id)
}

func raftCtx() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), raftTimeout)
	return ctx
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

func SetLogLevelSaneDebug() {
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

func parseUint64(s string) (uint64, error) {
	i, err := strconv.Atoi(s)
	return uint64(i), err
}
