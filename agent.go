package zongzi

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
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
		opt(a)
	}
	a.controller = newController(a)
	a.replicaConfig.ShardID = ZongziShardID
	a.hostConfig.DeploymentID = mustBase36Decode(clusterName)
	a.hostConfig.AddressByNodeHostID = true
	a.hostConfig.Gossip.Meta = append(a.hostConfig.Gossip.Meta, []byte(`|`+a.advertiseAddress)...)
	a.hostConfig.RaftEventListener = newCompositeRaftEventListener(a.controller, a.hostConfig.RaftEventListener)
	a.grpcClientPool = newGrpcClientPool(1e4, a.secrets)
	a.grpcServer = newGrpcServer(a.bindAddress, a.secrets)
	return a, nil
}

func (a *Agent) Start() (err error) {
	var init bool
	defer func() {
		if err == nil {
			a.setStatus(AgentStatus_Ready)
			a.controller.Start()
		} else {
			a.grpcServer.Stop()
		}
	}()
	a.ctx, a.ctxCancel = context.WithCancel(context.Background())
	// Start gRPC server
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer a.log.Debugf("Stopped gRPC Server")
		for {
			if err := a.grpcServer.Start(a); err != nil {
				a.log.Errorf("Error starting gRPC server: %s", err.Error())
			}
			select {
			case <-a.ctx.Done():
				return
			case <-a.clock.After(waitPeriod):
			}
		}
	}()
	// Resolve member gossip addresses
	a.hostConfig.Gossip.Seed, err = a.resolvePeerGossipSeed()
	if err != nil {
		a.log.Errorf(`Failed to resolve gossip seeds: %v`, err)
		return
	}
	// Start node host
	if a.host, err = dragonboat.NewNodeHost(a.hostConfig); err != nil {
		a.log.Errorf(`Failed to start host: %v`, err)
		return
	}
	a.log.Infof(`Started host "%s"`, a.HostID())
	// Find prime replicaID
	a.replicaConfig.ReplicaID = a.findLocalReplicaID(a.replicaConfig.ShardID)
	existing := a.replicaConfig.ReplicaID > 0
	a.replicaConfig.IsNonVoting = !sliceContains(a.peers, a.advertiseAddress)
	// Resolve prime member replicaIDs
	a.members, init, err = a.resolvePrimeMembership()
	if err != nil {
		a.log.Errorf(`Failed to resolve prime membership: %v`, err)
		return
	}
	if a.replicaConfig.ReplicaID == 0 {
		a.setStatus(AgentStatus_Joining)
		for {
			a.replicaConfig.ReplicaID, err = a.joinPrimeShard()
			if err != nil {
				a.log.Warningf(`Failed to join prime shard: %v`, err)
				a.clock.Sleep(waitPeriod)
				continue
			}
			break
		}
		for {
			err = a.startPrimeReplica(nil, true)
			if err != nil {
				a.log.Warningf("Error startPrimeReplica: (%s) %s", AgentStatus_Joining, err.Error())
				a.clock.Sleep(waitPeriod)
				continue
			}
			break
		}
	} else if !existing {
		if init {
			a.setStatus(AgentStatus_Initializing)
		} else {
			a.setStatus(AgentStatus_Joining)
		}
		err = a.startPrimeReplica(a.members, false)
		if err != nil {
			a.log.Errorf(`Failed to startPrimeReplica: %v`, err)
			return
		}
		if init {
			err = a.primeInit(a.members)
			if err != nil {
				a.log.Errorf(`Failed to primeInit: %v`, err)
				return
			}
		} else {
			err = a.primeInitAwait()
			if err != nil {
				a.log.Errorf(`Failed to primeInitAwait: %v`, err)
				return
			}
		}
	} else {
		a.setStatus(AgentStatus_Rejoining)
		err = a.startPrimeReplica(nil, false)
		if err != nil {
			a.log.Errorf(`Failed to startPrimeReplica: %v`, err)
			return
		}
	}
	err = a.updateHost()
	if err != nil {
		a.log.Errorf(`Failed to updateHost: %v`, err)
		return
	}
	err = a.updateReplica()
	if err != nil {
		a.log.Errorf(`Failed to updateReplica: %v`, err)
		return
	}
	// Start non-prime shards
	return
}

// Status returns the agent status
func (a *Agent) Status() AgentStatus {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.status
}

