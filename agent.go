package zongzi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/logger"
)

type AgentConfig struct {
	NodeHostConfig config.NodeHostConfig
	RaftNodeConfig config.Config
	Multicast      []string
}

type Agent interface {
	Start() error
	Init() error
	GetNodeHost() *dragonboat.NodeHost
	GetStatus() AgentStatus
	GetSnapshot() (snapshot, error)
	GetSnapshotJson() ([]byte, error)
	PeerCount() int
	Stop()
}

type agent struct {
	log         logger.ILogger
	hostConfig  config.NodeHostConfig
	metaConfig  config.Config
	clusterName string
	client      UDPClient
	multicast   []string

	host     *dragonboat.NodeHost
	hostFS   fs.FS
	raftNode *raftNode
	clock    clock.Clock
	status   AgentStatus
	mutex    sync.RWMutex
	cancel   context.CancelFunc
	peers    map[string]string // gossipAddr: discoveryAddr
	probes   map[string]bool   // gossipAddr: true
}

func NewAgent(cfg AgentConfig) (*agent, error) {
	clusterName := base36Encode(cfg.NodeHostConfig.DeploymentID)
	if cfg.RaftNodeConfig.ShardID == 0 {
		cfg.RaftNodeConfig = DefaultRaftNodeConfig
	}
	return &agent{
		log:         logger.GetLogger(magicPrefix),
		hostConfig:  cfg.NodeHostConfig,
		metaConfig:  cfg.RaftNodeConfig,
		clusterName: clusterName,
		client:      newUDPClient(magicPrefix, cfg.NodeHostConfig.RaftAddress, clusterName),
		hostFS:      os.DirFS(cfg.NodeHostConfig.NodeHostDir),
		multicast:   cfg.Multicast,
		clock:       clock.New(),
		peers:       map[string]string{},
		probes:      map[string]bool{},
	}, nil
}

func (a *agent) Start() (err error) {
	go a.startUDPListener()
	if a.hostExists() {
		a.setStatus(AgentStatus_Rejoining)
		err = a.rejoin()
	} else {
		a.setStatus(AgentStatus_Pending)
		err = a.join()
	}
	return
}

