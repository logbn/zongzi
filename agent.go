package zongzi

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/logger"

	"github.com/logbn/zongzi/internal"
)

type Agent struct {
	advertiseAddress string
	bindAddress      string
	clusterName      string
	controller       *controller
	fsm              *fsm
	grpcClientPool   *grpcClientPool
	grpcServer       *grpcServer
	host             *dragonboat.NodeHost
	hostConfig       HostConfig
	log              logger.ILogger
	members          map[uint64]string
	peers            []string
	replicaConfig    ReplicaConfig
	secrets          []string
	shardTypes       map[string]shardType
	status           AgentStatus

	clock     clock.Clock
	ctx       context.Context
	ctxCancel context.CancelFunc
	mutex     sync.RWMutex
	wg        sync.WaitGroup
}

func NewAgent(clusterName string, peers []string, opts ...AgentOption) (a *Agent, err error) {
	if !regexp.MustCompile(ClusterNameRegex).MatchString(clusterName) {
		err = fmt.Errorf(`%v: (%s)`, ErrClusterNameInvalid, clusterName)
		return
	}
	log := logger.GetLogger(projectName)
	sort.Strings(peers)
	a = &Agent{
		clock:       clock.New(),
		clusterName: clusterName,
		log:         log,
		peers:       peers,
		shardTypes:  map[string]shardType{},
		status:      AgentStatus_Pending,
	}
	for _, opt := range append([]AgentOption{
		WithApiAddress(DefaultApiAddress),
		WithGossipAddress(DefaultGossipAddress),
		WithHostConfig(DefaultHostConfig),
		WithReplicaConfig(DefaultReplicaConfig),
	}, opts...) {
		if err = opt(a); err != nil {
			return
		}
	}
	a.controller = newController(a)
	a.replicaConfig.ShardID = 0
	a.hostConfig.DeploymentID = mustBase36Decode(clusterName)
	a.hostConfig.AddressByNodeHostID = true
	a.hostConfig.Gossip.Meta = append(a.hostConfig.Gossip.Meta, []byte(`|`+a.advertiseAddress)...)
	a.hostConfig.RaftEventListener = newCompositeRaftEventListener(a.controller, a.hostConfig.RaftEventListener)
	a.grpcClientPool = newGrpcClientPool(1e4, a.secrets)
	a.grpcServer = newGrpcServer(a.bindAddress, a.secrets)
	return a, nil
}

func (a *Agent) Start() (err error) {
	defer func() {
		if err == nil {
			a.setStatus(AgentStatus_Ready)
			a.controller.Start()
		}
	}()
	a.ctx, a.ctxCancel = context.WithCancel(context.Background())
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for {
			if err := a.grpcServer.Start(a); err != nil {
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
		if info.ShardID == a.replicaConfig.ShardID {
			a.replicaConfig.ReplicaID = info.ReplicaID
			break
		}
	}
	a.replicaConfig.IsNonVoting = !sliceContains(a.peers, a.advertiseAddress)
	a.members, err = a.findMembers()
	if err != nil {
		return
	}
	a.log.Debugf(`Members: %+v`, a.members)
	// Start voting prime shard replica
	if !a.replicaConfig.IsNonVoting {
		err = a.startReplica(a.members)
		return
	}
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
		if a.replicaConfig.ReplicaID == 0 {
			for {
				shardView, _ := reg.GetShardInfo(a.replicaConfig.ShardID)
				for _, nhid := range shardView.Nodes {
					sv, _ := reg.GetShardInfo(a.replicaConfig.ShardID)
					index = sv.ConfigChangeIndex
					_, apiAddr, err = a.parseMeta(nhid)
					if err != nil {
						a.log.Warningf(err.Error())
						continue
					}
					res, err = a.grpcClientPool.get(apiAddr).Join(&internal.JoinRequest{
						Index: index,
						ReplicaID: index,
						HostID: a.GetHostID(),
						IsNonVoting: len(shardView.Nodes) >= minReplicas,
					})
					if err != nil {
						continue
					}
					if res.Value < 1 {
						err = fmt.Errorf("Join request refused: %s", res.Error)
						return
					}
					a.replicaConfig.ReplicaID = res.Value
					break
				}
				a.log.Warningf("Unable to add replica: %v", err)
				select {
				case <-ctx.Done():
					return
				case <-t.C:
				}
			}
			a.replicaConfig.ReplicaID = index
		}
		err = a.host.StartReplica(nil, true, fsmFactory(a), a.replicaConfig)
		return
	*/
}

// GetStatus returns the agent status
func (a *Agent) GetStatus() AgentStatus {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.status
}

// Read executes a callback function passing a reference to the state machine's internal cluster state.
//
//	err := agent.Read(func(s *State) error {
//	    log.Println(s.Index)
//	    return nil
//	})
//
// Callback function MAY NOT write to ANY fields of *[State]. Doing so WILL corrupt the state machine.
//
// Callback function WILL block writes to the state machine.
//
// Linear reads are supported to achieve "Read Your Writes" consistency.
//
// Read is thread safe.
func (a *Agent) Read(fn func(*State), linear ...bool) error {
	if len(linear) > 0 && linear[0] {
		if err := a.readIndex(); err != nil {
			return err
		}
	}
	a.fsm.state.mutex.RLock()
	defer a.fsm.state.mutex.RUnlock()
	fn(a.fsm.state)
	return nil
}

// CreateShard creates a shard
func (a *Agent) CreateShard(uri string) (shard *Shard, err error) {
	a.log.Infof("Create shard %s", uri)
	res, err := a.primePropose(newCmdShardPost(uri))
	if err != nil {
		return
	}
	return &Shard{
		ID:     res.Value,
		Type:   uri,
		Status: ShardStatus_New,
	}, nil
}

// CreateReplica creates a replica
func (a *Agent) CreateReplica(shardID uint64, nodeHostID string, isNonVoting bool) (id uint64, err error) {
	res, err := a.primePropose(newCmdReplicaPut(nodeHostID, shardID, 0, isNonVoting))
	if err == nil {
		id = res.Value
	}
	return
}

// RegisterStateMachine registers a shard type. Call before Start.
func (a *Agent) RegisterStateMachine(uri, version string, factory StateMachineFactory, config ...ReplicaConfig) {
	cfg := DefaultReplicaConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	a.shardTypes[uri] = shardType{
		Config:  cfg,
		Factory: factory,
		Uri:     uri,
		Version: version,
	}
}

func (a *Agent) GetHostID() (id string) {
	if a.host != nil {
		return a.host.ID()
	}
	return ""
}

func (a *Agent) GetReplicaClient(replicaID uint64) (c *ReplicaClient) {
	a.Read(func(s *State) {
		replica, ok := s.Replicas.Get(replicaID)
		if ok && replica.ShardID > 0 {
			c = newReplicaClient(replica, a)
		}
	})
	return
}

func (a *Agent) GetHostClient(hostID string) (c *HostClient) {
	a.Read(func(s *State) {
		host, ok := s.Hosts.Get(hostID)
		if ok {
			c = newHostClient(host, a)
		}
	})
	return
}

func (a *Agent) Stop() {
	a.ctxCancel()
	a.wg.Wait()
	a.stopReplica(a.replicaConfig)
	a.grpcClientPool.Close()
	a.log.Infof("Agent stopped.")
}

func (a *Agent) setStatus(s AgentStatus) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.log.Debugf("Agent Status: %v", s)
	a.status = s
}

