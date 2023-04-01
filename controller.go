package zongzi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/logbn/zongzi/internal"
)

type controller struct {
	agent  *Agent
	ctx    context.Context
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
		t := time.NewTicker(waitPeriod)
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

func (c *controller) tick() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var index uint64
	var hadErr bool
	c.agent.Read(func(state State) {
		index = state.Index()
		if index <= c.index {
			return
		}
		var found = map[uint64]bool{}
		state.ReplicaIterateByHostID(c.agent.HostID(), func(r Replica) bool {
			found[r.ID] = false
			return true
		})
		hostInfo := c.agent.host.GetNodeHostInfo(nodeHostInfoOption{})
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
			if replica.ShardID == 0 {
				continue
			}
			shard, ok := state.ShardGet(replica.ShardID)
			if !ok {
				hadErr = true
				err = fmt.Errorf("Shard not found")
				continue
			}
			item, ok := c.agent.shardTypes[shard.Type]
			if !ok {
				c.agent.log.Warningf("Shard name not found in registry: %s", shard.Type)
				continue
			}
			members := state.ShardMembers(shard.ID)
			item.Config.ShardID = shard.ID
			item.Config.ReplicaID = replica.ID
			item.Config.IsNonVoting = replica.IsNonVoting
			c.agent.log.Debugf("[%05d:%05d] Controller: Starting replica: %s", shard.ID, replica.ID, shard.Type)
			if item.StateMachineFactory != nil {
				shim := stateMachineFactoryShim(item.StateMachineFactory)
				switch replica.Status {
				case ReplicaStatus_Bootstrapping:
					if len(members) < 3 {
						err = fmt.Errorf("Not enough members")
						break
					}
					err = c.agent.host.StartConcurrentReplica(members, false, shim, item.Config)
				case ReplicaStatus_Joining:
					res := c.requestShardJoin(members, shard.ID, replica.ID, replica.IsNonVoting)
					if res == 0 {
						err = fmt.Errorf(`[%05d:%05d] Unable to join shard`, shard.ID, replica.ID)
						break
					}
					err = c.agent.host.StartConcurrentReplica(nil, true, shim, item.Config)
				case ReplicaStatus_Active:
					err = c.agent.host.StartConcurrentReplica(nil, false, shim, item.Config)
				}
			} else {
				shim := persistentStateMachineFactoryShim(item.PersistentStateMachineFactory)
				switch replica.Status {
				case ReplicaStatus_Bootstrapping:
					if len(members) < 3 {
						err = fmt.Errorf("Not enough members")
						break
					}
					err = c.agent.host.StartOnDiskReplica(members, false, shim, item.Config)
				case ReplicaStatus_Joining:
					res := c.requestShardJoin(members, shard.ID, replica.ID, replica.IsNonVoting)
					if res == 0 {
						err = fmt.Errorf(`[%05d:%05d] Unable to join persistent shard`, shard.ID, replica.ID)
						break
					}
					err = c.agent.host.StartOnDiskReplica(nil, true, shim, item.Config)
				case ReplicaStatus_Active:
					err = c.agent.host.StartOnDiskReplica(nil, false, shim, item.Config)
				}
			}
			if err != nil {
				hadErr = true
				c.agent.log.Warningf("Failed to start replica: %v", err)
				continue
			}
			var res Result
			res, err = c.agent.primePropose(newCmdReplicaUpdateStatus(replica.ID, ReplicaStatus_Active))
			if err != nil || res.Value != 1 {
				hadErr = true
				c.agent.log.Warningf("Failed to update replica status: %v", err)
			}
		}
	}, true)
	if !hadErr {
		c.index = index
	}
	return
}

// requestShardJoin requests host replica be added to a shard
func (c *controller) requestShardJoin(members map[uint64]string, shardID, replicaID uint64, isNonVoting bool) (v uint64) {
	c.agent.log.Debugf("[%05d:%05d] Joining shard (isNonVoting: %v)", shardID, replicaID, isNonVoting)
	var res *internal.ShardJoinResponse
	var host Host
	var ok bool
	var err error
	for _, hostID := range members {
		c.agent.Read(func(s State) {
			host, ok = s.HostGet(hostID)
		})
		if !ok {
			c.agent.log.Warningf(`Host not found %s`, hostID)
			continue
		}
		res, err = c.agent.grpcClientPool.get(host.ApiAddress).ShardJoin(raftCtx(), &internal.ShardJoinRequest{
			HostId:      c.agent.HostID(),
			ShardId:     shardID,
			ReplicaId:   replicaID,
			IsNonVoting: isNonVoting,
		})
		if err != nil {
			c.agent.log.Warningf(`[%05d:%05d] %s Unable to join shard (%v): %v`, shardID, replicaID, c.agent.HostID(), isNonVoting, err)
		}
		if res != nil && res.Value > 0 {
			v = res.Value
			break
		}
	}
	if err != nil {
		c.agent.log.Warningf(`Unable to join shard: %v`, err)
	}
	if res != nil {
		v = res.Value
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
