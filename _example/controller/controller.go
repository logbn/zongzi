package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/logbn/zongzi"
)

type controller struct {
	agent  zongzi.Agent
	cancel context.CancelFunc
	mutex  sync.RWMutex
	leader bool
}

func (c *controller) start() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.leader {
		return
	}
	var (
		ctx      context.Context
		index    uint64
		shard    *zongzi.Shard
		snapshot zongzi.Snapshot
	)
	ctx, c.cancel = context.WithCancel(context.Background())
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		var err error
		for {
			select {
			case <-t.C:
				var replicas = map[uint64]*zongzi.Replica{}
				snapshot, err = c.agent.GetSnapshot(index)
				if err != nil || snapshot == nil {
					break
				}
				if shard == nil {
					// Find shard (this controller supports only one shard of its type per cluster)
					for _, s := range snapshot.Shards {
						if s.Type == shardType && s.Status != zongzi.ShardStatus_Closed {
							shard = s
							break
						}
					}
					// Shard does not yet exist. Create it.
					if shard == nil {
						shard, err = c.agent.CreateShard(shardType)
						if err != nil {
							break
						}
					}
				}
				for _, r := range snapshot.Replicas {
					if r.ShardID == shard.ID {
						replicas[r.ID] = r
					}
				}
				// Add replicas to new hosts and remove replicas for missing hosts
				var zones = map[string]bool{}
				for _, h := range snapshot.Hosts {
					var meta map[string]any
					if err = json.Unmarshal(h.Meta, &meta); err != nil {
						break
					}
					var hasReplica bool
					for replicaID, shardID := range h.Replicas {
						if shardID == shard.ID {
							if !replicas[replicaID].IsNonVoting {
								zones[h.Meta["zone"].(string)] = true
							}
							hasReplica = true
							break
						}
					}
					if hasReplica {
						continue
					}
					// Not strictly correct.
					// Need to evaulate all hosts to ensure no members already exist in this zone.
					if _, ok := zones[h.Meta["zone"].(string)]; !ok {
						zones[h.Meta["zone"].(string)] = true
						if _, err = c.agent.CreateReplica(shard.ID, h.ID, false); err != nil {
							return
						}
					} else {
						if _, err = c.agent.CreateReplica(shard.ID, h.ID, true); err != nil {
							return
						}
					}
				}
				index = snapshot.Index
			case <-ctx.Done():
				c.mutex.Lock()
				defer c.mutex.Unlock()
				c.leader = false
				return
			}
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
		}
	}()
	c.leader = true
}

func (c *controller) stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.leader {
		c.cancel()
	}
}

func (c *controller) LeaderUpdated(info zongzi.LeaderInfo) {
	switch info.ShardID {
	case 1:
		log.Printf("[RAFT EVENT] LeaderUpdated: %#v", info)
		if info.LeaderID == info.ReplicaID {
			c.start()
		} else {
			c.stop()
		}
	}
}
