package zongzi

import (
	"context"
	"fmt"
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
	LogReader = dragonboat.ReadonlyLogReader
	ShardView = dragonboat.ShardView
	ShardInfo = dragonboat.ShardInfo

	GossipConfig  = config.GossipConfig
	HostConfig    = config.NodeHostConfig
	ReplicaConfig = config.Config

	ConnectionInfo = raftio.ConnectionInfo
	EntryInfo      = raftio.EntryInfo
	LeaderInfo     = raftio.LeaderInfo
	NodeInfo       = raftio.NodeInfo
	SnapshotInfo   = raftio.SnapshotInfo

	Entry  = statemachine.Entry
	Result = statemachine.Result
)

type Watch struct {
	ctx        context.Context
	query      Entry
	resultChan chan Result
}

var (
	ErrAborted       = dragonboat.ErrAborted
	ErrCanceled      = dragonboat.ErrCanceled
	ErrRejected      = dragonboat.ErrRejected
	ErrShardClosed   = dragonboat.ErrShardClosed
	ErrShardNotReady = dragonboat.ErrShardNotReady
	ErrTimeout       = dragonboat.ErrTimeout
)

var (
	ErrReplicaNotFound   = fmt.Errorf("Replica not found")
	ErrReplicaNotActive  = fmt.Errorf("Replica not active")
	ErrReplicaNotAllowed = fmt.Errorf("Replica not allowed")

	ErrAgentNotReady = fmt.Errorf("Agent not ready")

	// ErrNotifyCommitDisabled is logged when non-linearizable writes are requested but disabled.
	// Set property `NotifyCommit` to `true` in `HostConfig` to add support for non-linearizable writes.
	ErrNotifyCommitDisabled = fmt.Errorf("Attempting to make non-linearizable write while NotifyCommit is disabled")
)

type (
	AgentStatus   string
	HostStatus    string
	ShardStatus   string
	ReplicaStatus string
)

const (
	AgentStatus_Active       = AgentStatus("active")
	AgentStatus_Initializing = AgentStatus("initializing")
	AgentStatus_Joining      = AgentStatus("joining")
	AgentStatus_Pending      = AgentStatus("pending")
	AgentStatus_Ready        = AgentStatus("ready")
	AgentStatus_Rejoining    = AgentStatus("rejoining")

	HostStatus_Active     = HostStatus("active")
	HostStatus_Gone       = HostStatus("gone")
	HostStatus_Missing    = HostStatus("missing")
	HostStatus_New        = HostStatus("new")
	HostStatus_Recovering = HostStatus("recovering")

	ShardStatus_Active      = ShardStatus("active")
	ShardStatus_Closed      = ShardStatus("closed")
	ShardStatus_New         = ShardStatus("new")
	ShardStatus_Starting    = ShardStatus("starting")
	ShardStatus_Unavailable = ShardStatus("unavailable")

	ReplicaStatus_Active   = ReplicaStatus("active")
	ReplicaStatus_Done     = ReplicaStatus("done")
	ReplicaStatus_Inactive = ReplicaStatus("inactive")
	ReplicaStatus_New      = ReplicaStatus("new")
)

var DefaultReplicaConfig = ReplicaConfig{
	ShardID:             0,
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

func raftCtx(ctxs ...context.Context) (ctx context.Context) {
	ctx = context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}
	ctx, _ = context.WithTimeout(ctx, raftTimeout)
	return
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

func parseUint64(s string) (uint64, error) {
	i, err := strconv.Atoi(s)
	return uint64(i), err
}

func keys[K comparable, V any](m map[K]V) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

func sliceContains[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
