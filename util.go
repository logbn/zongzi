package zongzi

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/lni/dragonboat/v4/plugin/tan"
	"github.com/lni/dragonboat/v4/raftio"
	"github.com/lni/dragonboat/v4/statemachine"
)

// force google.golang.org/genproto v0.0.0-20250303144028-a0af3efb3deb to stay in go.mod
import _ "google.golang.org/genproto/protobuf/ptype"

const (
	projectName = "zongzi"
	projectUri  = "zongzi://github.com/logbn/zongzi"
	minReplicas = 3
	raftTimeout = time.Second
	joinTimeout = 5 * time.Second
	waitPeriod  = 500 * time.Millisecond
)

const (
	DefaultGossipAddress = "127.0.0.1:17001"
	DefaultRaftAddress   = "127.0.0.1:17002"
	DefaultApiAddress    = "127.0.0.1:17003"

	ShardID = 0
)

var (
	DefaultHostConfig = HostConfig{
		NodeHostDir:    "/var/lib/zongzi/raft",
		RaftAddress:    DefaultRaftAddress,
		RTTMillisecond: 10,
		WALDir:         "/var/lib/zongzi/wal",
		Expert: config.ExpertConfig{
			LogDBFactory: tan.Factory,
			LogDB:        config.GetMediumMemLogDBConfig(),
		},
	}
	DefaultReplicaConfig = ReplicaConfig{
		CheckQuorum:             true,
		CompactionOverhead:      10000,
		ElectionRTT:             100,
		EntryCompressionType:    config.Snappy,
		HeartbeatRTT:            10,
		OrderedConfigChange:     true,
		Quiesce:                 false,
		SnapshotCompressionType: config.Snappy,
		SnapshotEntries:         10000,
		WaitReady:               true,
	}
)

type (
	nodeHostInfoOption = dragonboat.NodeHostInfoOption

	HostConfig    = config.NodeHostConfig
	ReplicaConfig = config.Config
	GossipConfig  = config.GossipConfig
	LogDBConfig   = config.LogDBConfig
	EngineConfig  = config.EngineConfig

	LeaderInfo          = raftio.LeaderInfo
	RaftEventListener   = raftio.IRaftEventListener
	SystemEventListener = raftio.ISystemEventListener

	Entry                  = statemachine.Entry
	Result                 = statemachine.Result
	SnapshotFile           = statemachine.SnapshotFile
	SnapshotFileCollection = statemachine.ISnapshotFileCollection

	LogLevel = logger.LogLevel
	Logger   = logger.ILogger

	AgentStatus   string
	HostStatus    string
	ShardStatus   string
	ReplicaStatus string
)

var (
	GetLogger = logger.GetLogger

	GetDefaultEngineConfig = config.GetDefaultEngineConfig
)

const (
	LogLevelCritical = logger.CRITICAL
	LogLevelError    = logger.ERROR
	LogLevelWarning  = logger.WARNING
	LogLevelInfo     = logger.INFO
	LogLevelDebug    = logger.DEBUG
)

const (
	AgentStatus_Initializing = AgentStatus("initializing")
	AgentStatus_Joining      = AgentStatus("joining")
	AgentStatus_Pending      = AgentStatus("pending")
	AgentStatus_Ready        = AgentStatus("ready")
	AgentStatus_Rejoining    = AgentStatus("rejoining")
	AgentStatus_Stopped      = AgentStatus("stopped")

	HostStatus_Active     = HostStatus("active")
	HostStatus_Gone       = HostStatus("gone")
	HostStatus_Missing    = HostStatus("missing")
	HostStatus_New        = HostStatus("new")
	HostStatus_Recovering = HostStatus("recovering")

	ShardStatus_Active      = ShardStatus("active")
	ShardStatus_Closed      = ShardStatus("closed")
	ShardStatus_Closing     = ShardStatus("closing")
	ShardStatus_New         = ShardStatus("new")
	ShardStatus_Unavailable = ShardStatus("unavailable")

	ReplicaStatus_Active        = ReplicaStatus("active")
	ReplicaStatus_Closed        = ReplicaStatus("closed")
	ReplicaStatus_Closing       = ReplicaStatus("closing")
	ReplicaStatus_Bootstrapping = ReplicaStatus("bootstrapping")
	ReplicaStatus_Joining       = ReplicaStatus("joining")
	ReplicaStatus_New           = ReplicaStatus("new")
)