// Read executes a callback function passing a reference to the state machine's internal cluster state.
//
//	err := agent.Read(func(s *State) error {
//	    log.Println(s.Index())
//	    return nil
//	})
//
// Linear reads are supported to achieve "Read Your Writes" consistency following a proposal.
//
// Read is thread safe.
func (a *Agent) Read(fn func(State), linear ...bool) (err error) {
	if len(linear) > 0 && linear[0] {
		err = a.readIndex(a.replicaConfig.ShardID)
		if err != nil {
			return
		}
	}
	fn(a.fsm.state.withTxn(false))
	return
}

// CreateShard creates a new shard
func (a *Agent) CreateShard(uri, version string) (shard Shard, err error) {
	a.log.Infof("Create shard %s@%s", uri, version)
	res, err := a.primePropose(newCmdShardPost(uri, version))
	if err != nil {
		return
	}
	return Shard{
		ID:      res.Value,
		Type:    uri,
		Version: version,
		Status:  ShardStatus_New,
	}, nil
}

// CreateReplica creates a replica
func (a *Agent) CreateReplica(shardID uint64, nodeHostID string, isNonVoting bool) (id uint64, err error) {
	res, err := a.primePropose(newCmdReplicaPost(nodeHostID, shardID, isNonVoting))
	if err == nil {
		id = res.Value
	}
	a.log.Infof("[%05d:%05d] Created replica %s, %v", shardID, id, nodeHostID, isNonVoting)
	return
}

// RegisterStateMachine registers a non-persistent shard type. Call before Starting agent.
func (a *Agent) RegisterStateMachine(uri, version string, factory StateMachineFactory, config ...ReplicaConfig) {
	cfg := DefaultReplicaConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	a.shardTypes[uri] = shardType{
		Config:              cfg,
		StateMachineFactory: factory,
		Uri:                 uri,
		Version:             version,
	}
}

// RegisterPersistentStateMachine registers a persistent shard type. Call before Starting agent.
func (a *Agent) RegisterPersistentStateMachine(uri, version string, factory PersistentStateMachineFactory, config ...ReplicaConfig) {
	cfg := DefaultReplicaConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	a.shardTypes[uri] = shardType{
		Config:                        cfg,
		PersistentStateMachineFactory: factory,
		Uri:                           uri,
		Version:                       version,
	}
}

// GetHostClient returns a HostClient for a specific host.
func (a *Agent) GetHostClient(hostID string) (c *HostClient) {
	a.Read(func(s State) {
		host, ok := s.HostGet(hostID)
		if ok {
			c = newHostClient(host, a)
		}
	}, true)
	return
}

// HostID returns host ID if host is initialized, otherwise empty string.
func (a *Agent) HostID() (id string) {
	if a.host != nil {
		return a.host.ID()
	}
	return ""
}

// GetReplicaClient returns a ReplicaClient for a specific host.
func (a *Agent) GetReplicaClient(replicaID uint64) (c *ReplicaClient) {
	a.Read(func(s State) {
		replica, ok := s.ReplicaGet(replicaID)
		if !ok || replica.ShardID == 0 {
			return
		}
		host, ok := s.HostGet(replica.HostID)
		if !ok {
			return
		}
		a.log.Debugf(`[%05d:%05d] New replica client %s isNonVoting: %v`, replica.ShardID, replica.ID, replica.HostID, replica.IsNonVoting)
		c = newReplicaClient(replica, host, a)
	}, true)
	return
}

// Stop stops the agent
func (a *Agent) Stop() {
	a.controller.Stop()
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

// resolvePeerGossipSeed resolves peer api address list to peer gossip address list
func (a *Agent) resolvePeerGossipSeed() (gossip []string, err error) {
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
			} else if err != nil && !strings.HasSuffix(err.Error(), `connect: connection refused"`) {
				a.log.Warningf("No probe response for %s %+v %v", peerApiAddr, res, err.Error())
			}
		}
		a.log.Infof("Found %d of %d peers %+v", len(gossip), len(a.peers), gossip)
		if len(gossip) < len(a.peers) {
			select {
			case <-a.ctx.Done():
				err = fmt.Errorf(`Cancelling findGossip (agent stopped)`)
				return
			case <-a.clock.After(waitPeriod):
			}
			continue
		}
		break
	}
	return
}

