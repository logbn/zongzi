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
)

const (
	minReplicas  = 3
	raftTimeout  = time.Second
	joinTimeout  = 5 * time.Second
	projectName  = "zongzi"
	shardVersion = "v0.0.1"
)

var (
	DefaultApiAddress    = "127.0.0.1:17001"
	DefaultGossipAddress = "127.0.0.1:17002"
	DefaultHostConfig    = HostConfig{
		NodeHostDir:    "/var/lib/zongzi/raft",
		RaftAddress:    "127.0.0.1:17003",
		RTTMillisecond: 100,
		WALDir:         "/var/lib/zongzi/wal",
	}
	DefaultReplicaConfig = ReplicaConfig{
		CheckQuorum:         true,
		CompactionOverhead:  1000,
		ElectionRTT:         10,
		HeartbeatRTT:        2,
		OrderedConfigChange: true,
		Quiesce:             false,
		SnapshotEntries:     10,
	}
)

type (
	shardType struct {
		Config  ReplicaConfig
		Factory StateMachineFactory
		Uri     string
		Version string
	}
	watchQuery struct {
		close  chan struct{}
		query  Entry
		result chan Result
	}
)

type (
	HostConfig    = config.NodeHostConfig
	ReplicaConfig = config.Config

	LeaderInfo = raftio.LeaderInfo

	Entry  = statemachine.Entry
	Result = statemachine.Result

	LogLevel = logger.LogLevel

	AgentStatus   string
	HostStatus    string
	ShardStatus   string
	ReplicaStatus string
)

const (
	LogLevelCritical = logger.CRITICAL
	LogLevelError    = logger.ERROR
	LogLevelWarning  = logger.WARNING
	LogLevelInfo     = logger.INFO
	LogLevelDebug    = logger.DEBUG

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
	ShardStatus_Unavailable = ShardStatus("unavailable")

	ReplicaStatus_Active = ReplicaStatus("active")
	ReplicaStatus_Closed = ReplicaStatus("closed")
	ReplicaStatus_New    = ReplicaStatus("new")
)

var (
	ErrAborted       = dragonboat.ErrAborted
	ErrCanceled      = dragonboat.ErrCanceled
	ErrRejected      = dragonboat.ErrRejected
	ErrShardClosed   = dragonboat.ErrShardClosed
	ErrShardNotReady = dragonboat.ErrShardNotReady
	ErrTimeout       = dragonboat.ErrTimeout

	ErrHostNotFound      = fmt.Errorf(`Host not found`)
	ErrIDOutOfRange      = fmt.Errorf(`ID out of range`)
	ErrReplicaNotActive  = fmt.Errorf("Replica not active")
	ErrReplicaNotAllowed = fmt.Errorf("Replica not allowed")
	ErrReplicaNotFound   = fmt.Errorf("Replica not found")
	ErrShardNotFound     = fmt.Errorf(`Shard not found`)

	// ErrClusterNameInvalid indicates that the clusterName is invalid
	// Base36 supports only lowercase alphanumeric characters
	ErrClusterNameInvalid = fmt.Errorf("Invalid cluster name (base36 maxlen 12)")
	ClusterNameRegex      = `^[a-z0-9]{1,12}$`

	ErrAgentNotReady = fmt.Errorf("Agent not ready")

	// ErrNotifyCommitDisabled is logged when non-linearizable writes are requested but disabled.
	// Set property `NotifyCommit` to `true` in `HostConfig` to add support for non-linearizable writes.
	ErrNotifyCommitDisabled = fmt.Errorf("Attempted to make a non-linearizable write while NotifyCommit is disabled")
)

func mustBase36Decode(name string) uint64 {
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
	return strconv.FormatUint(id, 36)
}

func raftCtx(ctxs ...context.Context) (ctx context.Context) {
	ctx = context.Background()
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	}
	ctx, _ = context.WithTimeout(ctx, raftTimeout)
	return
}

// SetLogLevel sets log level for all zongzi and dragonboat loggers.
//
// Recommend [LogLevelWarning] for production.
func SetLogLevel(level LogLevel) {
	logger.GetLogger("dragonboat").SetLevel(level)
	logger.GetLogger("gossip").SetLevel(level)
	logger.GetLogger("grpc").SetLevel(level)
	logger.GetLogger("logdb").SetLevel(level)
	logger.GetLogger("raft").SetLevel(level)
	logger.GetLogger("rsm").SetLevel(level)
	logger.GetLogger("transport").SetLevel(level)
	logger.GetLogger("zongzi").SetLevel(level)
}

// SetLogLevelDebug sets a debug log level for most loggers
// but filters out loggers having tons of debug output
func SetLogLevelDebug() {
	SetLogLevel(logger.DEBUG)
	logger.GetLogger("gossip").SetLevel(logger.ERROR)
	logger.GetLogger("dragonboat").SetLevel(logger.WARNING)
	logger.GetLogger("raft").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)
}

// SetLogLevelProduction sets a good log level for production
func SetLogLevelProduction() {
	SetLogLevel(logger.WARNING)
	logger.GetLogger("gossip").SetLevel(logger.ERROR)
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

func sliceWithout[T comparable](slice []T, exclude T) []T {
	var out []T
	for _, v := range slice {
		if v != exclude {
			out = append(out, v)
		}
	}
	return out
}
