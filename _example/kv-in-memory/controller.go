package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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
	agent       *zongzi.Agent
	cancel      context.CancelFunc
	clock       clock.Clock
	leader      bool
	index       uint64
	leaderIndex uint64
	mutex       sync.RWMutex
	shard       zongzi.Shard
	members     []*zongzi.ReplicaClient
	clients     []*zongzi.ReplicaClient
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
				log.Printf("ERROR: %v", err)
			}
		}
	}()
	return
}

func (c *controller) tick() (err error) {
	var done bool
	if c.shard.ID == 0 {
		c.agent.Read(func(state zongzi.State) {
			if c.shard.ID == 0 {
				state.ShardIterate(func(s zongzi.Shard) bool {
					if s.Type == stateMachineUri && s.Status != zongzi.ShardStatus_Closed {
						c.shard = s
						return false
					}
					return true
				})
				// Shard does not yet exist. Create it.
				if c.shard.ID == 0 {
					c.shard, err = c.agent.CreateShard(stateMachineUri, stateMachineVersion)
					if err != nil {
						return
					}
				}
			}
		}, true)
	}
	c.mutex.RLock()
	leader := c.leader
	c.mutex.RUnlock()
	if leader {
		c.agent.Read(func(state zongzi.State) {
			if state.Index() <= c.leaderIndex {
				done = true
			}
			if done {
				return
			}
			c.mutex.Lock()
			defer c.mutex.Unlock()
			// Add replicas to new hosts and remove replicas for missing hosts
			var zones = map[string]bool{}
			state.HostIterate(func(h zongzi.Host) bool {
				var meta map[string]any
				if err = json.Unmarshal(h.Meta, &meta); err != nil {
					err = fmt.Errorf("Bad meta: %w", err)
					return true
				}
				var hasReplica bool
				state.ReplicaIterateByHostID(h.ID, func(r zongzi.Replica) bool {
					if r.ShardID == c.shard.ID {
						if !r.IsNonVoting {
							zones[meta["zone"].(string)] = true
						}
						hasReplica = true
						return false
					}
					return true
				})
				if hasReplica {
					return true
				}
				// Not strictly correct.
				// Need to evaulate all hosts to ensure no members already exist in this zone.
				if _, ok := zones[meta["zone"].(string)]; !ok {
					zones[meta["zone"].(string)] = true
					if _, err = c.agent.CreateReplica(c.shard.ID, h.ID, false); err != nil {
						return true
					}
				} else {
					if _, err = c.agent.CreateReplica(c.shard.ID, h.ID, true); err != nil {
						return true
					}
				}
				return true
			})
			c.leaderIndex = state.Index()
			// Print snapshot
			buf := bytes.NewBufferString("")
			state.Save(buf)
			log.Print(buf.String())
		}, true)
	}
	if c.shard.ID > 0 {
		// Resolve replica clients
		c.agent.Read(func(state zongzi.State) {
			if state.Index() <= c.index {
				done = true
			}
			if done {
				return
			}
			c.mutex.Lock()
			defer c.mutex.Unlock()
			c.members = c.members[:0]
			c.clients = c.clients[:0]
			state.ReplicaIterateByShardID(c.shard.ID, func(r zongzi.Replica) bool {
				rc := c.agent.GetReplicaClient(r.ID)
				c.clients = append(c.clients, rc)
				if !r.IsNonVoting {
					c.members = append(c.members, rc)
				}
				return true
			})
			c.index = state.Index()
		}, true)
	}
	return
}

func (c *controller) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.leader {
		c.cancel()
	}
}

func (c *controller) LeaderUpdated(info zongzi.LeaderInfo) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	switch info.ShardID {
	case 0:
		log.Printf("[RAFT EVENT] LeaderUpdated: %#v", info)
		if info.LeaderID == info.ReplicaID {
			c.leader = true
		} else {
			c.leader = false
		}
	}
}

func (c *controller) getRandClient(member bool) (rc *zongzi.ReplicaClient) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if member {
		if len(c.members) > 0 {
			rc = c.members[rand.Intn(len(c.members))]
		}
	} else {
		if len(c.clients) > 0 {
			rc = c.clients[rand.Intn(len(c.clients))]
		}
	}
	return
}
