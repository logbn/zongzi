package zongzi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/logger"

	"github.com/logbn/zongzi/udp"
)

type AgentConfig struct {
	ClusterName   string
	Discovery     GossipOracle
	HostConfig    HostConfig
	ReplicaConfig ReplicaConfig
}

type Agent interface {
	Start() error
	Stop()
	Init() error
	GetClusterName() string
	GetHost() *dragonboat.NodeHost
	HostID() string
	GetHostConfig() HostConfig
	GetStatus() AgentStatus
	GetSnapshot(index uint64) (Snapshot, error)
	GetSnapshotJson() ([]byte, error)
	CreateShard(shardTypeName string) (shard *Shard, err error)
	CreateReplica(shardID uint64, nodeHostID string, isNonVoting bool) (id uint64, err error)
}

type agent struct {
	log         logger.ILogger
	hostConfig  HostConfig
	primeConfig ReplicaConfig
	clusterName string
	client      udp.Client
	controller  *controller
	shardTypes  map[string]shardType
	discovery   GossipOracle

	host   *dragonboat.NodeHost
	hostFS fs.FS
	fsm    *fsm
	clock  clock.Clock
	status AgentStatus
	mutex  sync.RWMutex
	cancel context.CancelFunc
}

type shardType struct {
	Factory CreateStateMachineFunc
	Config  ReplicaConfig
}

func NewAgent(cfg AgentConfig) (a *agent, err error) {
	if cfg.ReplicaConfig.ElectionRTT == 0 {
		cfg.ReplicaConfig = DefaultReplicaConfig
	}
	if cfg.ReplicaConfig.ShardID == 0 {
		cfg.ReplicaConfig.ShardID = 1
	}
	cfg.HostConfig.DeploymentID, err = base36Decode(cfg.ClusterName)
	if err != nil {
		return nil, err
	}
	a = &agent{
		log:         logger.GetLogger(magicPrefix),
		hostConfig:  cfg.HostConfig,
		primeConfig: cfg.ReplicaConfig,
		clusterName: cfg.ClusterName,
		client:      udp.NewClient(magicPrefix, cfg.HostConfig.RaftAddress, cfg.ClusterName),
		hostFS:      os.DirFS(cfg.HostConfig.NodeHostDir),
		clock:       clock.New(),
		shardTypes:  map[string]shardType{},
		status:      AgentStatus_Pending,
		discovery:   cfg.Discovery,
	}
	a.controller = newController(a)
	a.hostConfig.RaftEventListener = newCompositeRaftEventListener(a.controller, a.hostConfig.RaftEventListener)
	return a, nil
}