func (h *agent) hostExists() bool {
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

func (a *agent) GetNodeHost() *dragonboat.NodeHost {
	return a.host
}

func (a *agent) startUDPListener() {
	defer func() {
		a.log.Debugf("Stopped Listener")
	}()
	var f = fmt.Sprintf
	var validate = func(cmd string, args []string, n int) (res, msg string, ok bool) {
		ok = len(args) == n
		if !ok {
			res = f("%s_INVALID", cmd)
			msg = f("%s requires %d arguments: %v", cmd, n, args)
		}
		return
	}
	var r = func(args ...string) []string {
		return args
	}
	var conflict = func(cmd, msg string) (logger.LogLevel, string, []string) {
		return logger.WARNING, cmd + "_CONFLICT", r(msg)
	}
	var noresp = func() (logger.LogLevel, string, []string) {
		return logger.DEBUG, "", nil
	}
	var err error
	var handler = func(cmd string, args ...string) (logger.LogLevel, string, []string) {
		msg := f("%s %s", cmd, strings.Join(args, " "))
		switch cmd {
		// Initialize the deployment - Sent manually to a single node during deployment boostrap
		case "INIT":
			if a.GetStatus() == AgentStatus_Active {
				return conflict(cmd, "Already initialized")
			}
			a.log.Debugf("init UDP")
			err = a.init(a.safeStrMapCopy(a.peers))
			if err != nil {
				return logger.ERROR, "INIT_ERROR", r(f("Failed to initialize: %v", err.Error()))
			}
			return logger.INFO, "INIT_SUCCESS", r(f("%s", a.hostID()))

		// Start nodehost - Called by the node that receives the init command
		case "INIT_START":
			if res, msg, ok := validate(cmd, args, 1); !ok {
				return logger.WARNING, res, r(msg)
			}
			seedList := args[0]
			seeds := strings.Split(seedList, ",")
			var replicaID uint64
			for i, gossipAddr := range seeds {
				if gossipAddr == a.hostConfig.Gossip.AdvertiseAddress {
					replicaID = uint64(i + 1)
				}
			}
			if replicaID == 0 {
				return noresp()
			}
			err = a.startHost(seeds)
			if err != nil {
				return logger.ERROR, "INIT_START_ERROR", r(f("Failed to start node host during init: %v %s", err.Error(), seedList))
			}
			return logger.INFO, "INIT_START_SUCCESS", r(strconv.Itoa(int(replicaID)), a.hostID())

		// Join control cluster - Called by the node that receives init command after all nodes have started
		case "INIT_JOIN":
			if res, msg, ok := validate(cmd, args, 1); !ok {
				return logger.WARNING, res, r(msg)
			}
			var replicaID uint64
			initialMembers := map[uint64]string{}
			err := json.Unmarshal([]byte(args[0]), &initialMembers)
			if err != nil {
				return logger.ERROR, "INIT_JOIN_ERROR", r(f("Failed to unmarshal initial members: %v", err.Error()))
			}
			for i, v := range initialMembers {
				if v == a.hostID() {
					replicaID = i
				}
			}
			if replicaID < 1 {
				return logger.ERROR, "INIT_JOIN_ERROR", r("Node not in initial members")
			}
			err = a.startReplica(initialMembers, false, primeShardID, replicaID)
			if err != nil {
				return logger.ERROR, "INIT_JOIN_ERROR", r(f("Failed to start meta shard during init: %v", err.Error()))
			}
			a.setStatus(AgentStatus_Active)
			return logger.INFO, "INIT_JOIN_SUCCESS", r(f("%d", replicaID))

		// Join a deployment - Sent by new nodes joining cluster
		case "JOIN":
			if res, msg, ok := validate(cmd, args, 3); !ok {
				return logger.WARNING, res, r(msg)
			}
			a.log.Debugf("Received JOIN request: %+v", args)
			nhid, gossipAddr, discoveryAddr := args[0], args[1], args[2]
			if a.GetStatus() != AgentStatus_Active {
				a.mutex.Lock()
				a.peers[gossipAddr] = discoveryAddr
				a.mutex.Unlock()
				return noresp()
			}
			if len(nhid) == 0 {
				return logger.INFO, "JOIN_START", r(a.hostConfig.Gossip.AdvertiseAddress)
			}
			replicaID, err := a.addReplica(nhid)
			if err != nil {
				return logger.ERROR, "JOIN_ERROR", r(f("Failed to add meta shard replica during join: %v", err.Error()))
			}
			return logger.INFO, "JOIN_SUCCESS", r(f("%d", replicaID))

		// Find seeds for deployment - Sent by existing nodes after restart
		case "REJOIN":
			if res, msg, ok := validate(cmd, args, 1); !ok {
				return logger.WARNING, res, r(msg)
			}
			gossipAddr := args[0]
			a.log.Debugf("REJOIN req from %s\n", gossipAddr)
			a.probes[gossipAddr] = true
			if a.GetStatus() == AgentStatus_Active || a.GetStatus() == AgentStatus_Rejoining {
				return logger.INFO, "REJOIN_SUCCESS", r(a.hostConfig.Gossip.AdvertiseAddress)
			}
			return noresp()

		// List peers - Sent manually prior to init
		case "LIST":
			nodeID, seedList, err := a.parsePeers(a.safeStrMapCopy(a.peers))
			if err == nil {
				return logger.INFO, "LIST_SUCCESS", r(f("%d %v %s %v", nodeID, err == nil, strings.Join(seedList, ","), err))
			} else {
				return noresp()
			}

		// Clear peers - Sent manually prior to init
		case "CLEAR":
			a.mutex.Lock()
			p := len(a.peers)
			a.peers = map[string]string{}
			a.mutex.Unlock()
			return logger.INFO, "CLEAR_SUCCESS", r(f("%d peers", p))

		default:
			return logger.WARNING, "ERROR", r(f("Unrecognized command: %s", msg))
		}
	}
	var ctx context.Context
	ctx, a.cancel = context.WithCancel(context.Background())
	a.log.Infof("UDP Listening on %s", fmt.Sprintf(":%s", strings.Split(a.hostConfig.RaftAddress, ":")[1]))
	for {
		err := a.client.Listen(ctx, UDPHandlerFunc(handler))
		a.log.Errorf("Error reading UDP: %s", err.Error())
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (a *agent) addReplica(nhid string) (replicaID uint64, err error) {
	host, err := a.host.SyncRead(raftCtx(), primeShardID, newQueryHostGet(nhid))
	if err != nil {
		return
	}
	if host == nil {
		err = a.primeAddHost(nhid)
		if err != nil {
			return
		}
	}
	res, err := a.host.SyncRead(raftCtx(), primeShardID, newQueryReplicaGet(nhid, primeShardID))
	if err != nil {
		return
	}
	if res == nil {
		if replicaID, err = a.primeAddReplica(nhid, 0); err != nil {
			return
		}
	} else {
		replicaID = res.(*replica).ID
	}
	m, err := a.host.SyncGetShardMembership(raftCtx(), primeShardID)
	if err != nil {
		return
	}
	err = a.host.SyncRequestAddReplica(raftCtx(), primeShardID, replicaID, nhid, m.ConfigChangeID)
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
		err = errNodeNotFound
	}
	return
}

func (a *agent) GetStatus() AgentStatus {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.status == AgentStatus_Pending {
		_, list, err := a.parsePeers(a.peers)
		if err == nil && len(list) >= minReplicas {
			a.status = AgentStatus_Ready
			a.log.Debugf("Status: %v", a.status)
		}
	}
	return a.status
}

func (a *agent) setStatus(s AgentStatus) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.log.Debugf("Status: %v", s)
	a.status = s
}

func (a *agent) GetSnapshotJson() (b []byte, err error) {
	if a.raftNode == nil {
		err = fmt.Errorf("Raft node not found")
		return
	}
	var bb bytes.Buffer
	if err = a.raftNode.SaveSnapshot(&bb, nil, nil); err != nil {
		return
	}
	b = bb.Bytes()
	return
}

func (a *agent) GetSnapshot() (res snapshot, err error) {
	b, err := a.GetSnapshotJson()
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &res)
	return
}

