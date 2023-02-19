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

func fsmFactory(agent *agent) dbsm.CreateStateMachineFunc {
	return dbsm.CreateStateMachineFunc(func(shardID, replicaID uint64) dbsm.IStateMachine {
		node := &fsm{
			log:       agent.log,
			replicaID: replicaID,
			shardID:   shardID,
			state: fsmState{
				hosts:    orderedmap.NewOrderedMap[string, *Host](),
				shards:   orderedmap.NewOrderedMap[uint64, *Shard](),
				replicas: orderedmap.NewOrderedMap[uint64, *Replica](),
			},
		}
		agent.fsm = node
		return node
	})
}

type fsm struct {
	index     uint64
	log       logger.ILogger
	replicaID uint64
	shardID   uint64
	state     fsmState
}

func (fsm *fsm) Update(ent dbsm.Entry) (res dbsm.Result, err error) {
	var cmd cmd
	if err = json.Unmarshal(ent.Cmd, &cmd); err != nil {
		err = fmt.Errorf("Invalid entry %#v, %w", ent, err)
		return
	}
	switch cmd.Type {
	// Host
	case cmd_type_host:
		var cmd cmdHost
		if err = json.Unmarshal(ent.Cmd, &cmd); err != nil {
			fsm.log.Errorf("Invalid host cmd %#v, %w", ent, err)
			break
		}
		if cmd.Host.Replicas == nil {
			cmd.Host.Replicas = map[uint64]uint64{}
		}
		switch cmd.Action {
		// Put
		case cmd_action_put:
			if host, ok := fsm.state.hosts.Get(cmd.Host.ID); ok {
				cmd.Host.Replicas = host.Replicas
			}
			fsm.state.hosts.Set(cmd.Host.ID, &cmd.Host)
			res.Value = cmd_result_success
			res.Data, _ = json.Marshal(cmd.Host)
		// Delete
		case cmd_action_del:
			fsm.state.hosts.Delete(cmd.Host.ID)
			res.Value = cmd_result_success
		default:
			fsm.log.Errorf("Unrecognized host action %s", cmd.Action)
		}
	// Shard
	case cmd_type_shard:
		var cmd cmdShard
		if err = json.Unmarshal(ent.Cmd, &cmd); err != nil {
			fsm.log.Errorf("Invalid shard cmd %#v, %w", ent, err)
			break
		}
		if cmd.Shard.Replicas == nil {
			cmd.Shard.Replicas = map[uint64]string{}
		}
		switch cmd.Action {
		// Put
		case cmd_action_put:
			if cmd.Shard.ID == 0 {
				cmd.Shard.ID = ent.Index
			} else if shard, ok := fsm.state.shards.Get(cmd.Shard.ID); ok {
				cmd.Shard.Replicas = shard.Replicas
			}
			fsm.state.shards.Set(cmd.Shard.ID, &cmd.Shard)
			res.Value = cmd.Shard.ID
			res.Data, _ = json.Marshal(cmd.Shard)
		// Delete
		case cmd_action_del:
			fsm.state.shards.Delete(cmd.Shard.ID)
			res.Value = cmd_result_success
		default:
			fsm.log.Errorf("Unrecognized shard action %s", cmd.Action)
		}
	// Replica
	case cmd_type_replica:
		var cmd cmdReplica
		if err = json.Unmarshal(ent.Cmd, &cmd); err != nil {
			fsm.log.Errorf("Invalid replica cmd %#v, %w", ent, err)
			break
		}
		switch cmd.Action {
		// Put
		case cmd_action_put:
			if cmd.Replica.ID == 0 {
				cmd.Replica.ID = ent.Index
			}
			if host, ok := fsm.state.hosts.Get(cmd.Replica.HostID); ok {
				host.Replicas[cmd.Replica.ID] = cmd.Replica.ShardID
			} else {
				fsm.log.Warningf("Host not found %#v", cmd)
				break
			}
			if shard, ok := fsm.state.shards.Get(cmd.Replica.ShardID); ok {
				shard.Replicas[cmd.Replica.ID] = cmd.Replica.HostID
			} else {
				fsm.log.Warningf("Shard not found %#v", cmd)
				break
			}
			fsm.state.replicas.Set(cmd.Replica.ID, &cmd.Replica)
			res.Value = cmd.Replica.ID
			res.Data, _ = json.Marshal(cmd.Replica)
		// Delete
		case cmd_action_del:
			if replica, ok := fsm.state.replicas.Get(cmd.Replica.ID); ok {
				if host, ok := fsm.state.hosts.Get(replica.HostID); ok {
					delete(host.Replicas, replica.ID)
				}
				if shard, ok := fsm.state.shards.Get(replica.ShardID); ok {
					delete(shard.Replicas, replica.ID)
				}
				fsm.state.replicas.Delete(replica.ID)
			}
			res.Value = cmd_result_success
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
			val = &Snapshot{
				Hosts:    fsm.state.listHosts(),
				Index:    fsm.index,
				Replicas: fsm.state.listReplicas(),
				Shards:   fsm.state.listShards(),
			}
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
		default:
			fsm.log.Errorf("Unrecognized replica query %s", query.Action)
		}
	} else {
		err = fmt.Errorf("Invalid query %#v", e)
	}

	return
}

func (fsm *fsm) SaveSnapshot(w io.Writer, sfc dbsm.ISnapshotFileCollection, stopc <-chan struct{}) (err error) {
	b, err := json.Marshal(Snapshot{
		Hosts:    fsm.state.listHosts(),
		Index:    fsm.index,
		Replicas: fsm.state.listReplicas(),
		Shards:   fsm.state.listShards(),
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
