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
	"github.com/lni/dragonboat/v4/logger"

	"github.com/logbn/zongzi/udp"
	"github.com/logbn/zongzi/util"
)

type AgentConfig struct {
	ClusterName   string
	HostConfig    HostConfig
	Peers         []string
	ReplicaConfig ReplicaConfig
	Secrets       []string
}

type Agent interface {
	AddUdpHandler(h udp.HandlerFunc)
	CreateReplica(shardID uint64, nodeHostID string, isNonVoting bool) (id uint64, err error)
	CreateShard(shardTypeName string) (shard *Shard, err error)
	GetClient() udp.Client
	GetClusterName() string
	GetHost() *dragonboat.NodeHost
	GetHostConfig() HostConfig
	GetHostID() string
	GetSnapshot(index uint64) (*Snapshot, error)
	GetSnapshotJson() ([]byte, error)
	GetStatus() AgentStatus
	Init() error
	Start() error
	Stop()
}

type agent struct {
	client      udp.Client
	clusterName string
	controller  *controller
	hostConfig  HostConfig
	peers       []string
	primeConfig ReplicaConfig
	shardTypes  map[string]shardType
	udpHandlers []udp.HandlerFunc

	cancel context.CancelFunc
	clock  clock.Clock
	fsm    *fsm
	host   *dragonboat.NodeHost
	hostFS fs.FS
	log    logger.ILogger
	mutex  sync.RWMutex
	status AgentStatus
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
	log := logger.GetLogger(magicPrefix)
	a = &agent{
		log:         log,
		hostConfig:  cfg.HostConfig,
		primeConfig: cfg.ReplicaConfig,
		clusterName: cfg.ClusterName,
		client:      udp.NewClient(log, magicPrefix, cfg.HostConfig.RaftAddress, cfg.ClusterName, cfg.Secrets),
		hostFS:      os.DirFS(cfg.HostConfig.NodeHostDir),
		clock:       clock.New(),
		shardTypes:  map[string]shardType{},
		status:      AgentStatus_Pending,
		peers:       cfg.Peers,
	}
	sort.Strings(a.peers)
	a.controller = newController(a)
	a.hostConfig.AddressByNodeHostID = true
	a.hostConfig.Gossip.Meta = append([]byte(a.hostConfig.RaftAddress+`|`), a.hostConfig.Gossip.Meta...)
	a.hostConfig.RaftEventListener = newCompositeRaftEventListener(a.controller, a.hostConfig.RaftEventListener)
	return a, nil
}

