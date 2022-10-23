package zongzi

import (
	"bytes"
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
	primeConfig config.Config
	clusterName string
	client      UDPClient
	multicast   []string
	listener    *udpListener

	host     *dragonboat.NodeHost
	hostFS   fs.FS
	raftNode *raftNode
	clock    clock.Clock
	status   AgentStatus
	mutex    sync.RWMutex
}

func NewAgent(cfg AgentConfig) (*agent, error) {
	clusterName := base36Encode(cfg.NodeHostConfig.DeploymentID)
	if cfg.RaftNodeConfig.ElectionRTT == 0 {
		cfg.RaftNodeConfig = DefaultRaftNodeConfig
	}
	if cfg.RaftNodeConfig.ShardID == 0 {
		cfg.RaftNodeConfig.ShardID = 1
	}
	a := &agent{
		log:         logger.GetLogger(magicPrefix),
		hostConfig:  cfg.NodeHostConfig,
		primeConfig: cfg.RaftNodeConfig,
		clusterName: clusterName,
		client:      newUDPClient(magicPrefix, cfg.NodeHostConfig.RaftAddress, clusterName),
		hostFS:      os.DirFS(cfg.NodeHostConfig.NodeHostDir),
		multicast:   cfg.Multicast,
		clock:       clock.New(),
	}
	a.listener = newUDPListener(a)
	return a, nil
}

func (a *agent) Start() (err error) {
	a.listener.Start()
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
		replicaID = res.(*replica).ID
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

// GetStatus returns the cluter status
func (a *agent) GetStatus() AgentStatus {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.status == AgentStatus_Pending {
		_, list, err := a.parsePeers(a.listener.Peers())
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

// GetSnapshotJson returns a snapshot of the prime shard state machine in JSON format
// TODO - Change snapshot format from JSON to JSON Lines for streaming https://jsonlines.org
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

// GetSnapshot returns a go object representing a snapshot of the prime shard
func (a *agent) GetSnapshot() (res snapshot, err error) {
	b, err := a.GetSnapshotJson()
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &res)
	return
}

// PeerCount returns the number of detected peer agents
func (a *agent) PeerCount() int {
	return len(a.listener.Peers())
}

func (a *agent) setRaftNode(r *raftNode) {
	a.raftNode = r
}

// Init initializes the cluster - This happens only once, at the beginning of the cluster lifecycle
func (a *agent) Init() (err error) {
	err = a.init(a.listener.Peers())
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
			members[replicaID] = a.hostID()
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
				err = a.startReplica(members, false, a.primeConfig)
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
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	_, err = a.host.SyncPropose(raftCtx(), sess, newCmdHostPut(nhid, meta))
	return
}

// primeAddReplica proposes addition of replica metadata to the prime shard state
func (a *agent) primeAddReplica(nhid string, replicaID uint64) (id uint64, err error) {
	sess := a.host.GetNoOPSession(a.primeConfig.ShardID)
	res, err := a.host.SyncPropose(raftCtx(), sess, newCmdReplicaPut(nhid, a.primeConfig.ShardID, replicaID))
	if err == nil {
		id = res.Value
	}
	return
}

func (a *agent) startReplica(members map[uint64]string, join bool, cfg config.Config) (err error) {
	if cfg.ReplicaID == 0 && a.host != nil {
		cfg.ReplicaID = a.getReplicaID()
	}
	if cfg.ReplicaID == 0 {
		return fmt.Errorf("Invalid replicaID: %v", err.Error())
	}
	err = a.host.StartReplica(members, join, raftNodeFactory(a), cfg)
	if err != nil {
		return fmt.Errorf("Failed to start prime shard: %w", err)
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
		res, args, err := a.client.Send(time.Second, addr, PROBE_JOIN, a.hostID(), a.hostConfig.Gossip.AdvertiseAddress, a.hostConfig.RaftAddress)
		if err == nil && len(res) == 0 {
			return
		}
		if a.GetStatus() != AgentStatus_Pending && a.GetStatus() != AgentStatus_Ready {
			return
		}
		if err == nil && res == JOIN_HOST && len(args) == 1 {
			seed := args[0]
			err = a.startHost(strings.Split(seed, ","))
			if err != nil {
				a.log.Errorf("[%s] Unable to restart node host: %s", a.hostID(), err.Error())
				return
			}
		}
		if err == nil && res == JOIN_SHARD && len(args) == 1 {
			replicaID, err := strconv.Atoi(args[0])
			if err != nil {
				a.log.Errorf("[%s] Invalid node id: %s", a.hostID(), args[0])
				return
			}
			a.setStatus(AgentStatus_Joining)
			a.primeConfig.ReplicaID = uint64(replicaID)
			err = a.startReplica(nil, true, a.primeConfig)
			if err != nil {
				a.log.Errorf("[%s] Unable to start meta shard in join: %s", a.hostID(), err.Error())
				return
			}
			a.setStatus(AgentStatus_Active)
			a.log.Infof("[%s] Joined deployment %d", a.hostID(), a.clusterName)
		} else {
			a.log.Errorf("[%s] Invalid join response: %s %v", a.hostID(), res, err)
			a.clock.Sleep(time.Second)
			return
		}
	}
	a.log.Debugf("Broadcasting: %#v", a.multicast)
	for {
		for _, addr := range a.multicast {
			if a.GetStatus() == AgentStatus_Active {
				return nil
			}
			broadcast(addr)
		}
	}
	return nil
}

func (a *agent) rejoin() (err error) {
	var seedList []string
	a.log.Infof("Finding gossip seeds: %s", strings.Join(a.multicast, ", "))
	for {
		var seedMap = map[string]bool{}
		for _, addr := range a.multicast {
			cmd, args, err := a.client.Send(time.Second, addr, PROBE_REJOIN, a.hostConfig.Gossip.AdvertiseAddress)
			if err != nil {
				continue
			}
			if cmd == REJOIN_PEER && len(args) == 1 {
				gossipAddr := args[0]
				seedMap[gossipAddr] = true
			} else if len(cmd) > 0 {
				a.log.Errorf("[%s] Invalid rejoin response: %s, %v", a.hostID(), cmd, args)
			}
		}
		if len(seedMap) >= minReplicas {
			for k := range seedMap {
				seedList = append(seedList, k)
			}
			break
		}
		a.clock.Sleep(time.Second)
	}
	a.log.Infof("Starting host: %v", seedList)
	if err = a.startHost(seedList); err != nil {
		return
	}
	a.log.Infof("Starting replica")
	if err = a.startReplica(nil, false, a.primeConfig); err != nil {
		return
	}
	a.setStatus(AgentStatus_Active)

	return nil
}

func (a *agent) getReplicaID() (replicaID uint64) {
	info := a.host.GetNodeHostInfo(dragonboat.NodeHostInfoOption{})
	a.log.Debugf("%v", info)
	for _, c := range info.LogInfo {
		if c.ShardID == a.primeConfig.ShardID {
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
	a.listener.Stop()
}