var (
	ErrAborted       = dragonboat.ErrAborted
	ErrCanceled      = dragonboat.ErrCanceled
	ErrRejected      = dragonboat.ErrRejected
	ErrShardClosed   = dragonboat.ErrShardClosed
	ErrShardNotReady = dragonboat.ErrShardNotReady
	ErrTimeout       = dragonboat.ErrTimeout

	ErrAgentNotReady     = fmt.Errorf("Agent not ready")
	ErrHostNotFound      = fmt.Errorf(`Host not found`)
	ErrIDOutOfRange      = fmt.Errorf(`ID out of range`)
	ErrInvalidFactory    = fmt.Errorf(`Invalid Factory`)
	ErrInvalidGossipAddr = fmt.Errorf(`Invalid gossip address`)
	ErrReplicaNotActive  = fmt.Errorf("Replica not active")
	ErrReplicaNotAllowed = fmt.Errorf("Replica not allowed")
	ErrReplicaNotFound   = fmt.Errorf("Replica not found")
	ErrShardExists       = fmt.Errorf(`Shard already exists`)
	ErrShardNotFound     = fmt.Errorf(`Shard not found`)

	ErrStreamConnectMissing   = fmt.Errorf(`First message of stream should be connect message`)
	ErrStreamConnectDuplicate = fmt.Errorf(`Connect message already received`)

	ErrInvalidNumberOfArguments = fmt.Errorf(`Invalid number of arguments`)

	// ErrClusterNameInvalid indicates that the clusterName is invalid
	// Base36 supports only lowercase alphanumeric characters
	ErrClusterNameInvalid = fmt.Errorf("Invalid cluster name (base36 maxlen 12)")
	ClusterNameRegex      = `^[a-z0-9]{1,12}$`

	// ErrNotifyCommitDisabled is logged when non-linearizable writes are requested but disabled.
	// Set property `NotifyCommit` to `true` in `HostConfig` to add support for non-linearizable writes.
	ErrNotifyCommitDisabled = fmt.Errorf("Attempted to make a non-linearizable write while NotifyCommit is disabled")
)

type (
	lookupQuery struct {
		ctx  context.Context
		data []byte
	}
	watchQuery struct {
		ctx    context.Context
		data   []byte
		result chan *Result
	}
	streamQuery struct {
		ctx context.Context
		in  chan []byte
		out chan *Result
	}
)

func (q *lookupQuery) Release() {
	q.ctx = nil
	q.data = q.data[:0]
	lookupQueryPool.Put(q)
}

var lookupQueryPool = sync.Pool{New: func() any { return &lookupQuery{} }}

func getLookupQuery() *lookupQuery {
	return lookupQueryPool.Get().(*lookupQuery)
}

func newLookupQuery(ctx context.Context, data []byte) (q *lookupQuery) {
	q = lookupQueryPool.Get().(*lookupQuery)
	q.ctx = ctx
	q.data = data
	return q
}

func (q *watchQuery) Release() {
	q.ctx = nil
	q.data = q.data[:0]
	q.result = nil
	watchQueryPool.Put(q)
}

var watchQueryPool = sync.Pool{New: func() any { return &watchQuery{} }}

func getWatchQuery() *watchQuery {
	return watchQueryPool.Get().(*watchQuery)
}

func newWatchQuery(ctx context.Context, data []byte, result chan *Result) (q *watchQuery) {
	q = watchQueryPool.Get().(*watchQuery)
	q.ctx = ctx
	q.data = data
	q.result = result
	return
}

func (q *streamQuery) Release() {
	q.ctx = nil
	q.in = nil
	q.out = nil
	streamQueryPool.Put(q)
}

var streamQueryPool = sync.Pool{New: func() any { return &streamQuery{} }}

func getStreamQuery() *streamQuery {
	return streamQueryPool.Get().(*streamQuery)
}

func newStreamQuery(ctx context.Context, in chan []byte, out chan *Result) (q *streamQuery) {
	q = streamQueryPool.Get().(*streamQuery)
	q.ctx = ctx
	q.in = in
	q.out = out
	return
}

var resultPool = sync.Pool{New: func() any { return &Result{} }}

func ReleaseResult(r *Result) {
	r.Value = 0
	r.Data = r.Data[:0]
	resultPool.Put(r)
}

// GetResult can be used to efficiently retrieve an empty Result from a global pool. It is recommended to use this
// method to instantiate Result objects returned by Lookup or sent over Watch channels as they will be automatically
// returned to the pool to reduce allocation overhead.
func GetResult() *Result {
	return resultPool.Get().(*Result)
}

var requestStatePool = sync.Pool{New: func() any { return &dragonboat.RequestState{} }}

func getRequestState() *dragonboat.RequestState {
	return requestStatePool.Get().(*dragonboat.RequestState)
}

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
	logger.GetLogger("tan").SetLevel(level)
	logger.GetLogger("registry").SetLevel(level)
	logger.GetLogger("config").SetLevel(level)
}

// SetLogLevelDebug sets a debug log level for most loggers.
// but filters out loggers having tons of debug output.
func SetLogLevelDebug() {
	SetLogLevel(logger.DEBUG)
	logger.GetLogger("dragonboat").SetLevel(logger.WARNING)
	logger.GetLogger("gossip").SetLevel(logger.ERROR)
	logger.GetLogger("raft").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)
}

// SetLogLevelProduction sets a good log level for production (gossip logger is a bit noisy).
func SetLogLevelProduction() {
	SetLogLevel(logger.WARNING)
	logger.GetLogger("gossip").SetLevel(logger.ERROR)
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