func (a *agent) PeerCount() int {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return len(a.peers)
}

func (a *agent) setRaftNode(r *raftNode) {
	a.raftNode = r
}

func (a *agent) Init() (err error) {
	err = a.init(a.safeStrMapCopy(a.peers))
	return
}

func (a *agent) init(peers map[string]string) (err error) {
	status := a.GetStatus()
	if status == AgentStatus_Initializing {
		return fmt.Errorf("Agent busy (%s)", status)
	}
	nodeID, seedList, err := a.parsePeers(peers)
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
	var members = map[uint64]string{}
	for _, gossipAddr := range seedList {
		if gossipAddr == a.hostConfig.Gossip.AdvertiseAddress {
			err = a.startHost(seedList)
			if err != nil {
				err = fmt.Errorf("Failed to start node host: %v %s", err.Error(), strings.Join(seedList, ", "))
				return
			}
			members[nodeID] = a.hostID()
			continue
		}
		discoveryAddr := peers[gossipAddr]
		res, args, err2 := a.client.Send(joinTimeout, discoveryAddr, "INIT_START", strings.Join(seedList, ","))
		if err2 != nil {
			err = err2
			return
		}
		if res == "INIT_START_SUCCESS" {
			if len(args) != 2 {
				err = fmt.Errorf("Invalid response from %s / %s: %v", discoveryAddr, gossipAddr, args)
				return
			}
			i, err2 := strconv.Atoi(args[0])
			if err2 != nil {
				err = fmt.Errorf("Invalid nodeID from %s / %s: %s", discoveryAddr, gossipAddr, args[0])
				return
			}
			members[uint64(i)] = args[1]
		} else {
			err = fmt.Errorf("Unrecognized INIT_START response from %s / %s: %s %v", discoveryAddr, gossipAddr, res, args)
			return
		}
	}
	memberJson, err := json.Marshal(members)
	if err != nil {
		err = fmt.Errorf("Error marshaling member json: %v", err)
		return
	}
	_, err = a.host.SyncGetShardMembership(raftCtx(), primeShardID)
	if err != nil {
		for _, gossipAddr := range seedList {
			if gossipAddr == a.hostConfig.Gossip.AdvertiseAddress {
				err = a.startReplica(members, false, primeShardID, nodeID)
				if err != nil {
					err = fmt.Errorf("Failed to start meta shard a: %v", err.Error())
					return
				}
				continue
			}
			discoveryAddr := peers[gossipAddr]
			var res string
			var args []string
			for i := 0; i < 10; i++ {
				res, args, err = a.client.Send(joinTimeout, discoveryAddr, "INIT_JOIN", string(memberJson))
				if err != nil {
					a.clock.Sleep(time.Second)
					a.log.Warningf("%v", err)
					continue
				}
				err = nil
				break
			}
			if res != "INIT_JOIN_SUCCESS" {
				err = fmt.Errorf("Unrecognized INIT_JOIN response from %s / %s: %s %v", discoveryAddr, gossipAddr, res, args)
				return
			}
		}
	}
	for {
		a.log.Debugf("init meta shard")
		if err = a.createPrimeShard(members); err == nil {
			break
		}
		a.log.Debugf("%v", err)
		time.Sleep(time.Second)
	}

	return
}

