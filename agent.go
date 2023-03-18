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
)

type ApiConfig struct {
	AdvertiseAddress string
	BindAddress      string
}

type Config struct {
	Api           ApiConfig
	ClusterName   string
	Host          HostConfig
	Peers         []string
	ReplicaConfig ReplicaConfig
	Secrets       []string
}

type Agent interface {
	CreateReplica(shardID uint64, nodeHostID string, isNonVoting bool) (*Replica, error)
	CreateShard(shardTypeName string) (shard *Shard, err error)
	GetClient() Client
	GetClusterName() string
	GetHost() *dragonboat.NodeHost
	GetHostConfig() HostConfig
	GetHostID() string
	GetPrimeConfig() ReplicaConfig
	GetSnapshot(index uint64) (*Snapshot, error)
	GetSnapshotJson() ([]byte, error)
	GetStatus() AgentStatus
	FindReplica(shardID, replicaID uint64) (*Replica, error)
	RegisterStateMachine(name string, fn SMFactory, cfg *ReplicaConfig)
	Start() error
	Stop()
}

type agent struct {
	apiConfig   ApiConfig
	clock       clock.Clock
	clusterName string
	controller  *controller
	ctx         context.Context
	ctxCancel   context.CancelFunc
	fsm         *fsm
	grpcClient  *grpcClient
	grpcServer  *grpcServer
	host        *dragonboat.NodeHost
	hostConfig  HostConfig
	hostFS      fs.FS
	log         logger.ILogger
	members     map[uint64]string
	mutex       sync.RWMutex
	peers       []string
	primeConfig ReplicaConfig
	shardTypes  map[string]shardType
	status      AgentStatus
	wg          sync.WaitGroup
}

type shardType struct {
	Factory CreateStateMachineFunc
	Config  ReplicaConfig
}

func NewAgent(cfg AgentConfig) (a *agent, err error) {
	if cfg.ReplicaConfig.ElectionRTT == 0 {
		cfg.ReplicaConfig = DefaultReplicaConfig
	}
	cfg.ReplicaConfig.ShardID = 0
	cfg.HostConfig.DeploymentID, err = base36Decode(cfg.ClusterName)
	if err != nil {
		return nil, err
	}
	log := logger.GetLogger(magicPrefix)
	a = &agent{
		apiConfig:   cfg.ApiConfig,
		clock:       clock.New(),
		clusterName: cfg.ClusterName,
		hostConfig:  cfg.HostConfig,
		hostFS:      os.DirFS(cfg.HostConfig.NodeHostDir),
		log:         log,
		peers:       cfg.Peers,
		primeConfig: cfg.ReplicaConfig,
		shardTypes:  map[string]shardType{},
		status:      AgentStatus_Pending,
	}
	sort.Strings(a.peers)
	a.controller = newController(a)
	a.grpcClient = newGrpcClient(a, 1e4)
	a.grpcServer = newGrpcServer(a, cfg.BindAddress, cfg.Secrets)
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
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for {
			if err := a.grpcServer.Start(a.ctx); err != nil {
				a.log.Errorf("Error starting gRPC server: %s", err.Error())
			}
			select {
			case <-a.ctx.Done():
				a.log.Debugf("Stopped gRPC Server")
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
	a.primeConfig.IsNonVoting = !sliceContains(a.peers, a.hostConfig.RaftAddress)
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
			res, err := a.grpcClient.get(peerRaftAddr).Probe(raftCtx(), nil)
			if err == nil && res != nil {
				gossip = append(gossip, res.GossipAdvertiseAddress)
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
			res, args, err = a.client.Send(time.Second, raftAddr, SHARD_INFO)
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
	addr, meta, err := a.parseMeta(nhid)
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
	cmd := newCmdHostPut(nhid, raftAddr, meta, HostStatus_Active, keys(a.shardTypes))
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
	for id, shardID := range host.(Host).Replicas {
		if shardID == a.primeConfig.ShardID {
			replicaID = id
			break
		}
	}
	if replicaID == 0 {
		if replicaID, err = a.primeAddReplica(nhid, 0); err != nil {
			return
		}
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

func (a *agent) parseMeta(nhid string) (apiAddr, raftAddr string, meta []byte, err error) {
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
	parts := bytes.SplitN(meta, []byte(`|`), 3)
	if len(parts) < 3 {
		err = fmt.Errorf("Malformed meta: %s", string(meta))
		return
	}
	apiAddr = string(parts[0])
	raftAddr = string(parts[1])
	meta = parts[2]
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
	addr, meta, err := a.parseMeta(nhid)
	if err != nil {
		return
	}
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	cmd := newCmdHostPut(nhid, addr, meta, HostStatus_New, nil)
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
	return &pb.Shard{
		ID:     res.Value,
		Type:   shardTypeName,
		Status: pb.Shard_STATUS_NEW,
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

// RegisterStateMachine registers a shard type. Call before Start.
func (a *agent) RegisterStateMachine(uri, version string, smf SMFactory, svc Service, cfg *ReplicaConfig) {
	if cfg == nil {
		cfg = &DefaultReplicaConfig
	}
	a.shardTypes[uri] = shardType{version, smf, qs, *cfg}
}

func (a *agent) stopReplica(cfg ReplicaConfig) (err error) {
	err = a.host.StopReplica(cfg.ShardID, cfg.ReplicaID)
	if err != nil {
		return fmt.Errorf("Failed to stop replica: %w", err)
	}
	return
}

func (a *agent) GetHostID() (res string) {
	if a.host != nil {
		res = a.host.ID()
	}
	return
}

func (a *agent) GetHostConfig() HostConfig {
	return a.hostConfig
}

func (a *agent) GetPrimeConfig() ReplicaConfig {
	return a.primeConfig
}

func (a *agent) GetMembers() map[uint64]string {
	return a.members
}

func (a *agent) Stop() {
	a.ctxCancel()
	a.wg.Wait()
	a.stopReplica(a.primeConfig)
	a.grpcClient.Stop()
	a.log.Infof("Agent stopped.")
}
