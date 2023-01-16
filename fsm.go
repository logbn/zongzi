package zongzi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/lni/dragonboat/v4/logger"
	dbsm "github.com/lni/dragonboat/v4/statemachine"
)

const (
	cmd_type_host     = "host"
	cmd_type_shard    = "shard"
	cmd_type_replica  = "replica"
	cmd_type_snapshot = "snapshot"

	cmd_action_put        = "put"
	cmd_action_del        = "del"
	cmd_action_set_status = "set-status"

	query_action_get = "get"

	cmd_result_failure uint64 = 0
	cmd_result_success uint64 = 1
)

func fsmFactory(agent *agent) dbsm.CreateStateMachineFunc {
	return dbsm.CreateStateMachineFunc(func(shardID, replicaID uint64) dbsm.IStateMachine {
		node := &fsm{
			shardID:   shardID,
			replicaID: replicaID,
			log:       agent.log,
			state: &fsmState{
				hosts:    orderedmap.NewOrderedMap[string, *Host](),
				shards:   orderedmap.NewOrderedMap[uint64, *Shard](),
				replicas: orderedmap.NewOrderedMap[uint64, *Replica](),
			},
		}
		agent.setRaftNode(node)
		return node
	})
}

type fsm struct {
	index     uint64
	log       logger.ILogger
	replicaID uint64
	shardID   uint64
	state     *fsmState
}

type fsmState struct {
	hosts    *orderedmap.OrderedMap[string, *Host]
	replicas *orderedmap.OrderedMap[uint64, *Replica]
	shards   *orderedmap.OrderedMap[uint64, *Shard]
}

func (fsm *fsm) Update(ent dbsm.Entry) (res dbsm.Result, err error) {
	var cmd cmd
	if err = json.Unmarshal(ent.Cmd, &cmd); err != nil {
		err = fmt.Errorf("Invalid entry %#v, %w", ent, err)
		return
	}
	res = dbsm.Result{Value: cmd_result_success}
	switch cmd.Type {
	case cmd_type_host:
		var cmd cmdHost
		if err = json.Unmarshal(ent.Cmd, &cmd); err != nil {
			res = dbsm.Result{Value: cmd_result_failure}
			fsm.log.Errorf("Invalid host cmd %#v, %w", ent, err)
			break
		}
		if cmd.Host.Replicas == nil {
			cmd.Host.Replicas = map[uint64]uint64{}
		}
		switch cmd.Action {
		// Put Host
		case cmd_action_put:
			if !fsm.state.hosts.Set(cmd.Host.ID, &cmd.Host) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to put host %#v", cmd)
			}
		// Delete Host
		case cmd_action_del:
			if !fsm.state.hosts.Delete(cmd.Host.ID) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to delete host %#v", cmd)
			}
		// Error
		default:
			fsm.log.Errorf("Unrecognized host action %s", cmd.Action)
		}
	case cmd_type_shard:
		var cmd cmdShard
		if err := json.Unmarshal(ent.Cmd, &cmd); err != nil {
			res = dbsm.Result{Value: cmd_result_failure}
			fsm.log.Errorf("Invalid shard cmd %#v, %w", ent, err)
			break
		}
		if cmd.Shard.Replicas == nil {
			cmd.Shard.Replicas = map[uint64]string{}
		}
		switch cmd.Action {
		// Put Shard
		case cmd_action_put:
			if cmd.Shard.ID == 0 {
				cmd.Shard.ID = ent.Index
			}
			if !fsm.state.shards.Set(cmd.Shard.ID, &cmd.Shard) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to put shard %#v", cmd)
			}
			if res.Data, err = json.Marshal(cmd.Shard); err != nil {
				fsm.log.Warningf("Error marshaling shard to json %#v", cmd.Shard)
			}
			res.Value = cmd.Shard.ID
		// Delete Shard
		case cmd_action_del:
			if !fsm.state.shards.Delete(cmd.Shard.ID) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to delete shard %#v", cmd)
			}
		// Error
		default:
			fsm.log.Errorf("Unrecognized shard action %s", cmd.Action)
		}
	case cmd_type_replica:
		var cmd cmdReplica
		if err := json.Unmarshal(ent.Cmd, &cmd); err != nil {
			res = dbsm.Result{Value: cmd_result_failure}
			fsm.log.Errorf("Invalid replica cmd %#v, %w", ent, err)
			break
		}
		switch cmd.Action {
		// Put Replica
		case cmd_action_put:
			if cmd.Replica.ID == 0 {
				cmd.Replica.ID = ent.Index
			}
			if host, ok := fsm.state.hosts.Get(cmd.Replica.HostID); ok {
				host.Replicas[cmd.Replica.ID] = cmd.Replica.ShardID
			} else {
				fsm.log.Warningf("Host not found %#v", cmd)
			}
			if shard, ok := fsm.state.shards.Get(cmd.Replica.ShardID); ok {
				shard.Replicas[cmd.Replica.ID] = cmd.Replica.HostID
			} else {
				fsm.log.Warningf("Shard not found %#v", cmd)
			}
			if !fsm.state.replicas.Set(cmd.Replica.ID, &cmd.Replica) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to put replica %#v", cmd)
				break
			}
			res.Value = cmd.Replica.ID
		// Delete Replica
		case cmd_action_del:
			replica, ok := fsm.state.replicas.Get(cmd.Replica.ID)
			if !ok {
				break
			}
			if host, ok := fsm.state.hosts.Get(replica.HostID); ok {
				delete(host.Replicas, replica.ID)
			} else {
				fsm.log.Warningf("Host not found %#v", cmd)
			}
			if shard, ok := fsm.state.shards.Get(replica.ShardID); ok {
				delete(shard.Replicas, replica.ID)
			} else {
				fsm.log.Warningf("Shard not found %#v", cmd)
			}
			if !fsm.state.replicas.Delete(replica.ID) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to delete replica %#v", cmd)
			}
		// Error
		default:
			fsm.log.Errorf("Unrecognized replica action %s", cmd.Action)
		}
	default:
		fsm.log.Errorf("Unrecognized type %s", cmd.Action)
	}
	fsm.index = ent.Index

	return
}

