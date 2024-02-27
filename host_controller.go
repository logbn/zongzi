package zongzi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/logbn/zongzi/internal"
)

// The hostController starts and stops replicas based on replica config.
type hostController struct {
	agent     *Agent
	ctx       context.Context
	ctxCancel context.CancelFunc
	mutex     sync.RWMutex
	index     uint64
}

func newHostController(a *Agent) *hostController {
	return &hostController{
		agent: a,
	}
}

func (c *hostController) Start() (err error) {
	c.mutex.Lock()
	c.ctx, c.ctxCancel = context.WithCancel(context.Background())
	go func() {
		t := time.NewTicker(waitPeriod)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				err = c.tick()
			case <-c.ctx.Done():
				return
			}
			if err != nil {
				c.agent.log.Errorf("Host Controller: %v", err)
			}
		}
	}()
	c.mutex.Unlock()
	return c.tick()
}

func (c *hostController) tick() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var index uint64
	var hadErr bool
	var shard Shard
	c.agent.Read(c.ctx, func(state *State) {
		host, ok := state.Host(c.agent.HostID())
		if !ok {
			hadErr = true
			c.agent.log.Warningf("Host not found %s", c.agent.HostID())
			return
		}
		index = host.Updated
		if index <= c.index {
			return
		}
		// These are all the replicas that SHOULD exist on the host
		var found = map[uint64]bool{}
		state.ReplicaIterateByHostID(c.agent.HostID(), func(r Replica) bool {
			if r.ShardID > 0 {
				found[r.ID] = false
			}
			return true
		})
		// These are all the replicas that DO exist on the host
		hostInfo := c.agent.host.GetNodeHostInfo(nodeHostInfoOption{})
		for _, info := range hostInfo.ShardInfoList {
			if replica, ok := state.Replica(info.ReplicaID); ok {
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
			} else if replica.ShardID > 0 {
				// Remove raftNode
			}
		}
		// This creates all the missing replicas
		for id, ok := range found {
			if ok {
				continue
			}
			replica, ok := state.Replica(id)
			if !ok {
				hadErr = true
				c.agent.log.Warningf("Replica not found")
				continue
			}
			shard, ok = state.Shard(replica.ShardID)
			if !ok {
				hadErr = true
				c.agent.log.Warningf("Shard not found")
				continue
			}
			item, ok := c.agent.shardTypes[shard.Type]
			if !ok {
				c.agent.log.Warningf("Shard name not found in registry: %s", shard.Type)
				continue
			}
			// err := c.add(shard, replica)
			members := state.ShardMembers(shard.ID)
			item.Config.ShardID = shard.ID
			item.Config.ReplicaID = replica.ID
			item.Config.IsNonVoting = replica.IsNonVoting
			c.agent.log.Infof("[%05d:%05d] Host Controller: Starting replica: %s", shard.ID, replica.ID, shard.Type)
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
				shim := stateMachinePersistentFactoryShim(item.StateMachinePersistentFactory)
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
	})
	if index <= c.index || hadErr {
		err = nil
		return
	}
	c.agent.log.Debugf("%s Finished processing %d", c.agent.HostID(), index)
	c.index = index
	return
}

// requestShardJoin requests host replica be added to a shard
func (c *hostController) requestShardJoin(members map[uint64]string, shardID, replicaID uint64, isNonVoting bool) (v uint64) {
	c.agent.log.Debugf("[%05d:%05d] Joining shard (isNonVoting: %v)", shardID, replicaID, isNonVoting)
	var res *internal.ShardJoinResponse
	var host Host
	var ok bool
	var err error
	for _, hostID := range members {
		c.agent.Read(c.ctx, func(s *State) {
			host, ok = s.Host(hostID)
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
			c.agent.log.Warningf(`[%05d:%05d] %s | %s Unable to join shard (%v): %v`, shardID, replicaID, c.agent.HostID(), hostID, isNonVoting, err)
		}
		if err == nil && res != nil && res.Value > 0 {
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

func (c *hostController) Stop() {
	defer c.agent.log.Infof(`Stopped hostController`)
	if c.ctxCancel != nil {
		c.ctxCancel()
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.index = 0
}
