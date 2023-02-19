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
	// "strings"
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

	clock     clock.Clock
	ctx       context.Context
	ctxCancel context.CancelFunc
	fsm       *fsm
	host      *dragonboat.NodeHost
	hostFS    fs.FS
	log       logger.ILogger
	members   map[uint64]string
	mutex     sync.RWMutex
	status    AgentStatus
	wg        sync.WaitGroup
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
	a.ctx, a.ctxCancel = context.WithCancel(context.Background())
	go func() {
		a.wg.Add(1)
		defer a.wg.Done()
		for {
			err := a.client.Listen(a.ctx, udp.HandlerFunc(a.udpHandlefunc))
			a.log.Errorf("Error reading UDP: %s", err.Error())
			select {
			case <-a.ctx.Done():
				a.log.Debugf("Stopped Agent Listener")
				return
			case <-a.clock.After(time.Second):
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
	a.members, err = a.findMembers()
	if err != nil {
		return
	}
	// Start voting prime shard replica
	if !a.primeConfig.IsNonVoting {
		err = a.startReplica(a.members)
		return
	}
	//
	return

	// if host in seedList
	//   if log exists
	//     Rejoin as member
	//   if cluster found
	//     Join as member
	//   else
	//     Find peers
	//     Bootstrap
	// else
	//   if log exists
	//     Rejoin as nonvoting
	//   else
	//     Join as nonvoting

	// If shard listed in log info, start non-voting and read cluster state
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

// findGossip resolves peer raft address list to peer gossip address list
func (a *agent) findGossip() (gossip []string, err error) {
	a.wg.Add(1)
	defer a.wg.Done()
	for {
		gossip = gossip[:0]
		for _, peerRaftAddr := range a.peers {
			if peerRaftAddr == a.hostConfig.RaftAddress {
				gossip = append(gossip, a.hostConfig.Gossip.AdvertiseAddress)
				continue
			}
			res, args, err := a.client.Send(time.Second, peerRaftAddr, PROBE)
			if err == nil && res == PROBE_RESPONSE && len(args) == 1 {
				gossip = append(gossip, args[0])
			} else {
				a.log.Warningf("No probe resp for %s %s %+v %v", peerRaftAddr, res, args, err)
			}
		}
		a.log.Infof("Found %d of %d peers %+v", len(gossip), len(a.peers), gossip)
		if len(gossip) < len(a.peers) {
			select {
			case <-a.ctx.Done():
				err = fmt.Errorf(`Cancelling findGossip (agent stopped)`)
				return
			case <-a.clock.After(time.Second):
			}
			continue
		}
		break
	}
	return
}

// findMembers resolves peer raft address list to replicaID and hostID
func (a *agent) findMembers() (members map[uint64]string, err error) {
	a.wg.Add(1)
	defer a.wg.Done()
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
		// All peers resolved. Start.
		if len(members) == len(a.peers) {
			break
		}
		// All peers unintialized. Init.
		if len(uninitialized) == len(a.peers) {
			for i, raftAddr := range a.peers {
				replicaID := uint64(i + 1)
				members[replicaID] = uninitialized[raftAddr]
				if raftAddr == a.hostConfig.RaftAddress {
					a.primeConfig.ReplicaID = replicaID
				}
			}
			if a.primeConfig.ReplicaID > 0 {
				break
			}
		}
		// Some peers initialized. Resolve.
		if len(members)+len(uninitialized) == len(a.peers) {
			for _, raftAddr := range a.peers {
				if raftAddr == a.hostConfig.RaftAddress {
					continue
				}
				if _, ok := uninitialized[raftAddr]; !ok {
					res, args, err = a.client.Send(time.Second, raftAddr, SHARD_MEMBERS)
					if err != nil {
						a.log.Errorf(`Error sending SHARD_MEMBERS: %v`, err)
						return
					}
					if res != SHARD_MEMBERS_RESPONSE || len(args) != 1 {
						a.log.Warningf(`Invalid response for SHARD_MEMBERS: %s, %+v`, res, args)
						continue
					}
					err = json.Unmarshal([]byte(args[0]), &members)
					if err != nil {
						a.log.Warningf(`Unable to deserialized SHARD_MEMBERS: %s`, args[0])
						continue
					}
					for replicaID, hostID := range members {
						if hostID == a.host.ID() {
							a.primeConfig.ReplicaID = replicaID
							break
						}
					}
					break
				}
			}
			if a.primeConfig.ReplicaID > 0 {
				break
			}
		}
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

	case SHARD_MEMBERS:
		if a.members == nil {
			return
		}
		b, _ := json.Marshal(a.members)
		return []string{SHARD_MEMBERS_RESPONSE, string(b)}, nil

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

func (a *agent) GetHostConfig() HostConfig {
	return a.hostConfig
}

func (a *agent) AddUdpHandler(h udp.HandlerFunc) {
	a.udpHandlers = append(a.udpHandlers, h)
}

func (a *agent) Stop() {
	a.ctxCancel()
	a.stopReplica(a.primeConfig)
	a.wg.Wait()
	a.log.Infof("Agent stopped.")
}