// createPrimeShard proposes addition of prime shard metadata to prime shard state
func (a *agent) createPrimeShard(members map[uint64]string) (err error) {
	sess := a.host.GetNoOPSession(primeShardID)
	var b []byte
	if b, err = json.Marshal(cmdShard{cmd{
		Type:   cmd_type_shard,
		Action: cmd_action_set,
	}, shard{
		ID:     primeShardID,
		Name:   magicPrefix,
		Status: "new",
	}}); err != nil {
		return
	}
	_, err = a.host.SyncPropose(raftCtx(), sess, b)
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
	sess := a.host.GetNoOPSession(primeShardID)
	reg, ok := a.host.GetNodeHostRegistry()
	if !ok {
		err = fmt.Errorf("Unable to retrieve NodeHostRegistry")
		return
	}
	metaJson, ok := reg.GetMeta(nhid)
	if !ok {
		err = fmt.Errorf("Unable to retrieve node host meta")
		return
	}
	var meta = map[string]interface{}{}
	err = json.Unmarshal(metaJson, &meta)
	if err != nil {
		return
	}
	b, err := json.Marshal(cmdHost{cmd{
		Type:   cmd_type_host,
		Action: cmd_action_set,
	}, host{
		ID:     nhid,
		Meta:   meta,
		Status: "new",
	}})
	if err != nil {
		return
	}
	_, err = a.host.SyncPropose(raftCtx(), sess, b)
	return
}

// primeAddReplica proposes addition of replica metadata to the prime shard state
func (a *agent) primeAddReplica(nhid string, replicaID uint64) (id uint64, err error) {
	sess := a.host.GetNoOPSession(primeShardID)
	b, err := json.Marshal(cmdReplica{cmd{
		Type:   cmd_type_replica,
		Action: cmd_action_set,
	}, replica{
		ID:      replicaID,
		ShardID: primeShardID,
		HostID:  nhid,
		Status:  "new",
	}})
	if err != nil {
		return
	}
	res, err := a.host.SyncPropose(raftCtx(), sess, b)
	if err == nil {
		id = res.Value
	}
	return
}

func (a *agent) startReplica(members map[uint64]string, join bool, shardID uint64, replicaID uint64) (err error) {
	a.log.Debugf("startReplica: %s", string(debug.Stack()))
	cfg := config.Config{
		CheckQuorum:         true,
		ShardID:             shardID,
		ReplicaID:           replicaID,
		CompactionOverhead:  5,
		ElectionRTT:         10,
		HeartbeatRTT:        1,
		OrderedConfigChange: true,
		Quiesce:             false,
		SnapshotEntries:     10,
	}
	if replicaID == 0 && a.host != nil {
		ctx, _ := context.WithTimeout(context.Background(), raftTimeout)
		m, err := a.host.SyncGetShardMembership(ctx, shardID)
		if err != nil {
			return fmt.Errorf("Failed to get membership during control start: %v", err.Error())
		}
		for i, nodeHostID := range m.Nodes {
			if nodeHostID == a.host.ID() {
				cfg.ReplicaID = i
			}
		}
	}
	if cfg.ReplicaID < 1 {
		return fmt.Errorf("Invalid replicaID: %v", err.Error())
	}
	err = a.host.StartReplica(members, join, raftNodeFactory(a), cfg)
	if err != nil {
		return fmt.Errorf("Failed to start meta shard b: %w", err)
	}
	return
}

