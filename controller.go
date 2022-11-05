package zongzi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lni/dragonboat/v4"
)

type controller struct {
	agent     *agent
	cancel    context.CancelFunc
	mutex     sync.RWMutex
	leader    bool
	index     uint64
	logReader ReadonlyLogReader
}

func newController(a *agent) *controller {
	return &controller{
		agent: a,
	}
}

func (c *controller) Start() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(context.Background())
	c.logReader, err = c.agent.host.GetLogReader(c.agent.primeConfig.ShardID)
	if err != nil {
		return
	}
	go func() {
		t := time.NewTicker(time.Second)
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
	// Short-circuit reconciliation if no new changes to cluster state
	_, index := c.logReader.GetRange()
	if index <= c.index {
		return
	}
	var (
		snapshot Snapshot
		host     *Host

		shards   = map[uint64]*Shard{}
		replicas = map[uint64]*Replica{}
		found    = map[uint64]bool{}
		nhid     = c.agent.host.ID()
	)
	snapshot, err = c.agent.GetSnapshot()
	if err != nil {
		return
	}
	for _, h := range snapshot.Hosts {
		if h.ID == nhid {
			host = h
			break
		}
	}
	if host == nil {
		err = fmt.Errorf("Host not found")
		return
	}
	for _, r := range snapshot.Replicas {
		if _, ok := host.Replicas[r.ID]; ok {
			replicas[r.ID] = r
			found[r.ID] = false
		}
	}
	for _, s := range snapshot.Shards {
		shards[s.ID] = s
	}
	hostInfo := c.agent.host.GetNodeHostInfo(dragonboat.NodeHostInfoOption{})
	for _, info := range hostInfo.ShardInfoList {
		if r, ok := replicas[info.ReplicaID]; ok {
			found[r.ID] = true
			if r.Status == ReplicaStatus_Gone {
				// Remove replica
			}
			if info.IsNonVoting && !r.IsNonVoting {
				// Promote to Voting
			}
			if !info.IsNonVoting && r.IsNonVoting {
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
		r := replicas[id]
		shard := shards[r.ShardID]
		if r.Status == ReplicaStatus_New {
			item, ok := c.agent.shardTypes[shard.Type]
			if !ok {
				err = fmt.Errorf("Shard name not found in registry %s (%#v)", shard.Type, r)
				break
			}
			item.Config.ShardID = r.ShardID
			item.Config.ReplicaID = r.ID
			err = c.agent.host.StartReplica(shard.Replicas, false, item.Factory, item.Config)
			if err != nil {
				err = fmt.Errorf("Failed to start replica: %w", err)
				break
			}
			// CMD_ReplicaStatus{id, ReplicaStatus_Ready}
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
	case c.agent.primeConfig.ShardID:
		c.mutex.Lock()
		defer c.mutex.Unlock()
		if info.LeaderID == info.ReplicaID {
			c.leader = true
		} else {
			c.leader = false
		}
	}
}
