package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	ctx         context.Context
	clock       clock.Clock
	isLeader    bool
	index       uint64
	leaderIndex uint64
	shard       zongzi.Shard
	members     []*zongzi.ReplicaClient
	clients     []*zongzi.ReplicaClient
	mutex       sync.RWMutex
	wg          sync.WaitGroup
}

func (c *controller) Start() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		t := c.clock.Ticker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				err = c.tick()
			case <-c.ctx.Done():
				return
			}
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
		}
	}()
	return
}

type hostMeta struct {
	Host zongzi.Host
	Meta struct {
		Zone string `json:"zone"`
	}
}

func (c *controller) tick() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var ok bool
	var done bool
	if c.shard.ID == 0 {
		c.agent.Read(func(state zongzi.State) {
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
				c.isLeader = true
			}
		}, true)
	}
	if c.isLeader {
		c.agent.Read(func(state zongzi.State) {
			if c.shard, ok = state.ShardGet(c.shard.ID); !ok {
				return
			}
			if c.shard.Updated <= c.leaderIndex {
				done = true
			}
			if done {
				return
			}
			// Ensure that every host has a replica of the kv store.
			// One voting replica per zone. The rest nonVoting (read replicas).
			var toAdd []hostMeta
			var zones = map[string]bool{}
			state.HostIterate(func(h zongzi.Host) bool {
				var item = hostMeta{Host: h}
				json.Unmarshal(h.Meta, &item.Meta)
				if len(item.Meta.Zone) == 0 {
					log.Println("Invalid host meta %s", string(h.Meta))
					return true
				}
				var hasReplica bool
				state.ReplicaIterateByHostID(h.ID, func(r zongzi.Replica) bool {
					if r.ShardID == c.shard.ID {
						if !r.IsNonVoting {
							zones[item.Meta.Zone] = true
						}
						hasReplica = true
						return false
					}
					return true
				})
				if !hasReplica {
					toAdd = append(toAdd, item)
				}
				return true
			})
			for _, item := range toAdd {
				if _, ok := zones[item.Meta.Zone]; !ok {
					_, err = c.agent.CreateReplica(c.shard.ID, item.Host.ID, false)
				} else {
					_, err = c.agent.CreateReplica(c.shard.ID, item.Host.ID, true)
				}
				if err != nil {
					log.Println(err)
				}
			}
			c.leaderIndex = c.shard.Updated
			// Print snapshot
			buf := bytes.NewBufferString("")
			state.Save(buf)
			log.Print(buf.String())
		}, true)
	}
	if c.shard.ID > 0 {
		// Resolve replica clients
		c.agent.Read(func(state zongzi.State) {
			if c.shard, ok = state.ShardGet(c.shard.ID); !ok {
				return
			}
			if c.shard.Updated <= c.index {
				done = true
			}
			if done {
				return
			}
			var members []*zongzi.ReplicaClient
			var clients []*zongzi.ReplicaClient
			state.ReplicaIterateByShardID(c.shard.ID, func(r zongzi.Replica) bool {
				rc := c.agent.GetReplicaClient(r.ID)
				clients = append(clients, rc)
				if !r.IsNonVoting {
					members = append(members, rc)
				}
				return true
			})
			c.members = members
			c.clients = clients
			c.index = c.shard.Updated
		}, true)
	}
	return
}

func (c *controller) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.isLeader {
		c.cancel()
		c.wg.Wait()
	}
}

func (c *controller) LeaderUpdated(info zongzi.LeaderInfo) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.shard.ID == 0 {
		return
	}
	log.Printf("[%05d:%05d] LeaderUpdated: %05d", info.ShardID, info.ReplicaID, info.LeaderID)
	switch info.ShardID {
	case c.shard.ID:
		c.isLeader = info.LeaderID == info.ReplicaID
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
