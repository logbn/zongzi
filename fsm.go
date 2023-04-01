package zongzi

import (
	"encoding/json"
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
			state:     newFsmStateRadix(),
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
	state     State
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
	// fsm.log.Debugf(`Update: %d %s`, entry.Index, string(entry.Cmd))
	state := fsm.state.withTxn(true)
	defer state.commit()
	switch cmd.(type) {
	// Host
	case commandHost:
		var cmd = cmd.(commandHost)
		switch cmd.Action {
		// Put
		case command_action_put:
			if old, ok := state.HostGet(cmd.Host.ID); ok {
				cmd.Host.Created = old.Created
			} else {
				cmd.Host.Created = entry.Index
			}
			cmd.Host.Updated = entry.Index
			state.ReplicaIterateByHostID(cmd.Host.ID, func(r Replica) bool {
				state.shardTouch(r.ShardID, entry.Index)
				return true
			})
			state.hostPut(cmd.Host)
			entry.Result.Value = 1
		// Delete
		case command_action_del:
			host, ok := state.HostGet(cmd.Host.ID)
			if !ok {
				fsm.log.Warningf("%v: %#v", ErrHostNotFound, cmd)
				break
			}
			state.ReplicaIterateByHostID(host.ID, func(r Replica) bool {
				state.replicaDelete(r)
				state.shardTouch(r.ShardID, entry.Index)
				return true
			})
			state.hostDelete(host)
			entry.Result.Value = 1
		default:
			fsm.log.Errorf("Unrecognized host action: %s - %#v", cmd.Action, cmd)
		}
	// Shard
	case commandShard:
		var cmd = cmd.(commandShard)
		switch cmd.Action {
		// Post
		case command_action_post:
			cmd.Shard.ID = state.shardIncr()
			cmd.Shard.Created = entry.Index
			cmd.Shard.Updated = entry.Index
			state.shardPut(cmd.Shard)
			entry.Result.Value = cmd.Shard.ID
		// Put
		case command_action_put:
			if old, ok := state.ShardGet(cmd.Shard.ID); ok {
				cmd.Shard.Created = old.Created
			} else if cmd.Shard.ID == 0 && state.metaGet(`shardIndex`) == 0 {
				cmd.Shard.Created = entry.Index
			} else {
				fsm.log.Errorf("%s: %s - %#v", ErrShardNotFound, cmd.Action, cmd)
				break
			}
			cmd.Shard.Updated = entry.Index
			state.shardPut(cmd.Shard)
			state.ReplicaIterateByShardID(cmd.Shard.ID, func(r Replica) bool {
				state.hostTouch(r.HostID, entry.Index)
				return true
			})
			entry.Result.Value = 1
		// Delete
		case command_action_del:
			shard, ok := state.ShardGet(cmd.Shard.ID)
			if !ok {
				fsm.log.Warningf("%v: %#v", ErrShardNotFound, cmd)
				break
			}
			state.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
				state.replicaDelete(r)
				state.hostTouch(r.HostID, entry.Index)
				return true
			})
			state.shardDelete(shard)
			entry.Result.Value = 1
		default:
			fsm.log.Errorf("Unrecognized shard action: %s - %#v", cmd.Action, cmd)
		}
	// Replica
	case commandReplica:
		var cmd = cmd.(commandReplica)
		switch cmd.Action {
		// Post
		case command_action_post:
			cmd.Replica.ID = state.replicaIncr()
			cmd.Replica.Created = entry.Index
			cmd.Replica.Updated = entry.Index
			if !cmd.Replica.IsNonVoting && len(state.ShardMembers(cmd.Replica.ShardID)) < 3 {
				cmd.Replica.Status = ReplicaStatus_Bootstrapping
			} else {
				cmd.Replica.Status = ReplicaStatus_Joining
			}
			state.replicaPut(cmd.Replica)
			state.hostTouch(cmd.Replica.HostID, entry.Index)
			state.shardTouch(cmd.Replica.ShardID, entry.Index)
			entry.Result.Value = cmd.Replica.ID
		// Status Update
		case command_action_status_update:
			replica, ok := state.ReplicaGet(cmd.Replica.ID)
			if !ok {
				fsm.log.Warningf("%v: %#v %#v", ErrReplicaNotFound, cmd, replica)
				break
			}
			replica.Status = cmd.Replica.Status
			state.replicaPut(replica)
			state.shardTouch(replica.ShardID, entry.Index)
			entry.Result.Value = 1
		// Put
		case command_action_put:
			if old, ok := state.ReplicaGet(cmd.Replica.ID); ok {
				cmd.Replica.Created = old.Created
			} else {
				fsm.log.Errorf("%s: %s - %#v", ErrReplicaNotFound, cmd.Action, cmd)
				break
			}
			cmd.Replica.Updated = entry.Index
			state.replicaPut(cmd.Replica)
			state.hostTouch(cmd.Replica.HostID, entry.Index)
			state.shardTouch(cmd.Replica.ShardID, entry.Index)
			entry.Result.Value = 1
		// Delete
		case command_action_del:
			replica, ok := state.ReplicaGet(cmd.Replica.ID)
			if !ok {
				fsm.log.Warningf("%v: %#v", ErrReplicaNotFound, cmd)
				break
			}
			state.hostTouch(replica.HostID, entry.Index)
			state.shardTouch(replica.ShardID, entry.Index)
			state.replicaDelete(replica)
			entry.Result.Value = 1
		default:
			fsm.log.Errorf("Unrecognized replica action: %s - %#v", cmd.Action, cmd)
		}
	}
	state.metaSetIndex(entry.Index)

	return entry.Result, nil
}

// func (fsm *fsm) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) (err error) {
func (fsm *fsm) SaveSnapshot(w io.Writer, _ statemachine.ISnapshotFileCollection, _ <-chan struct{}) error {
	return fsm.state.Save(w)
}

// func (fsm *fsm) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) (err error) {
func (fsm *fsm) RecoverFromSnapshot(r io.Reader, _ []statemachine.SnapshotFile, _ <-chan struct{}) error {
	return fsm.state.recover(r)
}

func (fsm *fsm) Lookup(interface{}) (res interface{}, err error) { return }
func (fsm *fsm) Close() (err error)                              { return }
