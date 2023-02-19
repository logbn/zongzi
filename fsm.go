package zongzi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/logger"
	dbsm "github.com/lni/dragonboat/v4/statemachine"
)

func fsmFactory(agent *agent) dbsm.CreateStateMachineFunc {
	return dbsm.CreateStateMachineFunc(func(shardID, replicaID uint64) dbsm.IStateMachine {
		node := &fsm{
			log:       agent.log,
			replicaID: replicaID,
			shardID:   shardID,
			store:     newFsmStoreMap(),
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
	store     fsmStore
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
			fsm.store.HostPut(&cmd.Host)
			res.Value = cmd_result_success
			res.Data, _ = json.Marshal(cmd.Host)
		// Delete
		case cmd_action_del:
			res.Value = cmd_result_success
			res.Data, _ = json.Marshal(map[string]any{
				"replicasDeleted": fsm.store.HostDelete(cmd.Host.ID),
			})
		default:
			err = fmt.Errorf("Unrecognized host action %s", cmd.Action)
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
			}
			fsm.store.ShardPut(&cmd.Shard)
			res.Value = cmd.Shard.ID
			res.Data, _ = json.Marshal(cmd.Shard)
		// Delete
		case cmd_action_del:
			res.Value = cmd_result_success
			res.Data, _ = json.Marshal(map[string]any{
				"replicasDeleted": fsm.store.ShardDelete(cmd.Shard.ID),
			})
		default:
			err = fmt.Errorf("Unrecognized shard action %s", cmd.Action)
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
			if err := fsm.store.ReplicaPut(&cmd.Replica); err != nil {
				fsm.log.Warningf("%w: %#v", err, cmd)
				break
			}
			res.Value = cmd.Replica.ID
			res.Data, _ = json.Marshal(cmd.Replica)
		// Delete
		case cmd_action_del:
			fsm.store.ReplicaDelete(cmd.Replica.ID)
			res.Value = cmd_result_success
		default:
			err = fmt.Errorf("Unrecognized replica action %s", cmd.Action)
		}
	default:
		err = fmt.Errorf("Unrecognized type %s", cmd.Action)
	}
	fsm.index = ent.Index

	return
}

func (fsm *fsm) Lookup(e any) (val any, err error) {
	if query, ok := e.(queryHost); ok {
		switch query.Action {
		// Get Host
		case query_action_get:
			if host := fsm.store.HostFind(query.Host.ID); host != nil {
				val = *host
			} else {
				fsm.log.Warningf("Host not found %#v", e)
			}
		default:
			err = fmt.Errorf("Unrecognized host query %s", query.Action)
		}
	} else if query, ok := e.(querySnapshot); ok {
		switch query.Action {
		// Get Snapshot
		case query_action_get:
			val = &Snapshot{
				Hosts:    fsm.store.HostList(),
				Index:    fsm.index,
				Replicas: fsm.store.ReplicaList(),
				Shards:   fsm.store.ShardList(),
			}
		default:
			err = fmt.Errorf("Unrecognized snapshot query %s", query.Action)
		}
	} else {
		err = fmt.Errorf("Invalid query %#v", e)
	}

	return
}

// TODO - Switch to jsonl
func (fsm *fsm) SaveSnapshot(w io.Writer, sfc dbsm.ISnapshotFileCollection, stopc <-chan struct{}) (err error) {
	b, err := json.Marshal(Snapshot{
		Hosts:    fsm.store.HostList(),
		Index:    fsm.index,
		Replicas: fsm.store.ReplicaList(),
		Shards:   fsm.store.ShardList(),
	})
	if err == nil {
		_, err = io.Copy(w, bytes.NewReader(b))
	}

	return
}

// TODO - Switch to jsonl
func (fsm *fsm) RecoverFromSnapshot(r io.Reader, sfc []dbsm.SnapshotFile, stopc <-chan struct{}) (err error) {
	var data Snapshot
	if err = json.NewDecoder(r).Decode(&data); err != nil {
		return
	}
	for _, host := range data.Hosts {
		fsm.store.HostPut(host)
	}
	for _, shard := range data.Shards {
		fsm.store.ShardPut(shard)
	}
	for _, replica := range data.Replicas {
		fsm.store.ReplicaPut(replica)
	}
	fsm.index = data.Index
	return
}

func (fsm *fsm) Close() (err error) {
	return
}