func (a *Agent) stopReplica(cfg ReplicaConfig) (err error) {
	err = a.host.StopReplica(cfg.ShardID, cfg.ReplicaID)
	if err != nil {
		return fmt.Errorf("Failed to stop replica: %w", err)
	}
	return
}

// findGossip resolves peer raft address list to peer gossip address list
func (a *Agent) findGossip() (gossip []string, err error) {
	a.wg.Add(1)
	defer a.wg.Done()
	for {
		gossip = gossip[:0]
		for _, peerApiAddr := range a.peers {
			if peerApiAddr == a.advertiseAddress {
				gossip = append(gossip, a.hostConfig.Gossip.AdvertiseAddress)
				continue
			}
			res, err := a.grpcClientPool.get(peerApiAddr).Probe(raftCtx(), &internal.ProbeRequest{})
			if err == nil && res != nil {
				gossip = append(gossip, res.GossipAdvertiseAddress)
			} else {
				a.log.Warningf("No probe response for %s %+v %v", peerApiAddr, res, err)
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
func (a *Agent) findMembers() (members map[uint64]string, err error) {
	a.wg.Add(1)
	defer a.wg.Done()
	for {
		members = map[uint64]string{}
		var uninitialized = map[string]string{}
		for _, apiAddr := range a.peers {
			var info *internal.InfoResponse
			if apiAddr == a.advertiseAddress && a.GetHostID() != "" {
				info = &internal.InfoResponse{
					ReplicaId: a.replicaConfig.ReplicaID,
					HostId:    a.GetHostID(),
				}
			} else {
				info, err = a.grpcClientPool.get(apiAddr).Info(raftCtx(), &internal.InfoRequest{})
				if err != nil {
					return
				}
			}
			if len(info.HostId) == 0 {
				continue
			}
			if info.ReplicaId == 0 {
				uninitialized[apiAddr] = info.HostId
			} else {
				members[info.ReplicaId] = info.HostId
			}
		}
		a.log.Infof("Found %d of %d peers (%d uninitialized)", len(members), len(a.peers), len(uninitialized))
		// All peers resolved. Start.
		if len(members) == len(a.peers) {
			break
		}
		// All peers unintialized. Init.
		if len(uninitialized) == len(a.peers) {
			for i, apiAddr := range a.peers {
				replicaID := uint64(i + 1)
				members[replicaID] = uninitialized[apiAddr]
				if apiAddr == a.advertiseAddress {
					a.replicaConfig.ReplicaID = replicaID
				}
			}
			if a.replicaConfig.ReplicaID > 0 {
				break
			}
		}
		// Some peers initialized. Retrieve member list from initialized host.
		if len(members)+len(uninitialized) == len(a.peers) {
			for _, apiAddr := range a.peers {
				if apiAddr == a.advertiseAddress {
					continue
				}
				if _, ok := uninitialized[apiAddr]; !ok {
					res, err := a.grpcClientPool.get(apiAddr).Members(raftCtx(), &internal.MembersRequest{})
					if err != nil {
						return nil, err
					}
					a.log.Debugf("Get Members: %+v", *res)
					for replicaID, hostID := range res.Members {
						members[replicaID] = hostID
						if hostID == a.host.ID() {
							a.replicaConfig.ReplicaID = replicaID
							break
						}
					}
					break
				}
			}
			if a.replicaConfig.ReplicaID > 0 && len(members) == len(a.peers) {
				break
			}
		}
		a.clock.Sleep(time.Second)
	}
	return
}

func (a *Agent) startReplica(members map[uint64]string) (err error) {
	a.log.Debugf("Starting Replica %+v", members)
	err = a.host.StartReplica(members, false, fsmFactory(a), a.replicaConfig)
	if err == dragonboat.ErrShardAlreadyExist {
		err = nil
	}
	if err != nil {
		err = fmt.Errorf(`startReplica: %w`, err)
		return
	}
	nhid := a.host.ID()
	meta, apiAddr, err := a.parseMeta(nhid)
	if err != nil {
		return
	}
	if err = a.readIndex(); err != nil {
		return
	}
	a.log.Debugf(`Updating host %s %s %s`, nhid, apiAddr, string(meta))
	shardTypes := keys(a.shardTypes)
	sort.Strings(shardTypes)
	_, err = a.primePropose(newCmdHostPut(nhid, apiAddr, meta, HostStatus_Active, shardTypes))
	if err != nil {
		return
	}
	return
}

func (a *Agent) readIndex() (err error) {
	var rs *dragonboat.RequestState
	for {
		rs, err = a.host.ReadIndex(a.replicaConfig.ShardID, raftTimeout)
		if err != nil || rs == nil {
			a.log.Warningf(`Error reading prime shard index: %v`, err)
			a.clock.Sleep(time.Second)
			continue
		}
		res := <-rs.ResultC()
		a.log.Debugf(`%+v`, res.GetResult())
		if !res.Completed() {
			a.log.Infof(`Waiting for other nodes`)
			rs.Release()
			a.clock.Sleep(time.Second)
			continue
		}
		rs.Release()
		break
	}
	return
}

func (a *Agent) addReplica(nhid string) (replicaID uint64, err error) {
	host, err := a.host.SyncRead(raftCtx(), a.replicaConfig.ShardID, newQueryHostGet(nhid))
	if err != nil {
		return
	}
	if host == nil {
		err = a.primeAddHost(nhid)
		if err != nil {
			return
		}
	}
	for id, r := range host.(Host).Replicas {
		if r.ShardID == a.replicaConfig.ShardID {
			replicaID = uint64(id)
			break
		}
	}
	if replicaID == 0 {
		if replicaID, err = a.primeAddReplica(nhid, 0); err != nil {
			return
		}
	}
	m, err := a.host.SyncGetShardMembership(raftCtx(), a.replicaConfig.ShardID)
	if err != nil {
		return
	}
	err = a.host.SyncRequestAddReplica(raftCtx(), a.replicaConfig.ShardID, replicaID, nhid, m.ConfigChangeID)
	if err != nil {
		return
	}
	return
}

func (a *Agent) parseMeta(nhid string) (meta []byte, apiAddr string, err error) {
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
	i := bytes.LastIndexByte(meta, '|')
	apiAddr = string(meta[i+1:])
	meta = meta[:i]
	return
}

func (a *Agent) primePropose(cmd []byte) (Result, error) {
	return a.host.SyncPropose(raftCtx(), a.host.GetNoOPSession(a.replicaConfig.ShardID), cmd)
}

// primeInit proposes addition of initial cluster state to prime shard
func (a *Agent) primeInit(members map[uint64]string) (err error) {
	_, err = a.primePropose(newCmdShardPut(a.replicaConfig.ShardID, projectName))
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
func (a *Agent) primeAddHost(nhid string) (err error) {
	meta, addr, err := a.parseMeta(nhid)
	if err != nil {
		return
	}
	_, err = a.primePropose(newCmdHostPut(nhid, addr, meta, HostStatus_New, nil))
	return
}

// primeAddReplica proposes addition of replica metadata to the prime shard state
func (a *Agent) primeAddReplica(nhid string, replicaID uint64) (id uint64, err error) {
	res, err := a.primePropose(newCmdReplicaPut(nhid, a.replicaConfig.ShardID, replicaID, false))
	if err == nil {
		id = res.Value
	}
	return
}
