package zongzi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/logger"
	"github.com/lni/dragonboat/v4/statemachine"
)

func fsmFactory(agent *Agent) statemachine.CreateOnDiskStateMachineFunc {
	return statemachine.CreateOnDiskStateMachineFunc(func(shardID, replicaID uint64) statemachine.IOnDiskStateMachine {
		node := &fsm{
			log:       agent.log,
			replicaID: replicaID,
			shardID:   shardID,
			state:     newState(),
		}
		agent.fsm = node
		return node
	})
}

var _ statemachine.IOnDiskStateMachine = (*fsm)(nil)

type fsm struct {
	log       logger.ILogger
	replicaID uint64
	shardID   uint64
	state     *State
}

func (fsm *fsm) Update(entries []Entry) ([]Entry, error) {
	var err error
	var cmds = make([]any, len(entries), 0)
	for i, entry := range entries {
		var cmdBase command
		if err = json.Unmarshal(entry.Cmd, &cmdBase); err != nil {
			fsm.log.Errorf("Invalid entry %#v, %w", entry, err)
			continue
		}
		switch cmdBase.Type {
		// Host
		case command_type_host:
			var cmd commandHost
			if err = json.Unmarshal(entry.Cmd, &cmd); err != nil {
				fsm.log.Errorf("Invalid host cmd %#v, %w", entry, err)
				break
			}
			cmds[i] = cmd
		// Shard
		case command_type_shard:
			var cmd commandShard
			if err = json.Unmarshal(entry.Cmd, &cmd); err != nil {
				fsm.log.Errorf("Invalid shard cmd %#v, %w", entry, err)
				break
			}
			cmds[i] = cmd
		// Replica
		case command_type_replica:
			var cmd commandReplica
			if err = json.Unmarshal(entry.Cmd, &cmd); err != nil {
				fsm.log.Errorf("Invalid replica cmd %#v, %w", entry, err)
				break
			}
			cmds[i] = cmd
		default:
			fsm.log.Errorf("Unrecognized cmd type %s", cmdBase.Type)
		}
	}
	fsm.state.mutex.Lock()
	defer fsm.state.mutex.Unlock()
	for i, entry := range entries {
		switch cmds[i].(type) {
		// Host
		case commandHost:
			var cmd = cmds[i].(commandHost)
			switch cmd.Action {
			// Put
			case command_action_put:
				if old, ok := fsm.state.Hosts.Get(cmd.Host.ID); ok {
					cmd.Host.Replicas = old.Replicas
				} else {
					cmd.Host.Created = entry.Index
				}
				cmd.Host.Updated = entry.Index
				fsm.state.Hosts.Set(cmd.Host.ID, &cmd.Host)
				entries[i].Result.Value = 1
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
				entries[i].Result.Value = 1
			default:
				err = fmt.Errorf("Unrecognized host action %s", cmd.Action)
			}
		// Shard
		case commandShard:
			var cmd = cmds[i].(commandShard)
			switch cmd.Action {
			// Post
			case command_action_post:
				fsm.state.ShardIndex++
				cmd.Shard.ID = fsm.state.ShardIndex
				cmd.Shard.Created = entry.Index
				cmd.Shard.Updated = entry.Index
				fsm.state.Shards.Set(cmd.Shard.ID, &cmd.Shard)
				entries[i].Result.Value = cmd.Shard.ID
			// Put
			case command_action_put:
				if cmd.Shard.ID > fsm.state.ShardIndex {
					fsm.log.Warningf("%w: %#v", ErrIDOutOfRange, cmd)
					break
				}
				if old, ok := fsm.state.Shards.Get(cmd.Shard.ID); ok {
					cmd.Shard.Replicas = old.Replicas
				} else {
					cmd.Shard.Created = entry.Index
				}
				cmd.Shard.Updated = entry.Index
				fsm.state.Shards.Set(cmd.Shard.ID, &cmd.Shard)
				entries[i].Result.Value = 1
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
				entries[i].Result.Value = 1
			default:
				err = fmt.Errorf("Unrecognized shard action %s", cmd.Action)
			}
		// Replica
		case commandReplica:
			var cmd = cmds[i].(commandReplica)
			switch cmd.Action {
			// Put
			case command_action_put:
				host, ok := fsm.state.Hosts.Get(cmd.Replica.HostID)
				if !ok {
					fsm.log.Warningf("%w: %#v", ErrHostNotFound, cmd)
					break
				}
				shard, ok := fsm.state.Shards.Get(cmd.Replica.ShardID)
				if !ok {
					fsm.log.Warningf("%w: %#v", ErrShardNotFound, cmd)
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
					fsm.log.Warningf("%w: %#v", ErrIDOutOfRange, cmd)
					break
				}
				fsm.state.Replicas.Set(cmd.Replica.ID, &cmd.Replica)
				entries[i].Result.Value = cmd.Replica.ID
			// Delete
			case command_action_del:
				replica, ok := fsm.state.Replicas.Get(cmd.Replica.ID)
				if !ok {
					fsm.log.Warningf("%w: %#v", ErrReplicaNotFound, cmd)
					break
				}
				host, _ := fsm.state.Hosts.Get(replica.HostID)
				host.Replicas = sliceWithout(host.Replicas, replica)
				host.Updated = entry.Index
				shard, _ := fsm.state.Shards.Get(replica.ShardID)
				shard.Replicas = sliceWithout(shard.Replicas, replica)
				shard.Updated = entry.Index
				fsm.state.Replicas.Delete(cmd.Replica.ID)
				entries[i].Result.Value = 1
			default:
				fsm.log.Errorf("Unrecognized replica action: %s - %#v", cmd.Action, cmd)
			}
		}
		fsm.state.Index = entry.Index
	}

	return entries, nil
}

func (fsm *fsm) PrepareSnapshot() (cursor any, err error) {
	fsm.state.mutex.RLock()
	return
}

func (fsm *fsm) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) (err error) {
	defer fsm.state.mutex.RUnlock()
	b, err := fsm.state.MarshalJson()
	if err == nil {
		_, err = io.Copy(w, bytes.NewReader(b))
	}
	return
}

func (fsm *fsm) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) (err error) {
	fsm.state.mutex.Lock()
	defer fsm.state.mutex.Unlock()
	b, err := io.ReadAll(r)
	if err != nil {
		return
	}
	return fsm.state.UnmarshalJSON(b)
}

func (fsm *fsm) Lookup(interface{}) (res interface{}, err error)  { return }
func (fsm *fsm) Close() (err error)                               { return }
func (fsm *fsm) Open(close <-chan struct{}) (i uint64, err error) { return }
func (fsm *fsm) Sync() (err error)                                { return }