func (a *agent) Start() (err error) {
	defer func() {
		if err == nil {
			a.setStatus(AgentStatus_Ready)
		}
	}()
	var ctx context.Context
	ctx, a.cancel = context.WithCancel(context.Background())
	go func() {
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
	if a.hostConfig.Gossip.Seed, err = a.findGossip(); err != nil {
		return
	}
	if a.host, err = dragonboat.NewNodeHost(a.hostConfig); err != nil {
		return
	}
	a.log.Infof(`Started host "%s"`, a.GetHostID())
	nhInfo := a.host.GetNodeHostInfo(dragonboat.NodeHostInfoOption{false})
	a.log.Debugf(`Get node host info: %+v`, nhInfo)
	for _, info := range nhInfo.LogInfo {
		if info.ShardID == a.primeConfig.ShardID {
			a.primeConfig.ReplicaID = info.ReplicaID
			break
		}
	}
	a.primeConfig.IsNonVoting = !util.SliceContains(a.peers, a.hostConfig.RaftAddress)
	members, err := a.findMembers()
	if err != nil {
		return
	}
	if err = a.startReplica(members); err != nil {
		return
	}
	return

	// If host in seedList
	//   If log not exists
	//     If cluster not found
	//       Bootstrap
	//     Else
	//       Join as member
	//   Else
	//     Rejoin as member
	// Else
	//   If log not exists
	//     Join as nonvoting
	//   Else
	//     Rejoin as nonvoting

	// If shard listed in log info, start non-voting and read cluster state
	/*
	 */
	/*
		a.log.Debugf(`Rejoining prime shard`)
		var t = a.clock.Ticker(time.Second)
		defer t.Stop()
		var i int
		var res string
		var args []string
		var index uint64
		if a.primeConfig.ReplicaID == 0 {
			for {
				shardView, _ := reg.GetShardInfo(a.primeConfig.ShardID)
				for _, nhid := range shardView.Nodes {
					sv, _ := reg.GetShardInfo(a.primeConfig.ShardID)
					index = sv.ConfigChangeIndex
					raftAddr, _, err = a.parseMeta(nhid)
					if err != nil {
						a.log.Warningf(err.Error())
						continue
					}
					voting := "true"
					if len(shardView.Nodes) > minReplicas {
						voting = "false"
					}
					indexStr := fmt.Sprintf("%d", index)
					res, args, err = a.client.Send(joinTimeout, raftAddr, SHARD_JOIN, a.GetHostID(), indexStr, voting)
					if err != nil {
						continue
					}
					if res == SHARD_JOIN_REFUSED {
						err = fmt.Errorf("Join request refused: %s", args[0])
						return
					}
					if res != SHARD_JOIN_SUCCESS || len(args) < 1 {
						a.log.Warningf("Invalid shard join response from %s: %s (%v)", raftAddr, res, args)
						continue
					}
					i, err = strconv.Atoi(args[0])
					if err != nil || i < 1 {
						a.log.Errorf("Invalid replicaID response from %s: %s", raftAddr, args[0])
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
			}
			a.primeConfig.ReplicaID = index
		}
		err = a.host.StartReplica(nil, true, fsmFactory(a), a.primeConfig)
		return
	*/
}

func (a *agent) findGossip() (gossip []string, err error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	for {
		gossip = gossip[:0]
		for i, p := range a.peers {
			if p == a.hostConfig.RaftAddress {
				mu.Lock()
				gossip = append(gossip, a.hostConfig.Gossip.AdvertiseAddress)
				mu.Unlock()
				continue
			}
			wg.Add(1)
			go func(i int, addr string) {
				defer wg.Done()
				res, args, err := a.client.Send(time.Second, addr, PROBE)
				if err != nil {
					return
				}
				if res == PROBE_RESPONSE && len(args) == 1 {
					mu.Lock()
					gossip = append(gossip, args[0])
					mu.Unlock()
				}
			}(i, p)
		}
		wg.Wait()
		a.log.Infof("Found %d of %d peers %+v", len(gossip), len(a.peers), gossip)
		if len(gossip) < len(a.peers) {
			a.clock.Sleep(time.Second)
			continue
		}
		break
	}
	return
}

func (a *agent) findMembers() (members map[uint64]string, err error) {
	var res string
	var args []string
	var replicaID int
	var uninitialized map[string]string
	for {
		uninitialized = map[string]string{}
		members = map[uint64]string{}
		for _, raftAddr := range a.peers {
			if raftAddr == a.hostConfig.RaftAddress && a.GetHostID() != "" {
				if a.primeConfig.ReplicaID > 0 {
					members[a.primeConfig.ReplicaID] = a.GetHostID()
				} else {
					uninitialized[raftAddr] = a.GetHostID()
				}
				continue
			}
			res, args, err = a.client.Send(time.Second, raftAddr, SHARD_INFO, strconv.Itoa(int(a.primeConfig.ShardID)))
			if err != nil {
				return
			}
			if res == SHARD_INFO_RESPONSE && len(args) == 2 {
				replicaID, err = strconv.Atoi(args[0])
				if err != nil {
					err = fmt.Errorf("Invalid replicaID: %w, %+v", err, args)
					return
				}
				hostID := args[1]
				if args[0] == "0" {
					uninitialized[raftAddr] = hostID
				} else {
					members[uint64(replicaID)] = hostID
				}
			}
		}
		a.log.Infof("Found %d of %d peers (%d uninitialized)", len(members), len(a.peers), len(uninitialized))
		if len(members) == len(a.peers) {
			break
		}
		if len(uninitialized) == len(a.peers) {
			for i, raftAddr := range a.peers {
				replicaID := uint64(i + 1)
				members[replicaID] = uninitialized[raftAddr]
				if raftAddr == a.hostConfig.RaftAddress {
					a.primeConfig.ReplicaID = replicaID
				} else {
					res, args, err = a.client.Send(time.Second, raftAddr, PROPOSE_INIT, fmt.Sprintf(`%d`, replicaID))
					if err != nil {
						return
					}
				}
			}
			if a.primeConfig.ReplicaID > 0 {
				break
			}
		}
		// TODO - Handle join case
		// TODO - Handle quorum case
		a.clock.Sleep(time.Second)
	}
	return
}

func (a *agent) udpHandlefunc(cmd string, args ...string) (res []string, err error) {
	switch cmd {
	case PROBE:
		return []string{PROBE_RESPONSE, a.hostConfig.Gossip.AdvertiseAddress}, nil

	case SHARD_INFO:
		if a.host == nil {
			return
		}
		return []string{SHARD_INFO_RESPONSE, strconv.Itoa(int(a.primeConfig.ReplicaID)), a.host.ID()}, nil

	case PROPOSE_INIT:
		if a.host == nil {
			return
		}
		if res, err = a.client.Validate(cmd, args, 1); err != nil {
			return
		}
		var i int
		i, err = strconv.Atoi(args[0])
		if err != nil {
			err = fmt.Errorf(`Invalid replicaID "%s" %w`, args[0], err)
			return
		}
		a.primeConfig.ReplicaID = uint64(i)
		return []string{PROPOSE_INIT_SUCCESS}, nil

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
		for _, h := range a.udpHandlers {
			res, err = h(cmd, args...)
			if res != nil || err != nil {
				break
			}
		}
		if res == nil && err == nil {
			err = fmt.Errorf("Unrecognized command: %s", cmd)
		}
	}
	return
}

func (a *agent) startReplica(members map[uint64]string) (err error) {
	a.log.Debugf("Starting Replica %+v", members)
	err = a.host.StartReplica(members, false, fsmFactory(a), a.primeConfig)
	if err == dragonboat.ErrShardAlreadyExist {
		err = nil
	}
	if err != nil {
		err = fmt.Errorf(`startReplica: %w`, err)
		return
	}
	nhid := a.host.ID()
	raftAddr, meta, err := a.parseMeta(nhid)
	if err != nil {
		return
	}
	var req *dragonboat.RequestState
	for {
		req, err = a.host.ReadIndex(a.primeConfig.ShardID, raftTimeout)
		if err != nil || req == nil {
			a.clock.Sleep(time.Second)
			continue
		}
		res := <-req.ResultC()
		a.log.Debugf(`%+v`, res.GetResult())
		if !res.Completed() {
			a.log.Infof(`Waiting for other nodes`)
			a.clock.Sleep(time.Second)
			continue
		}
		req.Release()
		break
	}
	a.log.Debugf(`Updating host %s %s %s`, nhid, raftAddr, string(meta))
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	cmd := newCmdHostPut(nhid, raftAddr, meta, HostStatus_Active, util.Keys(a.shardTypes))
	_, err = a.host.SyncPropose(raftCtx(), sess, cmd)
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
	seedList = util.Keys(peers)
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

func (a *agent) parseMeta(nhid string) (raftAddr string, meta []byte, err error) {
	reg, ok := a.host.GetNodeHostRegistry()
	if !ok {
		err = fmt.Errorf("Unable to retrieve HostRegistry")
		return
	}
	meta, ok = reg.GetMeta(nhid)
	if !ok {
		err = fmt.Errorf("Unable to retrieve node host meta (%s)", nhid)
		return
	}
	parts := bytes.Split(meta, []byte(`|`))
	raftAddr = string(parts[0])
	meta = parts[1]
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
func (a *agent) GetSnapshot(index uint64) (res *Snapshot, err error) {
	if a.fsm.index <= index {
		return
	}
	s, err := a.host.SyncRead(raftCtx(), a.primeConfig.ShardID, newQuerySnapshotGet())
	if s != nil {
		res = s.(*Snapshot)
	}
	return
}

func (a *agent) setRaftNode(r *fsm) {
	a.fsm = r
}

// Init initializes the cluster - This happens only once, at the beginning of the cluster lifecycle
func (a *agent) Init() (err error) {
	// err = a.init(a.discovery.Peers())
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
			members[replicaID] = a.GetHostID()
			continue
		}
		raftAddr := peers[gossipAddr]
		res, args, err = a.client.Send(joinTimeout, raftAddr, INIT_HOST, strings.Join(seedList, ","))
		if err != nil {
			return
		}
		if res == INIT_HOST_SUCCESS {
			if len(args) != 2 {
				err = fmt.Errorf("Invalid response from %s / %s: %v", raftAddr, gossipAddr, args)
				return
			}
			id, err2 := strconv.Atoi(args[0])
			if err2 != nil {
				err = fmt.Errorf("Invalid replicaID from %s / %s: %s", raftAddr, gossipAddr, args[0])
				return
			}
			members[uint64(id)] = args[1]
		} else {
			err = fmt.Errorf("Unrecognized INIT_HOST response from %s / %s: %s %v", raftAddr, gossipAddr, res, args)
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
					err = fmt.Errorf("Failed to start prime shard a: %v", err.Error())
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
		a.clock.Sleep(time.Second)
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
	raftAddr, meta, err := a.parseMeta(nhid)
	if err != nil {
		return
	}
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	cmd := newCmdHostPut(nhid, raftAddr, meta, HostStatus_New, nil)
	_, err = a.host.SyncPropose(raftCtx(), sess, cmd)
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

func (a *agent) stopReplica(cfg ReplicaConfig) (err error) {
	err = a.host.StopReplica(cfg.ShardID, cfg.ReplicaID)
	if err != nil {
		return fmt.Errorf("Failed to stop replica: %w", err)
	}
	return
}

func (a *agent) GetHostID() string {
	if a.host != nil {
		return a.host.ID()
	}
	return ""
}

func (a *agent) GetClient() udp.Client {
	return a.client
}

func (a *agent) startHost(gossipAddrs []string) (err error) {
	if a.host != nil {
		return nil
	}
	a.hostConfig.AddressByNodeHostID = true
	a.hostConfig.Gossip.Seed = gossipAddrs
	a.hostConfig.Gossip.Meta = append([]byte(a.hostConfig.RaftAddress+`|`), a.hostConfig.Gossip.Meta...)
	a.host, err = dragonboat.NewNodeHost(a.hostConfig)
	if err != nil {
		return
	}

	return
}

func (a *agent) GetHostConfig() HostConfig {
	return a.hostConfig
}

func (a *agent) AddUdpHandler(h udp.HandlerFunc) {
	a.udpHandlers = append(a.udpHandlers, h)
}

func (a *agent) Stop() {
	a.cancel()
	a.stopReplica(a.primeConfig)
	a.log.Infof("Agent stopped.")
}
