package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/benbjohnson/clock"

	"github.com/logbn/zongzi"
)

func newController() *controller {
	return &controller{
		clock: clock.New(),
	}
}

type controller struct {
	agent  *zongzi.Agent
	cancel context.CancelFunc
	clock  clock.Clock
	leader bool
	mutex  sync.RWMutex
}

func (c *controller) start() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.leader {
		return
	}
	var (
		ctx      context.Context
		err      error
		index    uint64
		shard    *zongzi.Shard
		snapshot *zongzi.Snapshot
	)
	ctx, c.cancel = context.WithCancel(context.Background())
	go func() {
		t := c.clock.Ticker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				replicas := map[uint64]*zongzi.Replica{}
				c.agent.Read(func(s *zongzi.State) {
					snapshot = s.GetSnapshot()
				})
				if snapshot.Index == 0 || snapshot.Index == index {
					break
				}
				if shard == nil {
					// Find shard (this controller supports only one shard of its type per cluster)
					for _, s := range snapshot.Shards {
						if s.Type == StateMachineUri && s.Status != zongzi.ShardStatus_Closed {
							shard = &s
							break
						}
					}
					// Shard does not yet exist. Create it.
					if shard == nil {
						shard, err = c.agent.CreateShard(StateMachineUri, StateMachineVersion)
						if err != nil {
							break
						}
					}
				}
				for _, r := range snapshot.Replicas {
					if r.ShardID == shard.ID {
						replicas[r.ID] = &r
					}
				}
				// Add replicas to new hosts and remove replicas for missing hosts
				var zones = map[string]bool{}
				for _, h := range snapshot.Hosts {
					var meta map[string]any
					if err = json.Unmarshal(h.Meta, &meta); err != nil {
						err = fmt.Errorf("Bad meta: %w", err)
						break
					}
					var hasReplica bool
					for _, replica := range h.Replicas {
						if replica.ShardID == shard.ID {
							if !replica.IsNonVoting {
								zones[meta["zone"].(string)] = true
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
					if _, ok := zones[meta["zone"].(string)]; !ok {
						zones[meta["zone"].(string)] = true
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
				c.agent.Read(func(s *zongzi.State) {
					b, _ := s.MarshalJSON()
					log.Println(string(b))
				})
			case <-ctx.Done():
				c.mutex.Lock()
				c.leader = false
				c.mutex.Unlock()
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
	case 0:
		log.Printf("[RAFT EVENT] LeaderUpdated: %#v", info)
		if info.LeaderID == info.ReplicaID {
			c.start()
		} else {
			c.stop()
		}
	}
}