// resolvePrimeMembership resolves peer raft address list to replicaID and hostID
func (a *Agent) resolvePrimeMembership() (members map[uint64]string, init bool, err error) {
	a.wg.Add(1)
	defer a.wg.Done()
	for {
		members = map[uint64]string{}
		var uninitialized = map[string]string{}
		// Get host info from all peers to determine which are initialized.
		for _, apiAddr := range a.peers {
			var info *internal.InfoResponse
			if apiAddr == a.advertiseAddress && a.HostID() != "" {
				info = &internal.InfoResponse{
					ReplicaId: a.replicaConfig.ReplicaID,
					HostId:    a.HostID(),
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
		// All peers resolved. Start agent.
		if len(members) == len(a.peers) {
			break
		}
		// All peers uninitialized. Initialize cluster.
		if len(uninitialized) == len(a.peers) {
			for i, apiAddr := range a.peers {
				replicaID := uint64(i + 1)
				members[replicaID] = uninitialized[apiAddr]
				if apiAddr == a.advertiseAddress {
					a.replicaConfig.ReplicaID = replicaID
				}
			}
			if a.replicaConfig.ReplicaID > 0 {
				init = true
				break
			}
		}
		// Some peers initialized. Retrieve member list from initialized host.
		if len(members)+len(uninitialized) == len(a.peers) {
			var res *internal.MembersResponse
			for _, apiAddr := range a.peers {
				if apiAddr == a.advertiseAddress {
					continue
				}
				if _, ok := uninitialized[apiAddr]; !ok {
					res, err = a.grpcClientPool.get(apiAddr).Members(raftCtx(), &internal.MembersRequest{})
					if err != nil {
						return
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
		a.clock.Sleep(waitPeriod)
	}
	a.log.Debugf(`Init: %v, Members: %+v`, init, members)
	return
}

// startPrimeReplica starts the prime replica
func (a *Agent) startPrimeReplica(members map[uint64]string, join bool) (err error) {
	// a.log.Debugf("Starting Replica %+v (%v)", members, join)
	err = a.host.StartReplica(members, join, fsmFactory(a), a.replicaConfig)
	if err == dragonboat.ErrShardAlreadyExist {
		a.log.Infof("Shard already exists %+v (%v) %+v", members, join, a.replicaConfig)
		err = nil
	}
	if err != nil {
		err = fmt.Errorf(`startPrimeReplica: %w`, err)
		return
	}
	err = a.readIndex(a.replicaConfig.ShardID)
	if err != nil {
		return
	}
	return
}

// joinPrimeShard requests host be added to prime shard
func (a *Agent) joinPrimeShard() (replicaID uint64, err error) {
	a.log.Debugf("Joining prime shard")
	var res *internal.JoinResponse
	for _, peerApiAddr := range a.peers {
		res, err = a.grpcClientPool.get(peerApiAddr).Join(raftCtx(), &internal.JoinRequest{
			HostId:      a.HostID(),
			IsNonVoting: a.replicaConfig.IsNonVoting,
		})
		if res != nil && res.Value > 0 {
			replicaID = res.Value
			break
		}
	}
	return
}

// updateHost adds host info to prime shard
func (a *Agent) updateHost() (err error) {
	meta, apiAddr, err := a.parseMeta(a.HostID())
	if err != nil {
		return
	}
	shardTypes := keys(a.shardTypes)
	sort.Strings(shardTypes)
	cmd := newCmdHostPut(a.HostID(), apiAddr, a.hostConfig.RaftAddress, meta, HostStatus_Active, shardTypes)
	a.log.Debugf("Updating host: %s", string(cmd))
	_, err = a.primePropose(cmd)
	return
}

// updateReplica sets prime shard replica to active
func (a *Agent) updateReplica() (err error) {
	_, err = a.primePropose(newCmdReplicaUpdateStatus(a.replicaConfig.ReplicaID, ReplicaStatus_Active))
	if err != nil {
		err = fmt.Errorf("Failed to update replica status: %w", err)
	}
	return
}

// readIndex blocks until it can read from the prime shard, indicating that the local replica is up to date.
func (a *Agent) readIndex(shardID uint64) (err error) {
	var rs *dragonboat.RequestState
	for i := 0; i < 10; i++ {
		rs, err = a.host.ReadIndex(shardID, raftTimeout)
		if err != nil || rs == nil {
			a.log.Infof(`[%05x] Error reading shard index: %s: %v`, shardID, a.HostID(), err)
			a.clock.Sleep(waitPeriod)
			continue
		}
		res := <-rs.ResultC()
		if !res.Completed() {
			a.log.Infof(`[%05x] Waiting for other nodes`, shardID)
			rs.Release()
			a.clock.Sleep(waitPeriod)
			continue
		}
		rs.Release()
		break
	}
	return
}

func (a *Agent) joinPrimeReplica(hostID string, shardID uint64, isNonVoting bool) (replicaID uint64, err error) {
	var ok bool
	var host Host
	a.Read(func(s State) {
		host, ok = s.HostGet(hostID)
		if !ok {
			return
		}
	}, true)
	if host.ID == "" {
		host, err = a.primeAddHost(hostID)
		if err != nil {
			return
		}
	}
	a.Read(func(s State) {
		s.ReplicaIterateByHostID(host.ID, func(r Replica) bool {
			if r.ShardID == shardID {
				replicaID = r.ID
				return false
			}
			return true
		})
	}, true)
	if replicaID == 0 {
		if replicaID, err = a.primeAddReplica(hostID, isNonVoting); err != nil {
			return
		}
	}
	return a.joinShardReplica(hostID, shardID, replicaID, isNonVoting)
}

func (a *Agent) joinShardReplica(hostID string, shardID, replicaID uint64, isNonVoting bool) (res uint64, err error) {
	m, err := a.host.SyncGetShardMembership(raftCtx(), shardID)
	if err != nil {
		return
	}
	if isNonVoting {
		if _, ok := m.NonVotings[replicaID]; ok {
			return replicaID, nil
		}
		err = a.host.SyncRequestAddNonVoting(raftCtx(), shardID, replicaID, hostID, m.ConfigChangeID)
	} else {
		if _, ok := m.Nodes[replicaID]; ok {
			return replicaID, nil
		}
		err = a.host.SyncRequestAddReplica(raftCtx(), shardID, replicaID, hostID, m.ConfigChangeID)
	}
	if err != nil {
		return
	}
	return replicaID, nil
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
	_, err = a.primePropose(newCmdShardPut(a.replicaConfig.ShardID, shardUri, shardVersion))
	if err != nil {
		return
	}
	toAdd := make([]string, len(members))
	for replicaID, nhid := range members {
		_, err = a.primeAddHost(nhid)
		if err != nil {
			return
		}
		toAdd[replicaID-1] = nhid
	}
	for _, nhid := range toAdd {
		if _, err = a.primeAddReplica(nhid, false); err != nil {
			return
		}
	}

	return
}

// primeInitAwait pauses non-initializers until prime shard is initialized
func (a *Agent) primeInitAwait() (err error) {
	for {
		var found bool
		err = a.Read(func(s State) {
			s.ReplicaIterateByHostID(a.HostID(), func(r Replica) bool {
				found = true
				return false
			})
		}, true)
		if err != nil || found {
			break
		}
		a.clock.Sleep(100 * time.Millisecond)
	}
	return
}

// primeAddHost proposes addition of host metadata to the prime shard state
func (a *Agent) primeAddHost(nhid string) (host Host, err error) {
	meta, addr, err := a.parseMeta(nhid)
	if err != nil {
		return
	}
	cmd := newCmdHostPut(nhid, addr, "", meta, HostStatus_New, nil)
	_, err = a.primePropose(cmd)
	if err != nil {
		return
	}
	host = Host{
		ID:     nhid,
		Meta:   meta,
		Status: HostStatus_New,
	}
	return
}

// primeAddReplica proposes addition of replica metadata to the prime shard state
func (a *Agent) primeAddReplica(nhid string, isNonVoting bool) (id uint64, err error) {
	res, err := a.primePropose(newCmdReplicaPost(nhid, a.replicaConfig.ShardID, isNonVoting))
	if err == nil {
		id = res.Value
	}
	return
}

// findLocalReplicaID proposes addition of replica metadata to the prime shard state
func (a *Agent) findLocalReplicaID(shardID uint64) (id uint64) {
	nhInfo := a.host.GetNodeHostInfo(dragonboat.NodeHostInfoOption{false})
	for _, info := range nhInfo.LogInfo {
		if info.ShardID == shardID {
			return info.ReplicaID
		}
	}
	return
}
