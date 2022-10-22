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
	cmd_type_host    = "host"
	cmd_type_shard   = "shard"
	cmd_type_replica = "replica"

	cmd_action_set = "set"
	cmd_action_del = "del"

	query_action_get = "get"

	cmd_result_failure uint64 = 0
	cmd_result_success uint64 = 1
)

func raftNodeFactory(agent *agent) dbsm.CreateStateMachineFunc {
	return dbsm.CreateStateMachineFunc(func(shardID, replicaID uint64) dbsm.IStateMachine {
		node := &raftNode{
			shardID:   shardID,
			replicaID: replicaID,
			log:       agent.log,
			hosts:     orderedmap.NewOrderedMap[string, *host](),
			shards:    orderedmap.NewOrderedMap[uint64, *shard](),
			replicas:  orderedmap.NewOrderedMap[uint64, *replica](),
		}
		agent.setRaftNode(node)
		return node
	})
}

type raftNode struct {
	shardID   uint64
	replicaID uint64
	log       logger.ILogger
	hosts     *orderedmap.OrderedMap[string, *host]
	shards    *orderedmap.OrderedMap[uint64, *shard]
	replicas  *orderedmap.OrderedMap[uint64, *replica]
}

func (fsm *raftNode) Update(ent dbsm.Entry) (res dbsm.Result, err error) {
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
		if cmd.Host.Meta == nil {
			cmd.Host.Meta = map[string]interface{}{}
		}
		switch cmd.Action {
		case cmd_action_set:
			if !fsm.hosts.Set(cmd.Host.ID, &cmd.Host) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to set host %#v", cmd)
			}
		case cmd_action_del:
			if !fsm.hosts.Delete(cmd.Host.ID) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to delete host %#v", cmd)
			}
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
		case cmd_action_set:
			if cmd.Shard.ID == 0 {
				cmd.Shard.ID = ent.Index
			}
			if !fsm.shards.Set(cmd.Shard.ID, &cmd.Shard) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to set shard %#v", cmd)
			}
			if res.Data, err = json.Marshal(cmd.Shard); err != nil {
				fsm.log.Warningf("Error marshaling shard to json %#v", cmd.Shard)
			}
		case cmd_action_del:
			if !fsm.shards.Delete(cmd.Shard.ID) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to delete shard %#v", cmd)
			}
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
		case cmd_action_set:
			if cmd.Replica.ID == 0 {
				cmd.Replica.ID = ent.Index
			}
			if host, ok := fsm.hosts.Get(cmd.Replica.HostID); ok {
				host.Replicas[cmd.Replica.ID] = cmd.Replica.ShardID
			} else {
				fsm.log.Warningf("Host not found %#v", cmd)
			}
			if shard, ok := fsm.shards.Get(cmd.Replica.ShardID); ok {
				shard.Replicas[cmd.Replica.ID] = cmd.Replica.HostID
			} else {
				fsm.log.Warningf("Shard not found %#v", cmd)
			}
			if !fsm.replicas.Set(cmd.Replica.ID, &cmd.Replica) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to set replica %#v", cmd)
				break
			}
			res.Value = cmd.Replica.ID
		case cmd_action_del:
			replica, ok := fsm.replicas.Get(cmd.Replica.ID)
			if !ok {
				break
			}
			if host, ok := fsm.hosts.Get(replica.HostID); ok {
				delete(host.Replicas, replica.ID)
			} else {
				fsm.log.Warningf("Host not found %#v", cmd)
			}
			if shard, ok := fsm.shards.Get(replica.ShardID); ok {
				delete(shard.Replicas, replica.ID)
			} else {
				fsm.log.Warningf("Shard not found %#v", cmd)
			}
			if !fsm.shards.Delete(replica.ID) {
				res = dbsm.Result{Value: cmd_result_failure}
				fsm.log.Warningf("Failed to delete replica %#v", cmd)
			}
		default:
			fsm.log.Errorf("Unrecognized replica action %s", cmd.Action)
		}
	default:
		fsm.log.Errorf("Unrecognized type %s", cmd.Action)
	}

	return
}