func (fsm *fsm) Lookup(e any) (val any, err error) {
	if query, ok := e.(querySnapshot); ok {
		switch query.Action {
		// Get Snapshot
		case query_action_get:
			val = Snapshot{
				Index:    fsm.index,
				Hosts:    fsm.allHosts(),
				Shards:   fsm.allShards(),
				Replicas: fsm.allReplicas(),
			}
		// Error
		default:
			fsm.log.Errorf("Unrecognized snapshot query %s", query.Action)
		}
	} else if query, ok := e.(queryHost); ok {
		switch query.Action {
		// Get Host
		case query_action_get:
			if host, ok := fsm.state.hosts.Get(query.Host.ID); ok {
				val = *host
			} else {
				fsm.log.Warningf("Host not found %#v", e)
			}
		// Error
		default:
			fsm.log.Errorf("Unrecognized host query %s", query.Action)
		}
	} else if query, ok := e.(queryReplica); ok {
		switch query.Action {
		// Get Replica
		case query_action_get:
			host, ok := fsm.state.hosts.Get(query.Replica.HostID)
			if !ok {
				fsm.log.Warningf("Host not found %#v", e)
				break
			}
			for shardID, replicaID := range host.Replicas {
				if shardID == query.Replica.ShardID {
					val, _ = fsm.state.replicas.Get(replicaID)
					break
				}
			}
		// Error
		default:
			fsm.log.Errorf("Unrecognized replica query %s", query.Action)
		}
	} else {
		err = fmt.Errorf("Invalid query %#v", e)
	}

	return
}

func (fsm *fsm) allHosts() (hosts []*Host) {
	for el := fsm.state.hosts.Front(); el != nil; el = el.Next() {
		hosts = append(hosts, el.Value)
	}
	return hosts
}

func (fsm *fsm) allShards() (shards []*Shard) {
	for el := fsm.state.shards.Front(); el != nil; el = el.Next() {
		shards = append(shards, el.Value)
	}
	return shards
}

func (fsm *fsm) allReplicas() (replicas []*Replica) {
	for el := fsm.state.replicas.Front(); el != nil; el = el.Next() {
		replicas = append(replicas, el.Value)
	}
	return replicas
}

func (fsm *fsm) SaveSnapshot(w io.Writer, sfc dbsm.ISnapshotFileCollection, stopc <-chan struct{}) (err error) {
	b, err := json.Marshal(Snapshot{
		Index:    fsm.index,
		Hosts:    fsm.allHosts(),
		Shards:   fsm.allShards(),
		Replicas: fsm.allReplicas(),
	})
	if err == nil {
		_, err = io.Copy(w, bytes.NewReader(b))
	}

	return
}

