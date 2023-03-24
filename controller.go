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
	if !c.leader {
		return
	}
	var (
		found   = map[uint64]bool{}
		nhid    = c.agent.host.ID()
		toStart = []controllerStartReplicaParams{}
	)
	hostInfo := c.agent.host.GetNodeHostInfo(dragonboat.NodeHostInfoOption{})
	c.agent.Read(func(state *State) {
		host, ok := state.Hosts.Get(nhid)
		if !ok {
			return
		}
		if host.Updated <= c.index {
			return
		}
		b, _ := state.MarshalJSON()
		c.agent.log.Errorf("controller: snapshot: %s", string(b))
		for _, r := range host.Replicas {
			if r.ID == 0 {
				continue
			}
			found[r.ID] = false
		}
		for _, info := range hostInfo.ShardInfoList {
			if replica, ok := state.Replicas.Get(info.ReplicaID); ok {
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
			replica, _ := state.Replicas.Get(id)
			if replica.Status == ReplicaStatus_New {
				toStart = append(toStart, controllerStartReplicaParams{
					index:        host.Updated,
					replicaID:    replica.ID,
					shardMembers: replica.Shard.Members(),
					shardID:      replica.Shard.ID,
					shardType:    replica.Shard.Type,
				})
			}
		}
		if len(toStart) == 0 {
			c.index = state.Index
		}
	})
	for _, params := range toStart {
		item, ok := c.agent.shardTypes[params.shardType]
		if !ok {
			err = fmt.Errorf("Shard name not found in registry: %s", params.shardType)
			break
		}
		item.Config.ShardID = params.shardID
		item.Config.ReplicaID = params.replicaID
		err = c.agent.host.StartOnDiskReplica(params.shardMembers, false, stateMachineFactoryShim(item.Factory), item.Config)
		if err != nil {
			err = fmt.Errorf("Failed to start replica: %w", err)
			break
		}
		// newCmdSetReplicaStatus(id, ReplicaStatus_Active)
		c.index = params.index
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