func (fsm *raftNode) Lookup(e interface{}) (val interface{}, err error) {
	if query, ok := e.(queryHost); ok {
		switch query.Action {
		case query_action_get:
			if host, ok := fsm.hosts.Get(query.Host.ID); ok {
				val = *host
			} else {
				fsm.log.Warningf("Host not found %#v", e)
			}
		}
	} else if query, ok := e.(queryReplica); ok {
		switch query.Action {
		case query_action_get:
			host, ok := fsm.hosts.Get(query.Replica.HostID)
			if !ok {
				fsm.log.Warningf("Host not found %#v", e)
				break
			}
			for replicaID, shardID := range host.Replicas {
				if shardID == query.Replica.ShardID {
					val, _ = fsm.replicas.Get(replicaID)
					break
				}
			}
		}
	} else {
		err = fmt.Errorf("Invalid query %#v", e)
	}

	return
}

func (fsm *raftNode) PrepareSnapshot() (ctx interface{}, err error) {
	return
}

func (fsm *raftNode) SaveSnapshot(w io.Writer, sfc dbsm.ISnapshotFileCollection, stopc <-chan struct{}) (err error) {
	var hosts []*host
	for el := fsm.hosts.Front(); el != nil; el = el.Next() {
		hosts = append(hosts, el.Value)
	}
	var shards []*shard
	for el := fsm.shards.Front(); el != nil; el = el.Next() {
		shards = append(shards, el.Value)
	}
	var replicas []*replica
	for el := fsm.replicas.Front(); el != nil; el = el.Next() {
		replicas = append(replicas, el.Value)
	}
	b, err := json.Marshal(snapshot{
		Hosts:    hosts,
		Shards:   shards,
		Replicas: replicas,
	})
	if err == nil {
		_, err = io.Copy(w, bytes.NewReader(b))
	}

	return
}

func (fsm *raftNode) RecoverFromSnapshot(r io.Reader, sfc []dbsm.SnapshotFile, stopc <-chan struct{}) (err error) {
	var data snapshot
	if err = json.NewDecoder(r).Decode(&data); err != nil {
		return
	}
	for _, host := range data.Hosts {
		fsm.hosts.Set(host.ID, host)
	}
	for _, shard := range data.Shards {
		fsm.shards.Set(shard.ID, shard)
	}
	for _, replica := range data.Replicas {
		fsm.replicas.Set(replica.ID, replica)
	}
	return
}

func (fsm *raftNode) Close() (err error) {
	return
}

type snapshot struct {
	Hosts    []*host
	Shards   []*shard
	Replicas []*replica
}

type host struct {
	ID       string
	Replicas map[uint64]uint64 // replicaID: shardID
	Meta     map[string]interface{}
	Status   string
}

type shard struct {
	ID       uint64
	Replicas map[uint64]string // replicaID: nodehostID
	Name     string
	Status   string
}

type replica struct {
	ID      uint64
	ShardID uint64
	HostID  string
	Status  string
}

type cmd struct {
	Type   string
	Action string
}

type cmdHost struct {
	cmd
	Host host
}

type cmdShard struct {
	cmd
	Shard shard
}

type cmdReplica struct {
	cmd
	Replica replica
}

func newCmdReplicaSet(primeShardID uint64, nhid string) cmdReplica {
	return cmdReplica{cmd{
		Type:   cmd_type_replica,
		Action: cmd_action_set,
	}, replica{
		ShardID: primeShardID,
		HostID:  nhid,
		Status:  ReplicaStatus_New,
	}}
}

func newCmdReplicaDel(replicaID uint64) cmdReplica {
	return cmdReplica{cmd{
		Type:   cmd_type_replica,
		Action: cmd_action_del,
	}, replica{
		ID: replicaID,
	}}
}

type query struct {
	Type   string
	Action string
}

type queryHost struct {
	query
	Host host
}

type queryReplica struct {
	query
	Replica replica
}

func newQueryHostGet(nhid string) queryHost {
	return queryHost{query{
		Type:   cmd_type_host,
		Action: query_action_get,
	}, host{
		ID: nhid,
	}}
}

func newQueryReplicaGet(nhid string, shardID uint64) queryReplica {
	return queryReplica{query{
		Type:   cmd_type_replica,
		Action: query_action_get,
	}, replica{
		HostID:  nhid,
		ShardID: shardID,
	}}
}
