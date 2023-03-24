package zongzi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/logger"
	"github.com/lni/dragonboat/v4/statemachine"
)

func fsmFactory(agent *Agent) statemachine.CreateStateMachineFunc {
	return func(shardID, replicaID uint64) statemachine.IStateMachine {
		node := &fsm{
			log:       agent.log,
			replicaID: replicaID,
			shardID:   shardID,
			state:     newState(),
		}
		agent.fsm = node
		return node
	}
}

var _ statemachine.IStateMachine = (*fsm)(nil)

type fsm struct {
	log       logger.ILogger
	replicaID uint64
	shardID   uint64
	state     *State
}

func (fsm *fsm) Update(entry Entry) (Result, error) {
	var err error
	var cmd any
	var cmdBase command
	if err = json.Unmarshal(entry.Cmd, &cmdBase); err != nil {
		fsm.log.Errorf("Invalid entry %#v, %v", entry, err)
		return entry.Result, nil
	}
	switch cmdBase.Type {
	// Host
	case command_type_host:
		var cmdHost commandHost
		if err = json.Unmarshal(entry.Cmd, &cmdHost); err != nil {
			fsm.log.Errorf("Invalid host cmd %#v, %v", entry, err)
			break
		}
		cmd = cmdHost
	// Shard
	case command_type_shard:
		var cmdShard commandShard
		if err = json.Unmarshal(entry.Cmd, &cmdShard); err != nil {
			fsm.log.Errorf("Invalid shard cmd %#v, %v", entry, err)
			break
		}
		cmd = cmdShard
	// Replica
	case command_type_replica:
		var cmdReplica commandReplica
		if err = json.Unmarshal(entry.Cmd, &cmdReplica); err != nil {
			fsm.log.Errorf("Invalid replica cmd %#v, %v", entry, err)
			break
		}
		cmd = cmdReplica
	default:
		fsm.log.Errorf("Unrecognized cmd type %s", cmdBase.Type, cmdBase)
	}
	fsm.state.mutex.Lock()
	defer fsm.state.mutex.Unlock()
	switch cmd.(type) {
	// Host
	case commandHost:
		var cmd = cmd.(commandHost)
		switch cmd.Action {
		// Put
		case command_action_put:
			if old, ok := fsm.state.Hosts.Get(cmd.Host.ID); ok {
				cmd.Host.Replicas = old.Replicas
				cmd.Host.Created = old.Created
			} else {
				cmd.Host.Created = entry.Index
			}
			cmd.Host.Updated = entry.Index
			fsm.state.Hosts.Set(cmd.Host.ID, &cmd.Host)
			entry.Result.Value = 1
		// Delete
		case command_action_del:
			host, ok := fsm.state.Hosts.Get(cmd.Host.ID)
			if !ok {
				break
			}
			for _, replica := range host.Replicas {
				shard, _ := fsm.state.Hosts.Get(replica.HostID)
				shard.Replicas = sliceWithout(shard.Replicas, replica)
				shard.Updated = entry.Index
				fsm.state.Replicas.Delete(replica.ID)
			}
			fsm.state.Hosts.Delete(host.ID)
			entry.Result.Value = 1
		default:
			err = fmt.Errorf("Unrecognized host action %s", cmd.Action)
		}
	// Shard
	case commandShard:
		var cmd = cmd.(commandShard)
		switch cmd.Action {
		// Post
		case command_action_post:
			fsm.state.ShardIndex++
			cmd.Shard.ID = fsm.state.ShardIndex
			cmd.Shard.Created = entry.Index
			cmd.Shard.Updated = entry.Index
			fsm.state.Shards.Set(cmd.Shard.ID, &cmd.Shard)
			entry.Result.Value = cmd.Shard.ID
		// Put
		case command_action_put:
			if cmd.Shard.ID > fsm.state.ShardIndex {
				fsm.log.Warningf("%v: %#v", ErrIDOutOfRange, cmd)
				break
			}
			if old, ok := fsm.state.Shards.Get(cmd.Shard.ID); ok {
				cmd.Shard.Replicas = old.Replicas
				cmd.Shard.Created = old.Created
			} else {
				cmd.Shard.Created = entry.Index
			}
			cmd.Shard.Updated = entry.Index
			fsm.state.Shards.Set(cmd.Shard.ID, &cmd.Shard)
			entry.Result.Value = 1
		// Delete
		case command_action_del:
			shard, ok := fsm.state.Shards.Get(cmd.Shard.ID)
			if !ok {
				break
			}
			for _, replica := range shard.Replicas {
				host, _ := fsm.state.Hosts.Get(replica.HostID)
				host.Replicas = sliceWithout(host.Replicas, replica)
				host.Updated = entry.Index
				fsm.state.Replicas.Delete(replica.ID)
			}
			fsm.state.Shards.Delete(shard.ID)
			entry.Result.Value = 1
		default:
			err = fmt.Errorf("Unrecognized shard action %s", cmd.Action)
		}
	// Replica
	case commandReplica:
		var cmd = cmd.(commandReplica)
		switch cmd.Action {
		// Put
		case command_action_put:
			host, ok := fsm.state.Hosts.Get(cmd.Replica.HostID)
			if !ok {
				fsm.log.Warningf("%v: %#v", ErrHostNotFound, cmd)
				break
			}
			shard, ok := fsm.state.Shards.Get(cmd.Replica.ShardID)
			if !ok {
				fsm.log.Warningf("%v: %#v", ErrShardNotFound, cmd)
				break
			}
			cmd.Replica.Updated = entry.Index
			if cmd.Replica.ID == 0 {
				fsm.state.ReplicaIndex++
				cmd.Replica.ID = fsm.state.ReplicaIndex
				cmd.Replica.Created = entry.Index
				cmd.Replica.Host = host
				cmd.Replica.Shard = shard
				host.Replicas = append(host.Replicas, &cmd.Replica)
				host.Updated = entry.Index
				shard.Replicas = append(shard.Replicas, &cmd.Replica)
				shard.Updated = entry.Index
			} else if cmd.Replica.ID > fsm.state.ReplicaIndex {
				fsm.log.Warningf("%v: %#v", ErrIDOutOfRange, cmd)
				break
			}
			fsm.state.Replicas.Set(cmd.Replica.ID, &cmd.Replica)
			entry.Result.Value = cmd.Replica.ID
		// Delete
		case command_action_del:
			replica, ok := fsm.state.Replicas.Get(cmd.Replica.ID)
			if !ok {
				fsm.log.Warningf("%v: %#v", ErrReplicaNotFound, cmd)
				break
			}
			host, _ := fsm.state.Hosts.Get(replica.HostID)
			host.Replicas = sliceWithout(host.Replicas, replica)
			host.Updated = entry.Index
			shard, _ := fsm.state.Shards.Get(replica.ShardID)
			shard.Replicas = sliceWithout(shard.Replicas, replica)
			shard.Updated = entry.Index
			fsm.state.Replicas.Delete(cmd.Replica.ID)
			entry.Result.Value = 1
		default:
			fsm.log.Errorf("Unrecognized replica action: %s - %#v", cmd.Action, cmd)
		}
	}
	fsm.state.Index = entry.Index

	return entry.Result, nil
}

// func (fsm *fsm) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) (err error) {
func (fsm *fsm) SaveSnapshot(w io.Writer, _ statemachine.ISnapshotFileCollection, _ <-chan struct{}) error {
	b, err := fsm.state.MarshalJSON()
	if err == nil {
		_, err = io.Copy(w, bytes.NewReader(b))
	}
	return nil
}

// func (fsm *fsm) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) (err error) {
func (fsm *fsm) RecoverFromSnapshot(r io.Reader, _ []statemachine.SnapshotFile, _ <-chan struct{}) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil
	}
	return fsm.state.UnmarshalJSON(b)
}

func (fsm *fsm) Lookup(interface{}) (res interface{}, err error) { return }
func (fsm *fsm) Close() (err error)                              { return }