func (a *agent) findGossipSeeds(min int) (res []string) {
	var seeds = map[string]bool{}
	a.log.Infof("Finding gossip seeds: %s", strings.Join(a.multicast, ", "))
	for {
		for _, addr := range a.multicast {
			cmd, args, err := a.client.Send(time.Second, addr, "REJOIN", a.hostConfig.Gossip.AdvertiseAddress)
			if err != nil {
				continue
			}
			if cmd == "REJOIN_SUCCESS" && len(args) == 1 {
				gossipAddr := args[0]
				seeds[gossipAddr] = true
			} else if len(cmd) > 0 {
				a.log.Errorf("[%s] Invalid rejoin response: %s, %v", a.hostID(), cmd, args)
			}
		}
		if len(seeds) >= min {
			for k := range seeds {
				res = append(res, k)
			}
			break
		}
		a.clock.Sleep(time.Second)
	}

	return
}

func (a *agent) hostID() string {
	if a.host != nil {
		return a.host.ID()
	}
	return ""
}

func (a *agent) join() error {
	if len(a.multicast) == 0 {
		return fmt.Errorf("No broadcast address configured in discovery agent")
	}
	defer func() {
		a.log.Debugf("Stopped Joining")
	}()
	var broadcast = func(addr string) {
		res, args, err := a.client.Send(time.Second, addr, "JOIN", a.hostID(), a.hostConfig.Gossip.AdvertiseAddress, a.hostConfig.RaftAddress)
		if err == nil && len(res) == 0 {
			return
		}
		if a.GetStatus() != AgentStatus_Pending && a.GetStatus() != AgentStatus_Ready {
			return
		}
		if err == nil && res == "JOIN_START" && len(args) == 1 {
			seed := args[0]
			err = a.startHost(strings.Split(seed, ","))
			if err != nil {
				a.log.Errorf("[%s] Unable to restart node host: %s", a.hostID(), err.Error())
				return
			}
			res, args, err = a.client.Send(time.Second, addr, "JOIN", a.hostID(), a.hostConfig.Gossip.AdvertiseAddress, a.hostConfig.RaftAddress)
			if err == nil && len(res) == 0 {
				return
			}
		}
		if err == nil && res == "JOIN_SUCCESS" && len(args) == 1 {
			replicaID, err := strconv.Atoi(args[0])
			if err != nil {
				a.log.Errorf("[%s] Invalid node id: %s", a.hostID(), args[0])
				return
			}
			a.setStatus(AgentStatus_Joining)
			err = a.startReplica(nil, true, primeShardID, uint64(replicaID))
			if err != nil {
				a.log.Errorf("[%s] Unable to start meta shard in join: %s", a.hostID(), err.Error())
				return
			}
			a.setStatus(AgentStatus_Active)
			a.log.Infof("[%s] Joined deployment %d", a.hostID(), a.clusterName)
		} else {
			a.log.Errorf("[%s] Invalid join response: %s %v", a.hostID(), res, err)
			return
		}
	}
	a.log.Debugf("Broadcasting: %#v", a.multicast)
	for {
		for _, addr := range a.multicast {
			// a.log.Debugf("Broadcast: %v, %s", addr, a.GetStatus())
			if a.GetStatus() != AgentStatus_Pending && a.GetStatus() != AgentStatus_Ready {
				return nil
			}
			broadcast(addr)
		}
	}
	return nil
}

func (a *agent) rejoin() (err error) {
	seeds := a.findGossipSeeds(minReplicas)
	if err = a.startHost(seeds); err != nil {
		return
	}
	replicaID := a.getReplicaID()
	if err = a.startReplica(nil, false, primeShardID, uint64(replicaID)); err == nil {
		return
	}
	a.setStatus(AgentStatus_Active)

	return nil
}

func (a *agent) getReplicaID() (replicaID uint64) {
	info := a.host.GetNodeHostInfo(dragonboat.NodeHostInfoOption{})
	a.log.Debugf("%v", info)
	for _, c := range info.LogInfo {
		if c.ShardID == primeShardID {
			replicaID = c.ReplicaID
			break
		}
	}
	return
}

func (a *agent) startHost(seeds []string) (err error) {
	if a.host != nil {
		return nil
	}
	a.hostConfig.AddressByNodeHostID = true
	a.hostConfig.Gossip.Seed = seeds
	a.host, err = dragonboat.NewNodeHost(a.hostConfig)
	if err != nil {
		return
	}

	return
}

func (a *agent) Stop() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.cancel()
}

func (a *agent) safeStrMapCopy(m map[string]string) map[string]string {
	c := map[string]string{}
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	for k, v := range m {
		c[k] = v
	}
	return c
}
