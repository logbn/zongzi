package zongzi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lni/dragonboat/v4"
)

type controller struct {
	agent  *Agent
	cancel context.CancelFunc
	mutex  sync.RWMutex
	leader bool
	index  uint64
}

func newController(a *Agent) *controller {
	return &controller{
		agent: a,
	}
}

func (c *controller) Start() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(context.Background())
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				err = c.tick()
			case <-ctx.Done():
				return
			}
			if err != nil {
				c.agent.log.Errorf("controller: %v", err)
			}
		}
	}()
	return
}

type controllerStartReplicaParams struct {
	shardMembers map[uint64]string
	shardType    string
	shardID      uint64
	replicaID    uint64
	index        uint64
}

func (c *controller) tick() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var (
		found   = map[uint64]bool{}
		nhid    = c.agent.host.ID()
		toStart = []controllerStartReplicaParams{}
	)
	var ok bool
	var host Host
	hostInfo := c.agent.host.GetNodeHostInfo(dragonboat.NodeHostInfoOption{})
	var index uint64
	c.agent.Read(func(state State) {
		index = state.Index()
		host, ok = state.HostGet(nhid)
		if !ok {
			return
		}
		if host.Updated <= c.index {
			return
		}
		state.ReplicaIterateByHostID(host.ID, func(r Replica) bool {
			found[r.ID] = false
			return true
		})
		for _, info := range hostInfo.ShardInfoList {
			if replica, ok := state.ReplicaGet(info.ReplicaID); ok {
				found[replica.ID] = true
				if replica.Status == ReplicaStatus_Closed {
					// Remove replica
				}
				if info.IsNonVoting && !replica.IsNonVoting {
					// Promote to Voting
				}
				if !info.IsNonVoting && replica.IsNonVoting {
					// Demote to NonVoting
				}
			} else {
				// Remove raftNode
			}
		}
		for id, ok := range found {
			if ok {
				continue
			}
			replica, _ := state.ReplicaGet(id)
			if replica.Status == ReplicaStatus_New {
				if replica.ShardID == 0 {
					continue
				}
				shard, ok := state.ShardGet(replica.ShardID)
				if !ok {
					continue
				}
				members := map[uint64]string{}
				state.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
					if !r.IsNonVoting && !r.IsWitness {
						members[r.ID] = r.HostID
					}
					return true
				})
				toStart = append(toStart, controllerStartReplicaParams{
					index:        host.Updated,
					replicaID:    replica.ID,
					shardID:      shard.ID,
					shardType:    shard.Type,
					shardMembers: members,
				})
			}
		}
	}, true)
	for _, params := range toStart {
		item, ok := c.agent.shardTypes[params.shardType]
		if !ok {
			err = fmt.Errorf("Shard name not found in registry: %s", params.shardType)
			break
		}
		item.Config.ShardID = params.shardID
		item.Config.ReplicaID = params.replicaID
		if item.StateMachineFactory != nil {
			shim := stateMachineFactoryShim(item.StateMachineFactory)
			err = c.agent.host.StartConcurrentReplica(params.shardMembers, false, shim, item.Config)
		} else {
			shim := persistentStateMachineFactoryShim(item.PersistentStateMachineFactory)
			err = c.agent.host.StartOnDiskReplica(params.shardMembers, false, shim, item.Config)
		}
		if err != nil {
			err = fmt.Errorf("Failed to start replica: %w", err)
			break
		}
		var res Result
		res, err = c.agent.primePropose(newCmdReplicaUpdateStatus(params.replicaID, ReplicaStatus_Active))
		if err != nil || res.Value != 1 {
			err = fmt.Errorf("Failed to update replica status: %w", err)
			break
		}
	}
	if err == nil {
		c.index = index
	}
	return
}

func (c *controller) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *controller) LeaderUpdated(info LeaderInfo) {
	switch info.ShardID {
	case c.agent.replicaConfig.ShardID:
		c.mutex.Lock()
		defer c.mutex.Unlock()
		if info.LeaderID == info.ReplicaID {
			c.leader = true
		} else {
			c.leader = false
		}
	}
}
