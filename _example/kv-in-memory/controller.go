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
	ctx         context.Context
	ctxCancel   context.CancelFunc
	clock       clock.Clock
	isLeader    bool
	index       uint64
	leaderIndex uint64
	shard       zongzi.Shard
	members     []*zongzi.Client
	clients     []*zongzi.Client
	mutex       sync.RWMutex
	wg          sync.WaitGroup
}

func (c *controller) Start() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.ctx, c.ctxCancel = context.WithCancel(context.Background())
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		t := c.clock.Ticker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-c.ctx.Done():
				log.Println("example: Controller stopped")
				return
			case <-t.C:
				err = c.tick()
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
		c.agent.Read(c.ctx, func(state zongzi.State) {
			state.ShardIterate(func(s zongzi.Shard) bool {
				if s.Type == stateMachineUri && s.Status != zongzi.ShardStatus_Closed {
					c.shard = s
					return false
				}
				return true
			})
			// Shard does not yet exist. Create it.
			if c.shard.ID == 0 {
				c.shard, err = c.agent.ShardCreate(stateMachineUri)
				if err != nil {
					return
				}
				c.isLeader = true
			}
		})
	}
	if c.isLeader {
		c.agent.Read(c.ctx, func(state zongzi.State) {
			if c.shard, ok = state.Shard(c.shard.ID); !ok {
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
					_, err = c.agent.ReplicaCreate(item.Host.ID, c.shard.ID, false)
				} else {
					_, err = c.agent.ReplicaCreate(item.Host.ID, c.shard.ID, true)
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
		})
	}
	if c.shard.ID > 0 {
		// Resolve replica clients
		c.agent.Read(c.ctx, func(state zongzi.State) {
			if c.shard, ok = state.Shard(c.shard.ID); !ok {
				return
			}
			if c.shard.Updated <= c.index {
				done = true
			}
			if done {
				return
			}
			var members []*zongzi.Client
			var clients []*zongzi.Client
			state.ReplicaIterateByShardID(c.shard.ID, func(r zongzi.Replica) bool {
				rc := c.agent.Client(r.HostID)
				clients = append(clients, rc)
				if !r.IsNonVoting {
					members = append(members, rc)
				}
				return true
			})
			c.members = members
			c.clients = clients
			c.index = c.shard.Updated
		})
	}
	return
}

func (c *controller) Stop() {
	c.ctxCancel()
	c.wg.Wait()
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

func (c *controller) getClient(random, member bool) (rc *zongzi.Client) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if !random {
		rc = c.agent.Client(c.agent.HostID())
	} else if member {
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