func (fsm *fsm) RecoverFromSnapshot(r io.Reader, sfc []dbsm.SnapshotFile, stopc <-chan struct{}) (err error) {
	var data Snapshot
	if err = json.NewDecoder(r).Decode(&data); err != nil {
		return
	}
	for _, host := range data.Hosts {
		fsm.state.hosts.Set(host.ID, host)
	}
	for _, shard := range data.Shards {
		fsm.state.shards.Set(shard.ID, shard)
	}
	for _, replica := range data.Replicas {
		fsm.state.replicas.Set(replica.ID, replica)
	}
	fsm.index = data.Index
	return
}

func (fsm *fsm) Close() (err error) {
	return
}

type Snapshot struct {
	Index    uint64
	Hosts    []*Host
	Replicas []*Replica
	Shards   []*Shard
}

type Host struct {
	ID         string            `json:"id"`
	Meta       []byte            `json:"meta"`
	Replicas   map[uint64]uint64 `json:"replicas"` // replicaID: shardID
	ShardTypes []string          `json:"shardTypes"`
	Status     HostStatus        `json:"status"`
}

type Shard struct {
	Replicas map[uint64]string `json:"replicas"` // replicaID: nodehostID
	ID       uint64            `json:"id"`
	Status   ShardStatus       `json:"status"`
	Type     string            `json:"type"`
	Version  string            `json:"version"`
}

type Replica struct {
	HostID      string        `json:"hostID"`
	ID          uint64        `json:"id"`
	IsNonVoting bool          `json:"isNonVoting"`
	IsWitness   bool          `json:"isWitness"`
	ShardID     uint64        `json:"shardID"`
	Status      ReplicaStatus `json:"status"`
}

type cmd struct {
	Action string
	Type   string `json:"type"`
}

type cmdHost struct {
	cmd
	Host Host `json:"host"`
}

type cmdShard struct {
	cmd
	Shard Shard `json:"shard"`
}

type cmdReplica struct {
	cmd
	Replica Replica `json:"replica"`
}

type cmd_SetReplicaStatus struct {
	cmd
	ID     uint64
	Status ReplicaStatus
}

func newCmdSetReplicaStatus(id uint64, status ReplicaStatus) (b []byte) {
	b, _ = json.Marshal(cmdReplica{cmd{
		Type:   cmd_type_replica,
		Action: cmd_action_set_status,
	}, Replica{
		ID:     id,
		Status: status,
	}})
	return
}

func newCmdReplicaPut(nhid string, shardID, replicaID uint64, isNonVoting bool) (b []byte) {
	b, _ = json.Marshal(cmdReplica{cmd{
		Type:   cmd_type_replica,
		Action: cmd_action_put,
	}, Replica{
		ID:          replicaID,
		ShardID:     shardID,
		HostID:      nhid,
		Status:      ReplicaStatus_New,
		IsNonVoting: isNonVoting,
	}})
	return
}

func newCmdHostPut(nhid string, meta []byte, status HostStatus, shardTypes []string) (b []byte) {
	b, _ = json.Marshal(cmdHost{cmd{
		Type:   cmd_type_host,
		Action: cmd_action_put,
	}, Host{
		ID:         nhid,
		Meta:       meta,
		Status:     status,
		ShardTypes: shardTypes,
	}})
	return
}

func newCmdShardPut(shardID uint64, shardType string) (b []byte) {
	b, _ = json.Marshal(cmdShard{cmd{
		Type:   cmd_type_shard,
		Action: cmd_action_put,
	}, Shard{
		ID:     shardID,
		Type:   shardType,
		Status: ShardStatus_New,
	}})
	return
}

func newCmdReplicaDel(replicaID uint64) cmdReplica {
	return cmdReplica{cmd{
		Type:   cmd_type_replica,
		Action: cmd_action_del,
	}, Replica{
		ID: replicaID,
	}}
}

type query struct {
	Type   string `json:"type"`
	Action string `json:"actioon"`
}

type queryHost struct {
	query
	Host Host `json:"host"`
}

type queryReplica struct {
	query
	Replica Replica `json:"replica"`
}

type querySnapshot struct {
	query
}

func newQueryHostGet(nhid string) queryHost {
	return queryHost{query{
		Type:   cmd_type_host,
		Action: query_action_get,
	}, Host{
		ID: nhid,
	}}
}

func newQueryReplicaGet(nhid string, shardID uint64) queryReplica {
	return queryReplica{query{
		Type:   cmd_type_replica,
		Action: query_action_get,
	}, Replica{
		HostID:  nhid,
		ShardID: shardID,
	}}
}

func newQuerySnapshotGet() querySnapshot {
	return querySnapshot{query{
		Type:   cmd_type_snapshot,
		Action: query_action_get,
	}}
}