func (a *agent) Start() (err error) {
	seedList, err := a.discovery.GetSeedList(a)
	if err != nil {
		return
	}
	err = a.startHost(seedList)
	if err != nil {
		err = fmt.Errorf("Failed to start node host: %v %s", err.Error(), strings.Join(seedList, ", "))
		return
	}
	a.setStatus(AgentStatus_Ready)
	var ctx context.Context
	ctx, a.cancel = context.WithCancel(context.Background())
	go func() {
		// Agent listener complies with node join requests
		for {
			err := a.client.Listen(ctx, udp.HandlerFunc(a.udpHandlefunc))
			a.log.Errorf("Error reading UDP: %s", err.Error())
			select {
			case <-ctx.Done():
				a.log.Debugf("Stopped Agent Listener")
				return
			case <-time.After(time.Second):
			}
		}
	}()
	var shardInfo *ShardInfo
	nhInfo := a.host.GetNodeHostInfo(dragonboat.NodeHostInfoOption{true})
	for _, info := range nhInfo.ShardInfoList {
		if info.ShardID == a.primeConfig.ShardID {
			shardInfo = &info
		}
	}
	if shardInfo != nil {
		// Host already has prime replica shard, start and return
		a.primeConfig.ReplicaID = shardInfo.ReplicaID
		a.primeConfig.IsNonVoting = shardInfo.IsNonVoting
		err = a.startReplica()
		return
	}
	reg, _ := a.host.GetNodeHostRegistry()
	var shardView ShardView
	var ok bool
	var t = a.clock.Ticker(time.Second)
	defer t.Stop()
	for {
		shardView, ok = reg.GetShardInfo(a.primeConfig.ShardID)
		if ok {
			a.log.Infof("Prime shard located")
			break
		}
		a.log.Infof("Awaiting cluster init")
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
	for id, nhid := range shardView.Nodes {
		if nhid == a.HostID() {
			// Replica ID found.
			a.primeConfig.ReplicaID = id
			break
		}
	}
	var i int
	var res string
	var args []string
	var index uint64
	if a.primeConfig.ReplicaID == 0 {
		for {
			for _, nhid := range shardView.Nodes {
				sv, _ := reg.GetShardInfo(a.primeConfig.ShardID)
				index = sv.ConfigChangeIndex
				meta, ok := reg.GetMeta(nhid)
				if !ok {
					err = fmt.Errorf("Unable to retrieve node host meta")
					return
				}
				pipe := bytes.LastIndex(meta, []byte(`|`))
				addr := string(meta[pipe+1:])
				voting := "true"
				if len(shardView.Nodes) > minReplicas {
					voting = "false"
				}
				indexStr := fmt.Sprintf("%d", index)
				res, args, err = a.client.Send(joinTimeout, addr, SHARD_JOIN, a.HostID(), indexStr, voting)
				if err != nil {
					continue
				}
				if res == SHARD_JOIN_REFUSED {
					err = fmt.Errorf("Join request refused: %s", args[0])
					return
				}
				if res != SHARD_JOIN_SUCCESS || len(args) < 1 {
					a.log.Warningf("Invalid shard join response from %s: %s (%v)", addr, res, args)
					continue
				}
				i, err = strconv.Atoi(args[0])
				if err != nil || i < 1 {
					a.log.Errorf("Invalid replicaID response from %s: %s", addr, args[0])
					continue
				}
				a.primeConfig.ReplicaID = uint64(i)
				break
			}
			a.log.Warningf("Unable to add replica: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			shardView, _ = reg.GetShardInfo(a.primeConfig.ShardID)
		}
		a.primeConfig.ReplicaID = index
	}
	a.primeConfig.IsNonVoting = len(shardView.Nodes) > minReplicas
	err = a.host.StartReplica(nil, true, fsmFactory(a), a.primeConfig)
	return
}

func (a *agent) udpHandlefunc(cmd string, args ...string) (res []string, err error) {
	switch cmd { // Some out of band system wants this node to begin the initialization process
	case INIT:
		if a.GetStatus() == AgentStatus_Active {
			return []string{INIT_CONFLICT}, fmt.Errorf("Already initialized")
		}
		if err = a.Init(); err != nil {
			return []string{INIT_ERROR}, err
		}
		res = []string{INIT_SUCCESS, a.host.ID()}

	// Another node wants this node to participate in cluster initialization
	case INIT_HOST:
		if res, err = a.client.Validate(cmd, args, 1); err != nil {
			return
		}
		seedList := args[0]
		seeds := strings.Split(seedList, ",")
		var replicaID uint64
		var advertiseAddr = a.GetHostConfig().Gossip.AdvertiseAddress
		for i, gossipAddr := range seeds {
			if gossipAddr == advertiseAddr {
				replicaID = uint64(i + 1)
			}
		}
		if replicaID == 0 {
			return
		}
		err = a.startHost(seeds)
		if err != nil {
			return []string{INIT_HOST_ERROR}, err
		}
		res = []string{INIT_HOST_SUCCESS, strconv.Itoa(int(replicaID)), a.HostID()}

	// Another node wants this node to participate in prime shard initialization
	case INIT_SHARD:
		if res, err = a.client.Validate(cmd, args, 1); err != nil {
			return
		}
		var replicaID uint64
		initialMembers := map[uint64]string{}
		err := json.Unmarshal([]byte(args[0]), &initialMembers)
		if err != nil {
			return []string{INIT_SHARD_ERROR}, err
		}
		for i, v := range initialMembers {
			if v == a.HostID() {
				replicaID = i
			}
		}
		if replicaID < 1 {
			return []string{INIT_SHARD_ERROR}, fmt.Errorf("Node not in initial members")
		}
		a.primeConfig.ReplicaID = replicaID
		err = a.host.StartReplica(initialMembers, false, fsmFactory(a), a.primeConfig)
		if err != nil {
			return []string{INIT_SHARD_ERROR}, fmt.Errorf("Failed to start prime shard during init: %w", err)
		}
		a.setStatus(AgentStatus_Active)
		res = []string{INIT_SHARD_SUCCESS, fmt.Sprintf("%d", replicaID)}

	case SHARD_JOIN:
		if len(args) != 3 {
			err = fmt.Errorf("Incorrect arguments in SHARD_JOIN: %#v", args)
			break
		}
		HostID := args[0]
		indexStr := args[1]
		index, err := parseUint64(indexStr)
		if err != nil {
			res = []string{SHARD_JOIN_ERROR}
			err = fmt.Errorf("Invalid index in SHARD_JOIN: %v", indexStr)
			break
		}
		voting := args[2] == "true"
		if voting {
			err = a.host.SyncRequestAddReplica(raftCtx(), a.primeConfig.ShardID, index, HostID, index)
		} else {
			err = a.host.SyncRequestAddNonVoting(raftCtx(), a.primeConfig.ShardID, index, HostID, index)
		}
		if err != nil {
			res = []string{SHARD_JOIN_ERROR}
			break
		}
		res = []string{SHARD_JOIN_SUCCESS, indexStr}
	default:
		err = fmt.Errorf("Unrecognized command: %s", cmd)
	}
	return
}

func (a *agent) startReplica() (err error) {
	err = a.host.StartReplica(nil, false, fsmFactory(a), a.primeConfig)
	if err != nil {
		return
	}
	nhid := a.HostID()
	reg, _ := a.host.GetNodeHostRegistry()
	meta, ok := reg.GetMeta(nhid)
	if !ok {
		err = fmt.Errorf("Unable to retrieve node host meta")
		return
	}
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	_, err = a.host.SyncPropose(raftCtx(), sess, newCmdHostPut(nhid, meta, HostStatus_Active, keys(a.shardTypes)))
	if err != nil {
		return
	}
	return
}

func (a *agent) GetClusterName() string {
	return a.clusterName
}

func (h *agent) HostExists() bool {
	_, err := h.hostFS.Open("NODEHOST.ID")
	switch err.(type) {
	case nil:
		return true
	case *os.PathError:
		return false
	}
	panic(err)
	return false
}

func (a *agent) GetHost() *dragonboat.NodeHost {
	return a.host
}

func (a *agent) addReplica(nhid string) (replicaID uint64, err error) {
	host, err := a.host.SyncRead(raftCtx(), a.primeConfig.ShardID, newQueryHostGet(nhid))
	if err != nil {
		return
	}
	if host == nil {
		err = a.primeAddHost(nhid)
		if err != nil {
			return
		}
	}
	res, err := a.host.SyncRead(raftCtx(), a.primeConfig.ShardID, newQueryReplicaGet(nhid, a.primeConfig.ShardID))
	if err != nil {
		return
	}
	if res == nil {
		if replicaID, err = a.primeAddReplica(nhid, 0); err != nil {
			return
		}
	} else {
		replicaID = res.(*Replica).ID
	}
	m, err := a.host.SyncGetShardMembership(raftCtx(), a.primeConfig.ShardID)
	if err != nil {
		return
	}
	err = a.host.SyncRequestAddReplica(raftCtx(), a.primeConfig.ShardID, replicaID, nhid, m.ConfigChangeID)
	if err != nil {
		return
	}
	return
}

func (a *agent) parsePeers(peers map[string]string) (replicaID uint64, seedList []string, err error) {
	seedList = make([]string, 0, len(peers))
	for k := range peers {
		seedList = append(seedList, k)
	}
	sort.Strings(seedList)
	for i, gossipAddr := range seedList {
		if gossipAddr == a.hostConfig.Gossip.AdvertiseAddress {
			replicaID = uint64(i + 1)
		}
	}
	if replicaID == 0 {
		err = fmt.Errorf("Node not found")
	}
	return
}

// GetStatus returns the agent status
func (a *agent) GetStatus() AgentStatus {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.status
}

func (a *agent) setStatus(s AgentStatus) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.log.Debugf("Agent Status: %v", s)
	a.status = s
}

// GetSnapshotJson returns a snapshot of the prime shard state machine in JSON format
// TODO - Change snapshot format from JSON to JSON Lines for streaming https://jsonlines.org
func (a *agent) GetSnapshotJson() (b []byte, err error) {
	if a.fsm == nil {
		err = fmt.Errorf("Raft node not found")
		return
	}
	var bb bytes.Buffer
	if err = a.fsm.SaveSnapshot(&bb, nil, nil); err != nil {
		return
	}
	b = bb.Bytes()
	return
}

// GetSnapshot returns a go object representing a snapshot of the prime shard if index does not match
func (a *agent) GetSnapshot(index uint64) (res Snapshot, err error) {
	if a.fsm.index <= index {
		return
	}
	s, err := a.host.SyncRead(raftCtx(), a.primeConfig.ShardID, newQuerySnapshotGet())
	if s != nil {
		res = s.(Snapshot)
	}
	return
}

func (a *agent) setRaftNode(r *fsm) {
	a.fsm = r
}

// Init initializes the cluster - This happens only once, at the beginning of the cluster lifecycle
func (a *agent) Init() (err error) {
	err = a.init(a.discovery.Peers())
	return
}

func (a *agent) init(peers map[string]string) (err error) {
	status := a.GetStatus()
	if status == AgentStatus_Initializing {
		return fmt.Errorf("Agent busy (%s)", status)
	}
	replicaID, seedList, err := a.parsePeers(peers)
	if err != nil {
		return fmt.Errorf("Agent not ready (%w)", err)
	}
	a.log.Infof("Initializing peers %+v", peers)
	a.setStatus(AgentStatus_Initializing)
	defer func() {
		if err == nil {
			a.setStatus(AgentStatus_Active)
		} else {
			a.setStatus(AgentStatus_Pending)
		}
	}()
	// Start nodeHosts w/ collected gossip addresses
	var members = map[uint64]string{}
	var res string
	var args []string
	for _, gossipAddr := range seedList {
		if gossipAddr == a.hostConfig.Gossip.AdvertiseAddress {
			err = a.startHost(seedList)
			if err != nil {
				err = fmt.Errorf("Failed to start node host: %v %s", err.Error(), strings.Join(seedList, ", "))
				return
			}
			members[replicaID] = a.HostID()
			continue
		}
		discoveryAddr := peers[gossipAddr]
		res, args, err = a.client.Send(joinTimeout, discoveryAddr, INIT_HOST, strings.Join(seedList, ","))
		if err != nil {
			return
		}
		if res == INIT_HOST_SUCCESS {
			if len(args) != 2 {
				err = fmt.Errorf("Invalid response from %s / %s: %v", discoveryAddr, gossipAddr, args)
				return
			}
			id, err2 := strconv.Atoi(args[0])
			if err2 != nil {
				err = fmt.Errorf("Invalid replicaID from %s / %s: %s", discoveryAddr, gossipAddr, args[0])
				return
			}
			members[uint64(id)] = args[1]
		} else {
			err = fmt.Errorf("Unrecognized INIT_HOST response from %s / %s: %s %v", discoveryAddr, gossipAddr, res, args)
			return
		}
	}
	memberJson, err := json.Marshal(members)
	if err != nil {
		err = fmt.Errorf("Error marshaling member json: %v", err)
		return
	}
	// Start prime shard replicas
	_, err = a.host.SyncGetShardMembership(raftCtx(), a.primeConfig.ShardID)
	if err != nil {
		for _, gossipAddr := range seedList {
			if gossipAddr == a.hostConfig.Gossip.AdvertiseAddress {
				a.primeConfig.ReplicaID = replicaID
				err = a.host.StartReplica(members, false, fsmFactory(a), a.primeConfig)
				if err != nil {
					err = fmt.Errorf("Failed to start meta shard a: %v", err.Error())
					return
				}
				continue
			}
			discoveryAddr := peers[gossipAddr]
			for i := 0; i < 10; i++ {
				res, args, err = a.client.Send(joinTimeout, discoveryAddr, INIT_SHARD, string(memberJson))
				if err != nil {
					a.log.Warningf("%v", err)
					a.clock.Sleep(time.Second)
					continue
				}
				err = nil
				break
			}
			if res != INIT_SHARD_SUCCESS {
				err = fmt.Errorf("Unrecognized INIT_SHARD response from %s / %s: %s %v", discoveryAddr, gossipAddr, res, args)
				return
			}
		}
	}
	// Initialize prime shard fsm
	for {
		a.log.Debugf("init prime shard")
		if err = a.primeInit(members); err == nil {
			break
		}
		a.log.Debugf("%v", err)
		time.Sleep(time.Second)
	}

	return
}

// primeInit proposes addition of initial cluster state to prime shard
func (a *agent) primeInit(members map[uint64]string) (err error) {
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	_, err = a.host.SyncPropose(raftCtx(), sess, newCmdShardPut(a.primeConfig.ShardID, magicPrefix))
	if err != nil {
		return
	}
	for replicaID, nhid := range members {
		if err = a.primeAddHost(nhid); err != nil {
			return
		}
		if _, err = a.primeAddReplica(nhid, replicaID); err != nil {
			return
		}
	}

	return
}

// primeAddHost proposes addition of host metadata to the prime shard state
func (a *agent) primeAddHost(nhid string) (err error) {
	reg, ok := a.host.GetNodeHostRegistry()
	if !ok {
		err = fmt.Errorf("Unable to retrieve HostRegistry")
		return
	}
	meta, ok := reg.GetMeta(nhid)
	if !ok {
		err = fmt.Errorf("Unable to retrieve node host meta")
		return
	}
	pipe := bytes.LastIndex(meta, []byte(`|`))
	if pipe > 0 {
		meta = meta[:pipe]
	}
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	_, err = a.host.SyncPropose(raftCtx(), sess, newCmdHostPut(nhid, meta, HostStatus_New, nil))
	return
}

// primeAddReplica proposes addition of replica metadata to the prime shard state
func (a *agent) primeAddReplica(nhid string, replicaID uint64) (id uint64, err error) {
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	res, err := a.host.SyncPropose(raftCtx(), sess, newCmdReplicaPut(nhid, a.primeConfig.ShardID, replicaID, false))
	if err == nil {
		id = res.Value
	}
	return
}

// CreateShard creates a shard
func (a *agent) CreateShard(shardTypeName string) (shard *Shard, err error) {
	a.log.Infof("Create shard %s", shardTypeName)
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	res, err := a.host.SyncPropose(raftCtx(), sess, newCmdShardPut(0, shardTypeName))
	if err != nil {
		return
	}
	return &Shard{
		ID:       res.Value,
		Type:     shardTypeName,
		Replicas: map[uint64]string{},
		Status:   "new",
	}, nil
}

// CreateReplica creates a replica
func (a *agent) CreateReplica(shardID uint64, nodeHostID string, isNonVoting bool) (id uint64, err error) {
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	res, err := a.host.SyncPropose(raftCtx(), sess, newCmdReplicaPut(nodeHostID, shardID, 0, isNonVoting))
	if err == nil {
		id = res.Value
	}
	return
}

// RegisterShardType registers a state machine factory for a shard. Call before Start.
func (a *agent) RegisterShardType(shardTypeName string, fn CreateStateMachineFunc, cfg *ReplicaConfig) {
	if cfg == nil {
		cfg = &DefaultReplicaConfig
	}
	a.shardTypes[shardTypeName] = shardType{fn, *cfg}
}

func (a *agent) stopReplica(cfg config.Config) (err error) {
	err = a.host.StopReplica(cfg.ShardID, cfg.ReplicaID)
	if err != nil {
		return fmt.Errorf("Failed to stop replica: %w", err)
	}
	return
}

func (a *agent) HostID() string {
	if a.host != nil {
		return a.host.ID()
	}
	return ""
}

func (a *agent) startHost(seeds []string) (err error) {
	if a.host != nil {
		return nil
	}
	a.hostConfig.AddressByNodeHostID = true
	a.hostConfig.Gossip.Seed = seeds
	a.hostConfig.Gossip.Meta = append(a.hostConfig.Gossip.Meta, []byte(`|`+a.hostConfig.RaftAddress)...)
	a.host, err = dragonboat.NewNodeHost(a.hostConfig)
	if err != nil {
		return
	}

	return
}

func (a *agent) GetHostConfig() HostConfig {
	return a.hostConfig
}

func (a *agent) Stop() {
	a.cancel()
	a.stopReplica(a.primeConfig)
	a.log.Infof("Agent stopped.")
}
