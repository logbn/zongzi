package main

import (
	"bytes"
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/logbn/zongzi"
)

type controller struct {
	agent      *zongzi.Agent
	ctx        context.Context
	ctxCancel  context.CancelFunc
	members    []*zongzi.Client
	clients    []*zongzi.Client
	mutex      sync.RWMutex
	wg         sync.WaitGroup
	shard      zongzi.Shard
	index      uint64
	lastHostID string
}

func newController() *controller {
	return &controller{}
}

func (c *controller) Start(agent *zongzi.Agent) (err error) {
	c.mutex.Lock()
	c.agent = agent
	c.ctx, c.ctxCancel = context.WithCancel(context.Background())
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				err = c.tick()
			case <-c.ctx.Done():
				return
			}
			if err != nil {
				log.Printf("controller: %v", err)
			}
		}
	}()
	c.mutex.Unlock()
	return c.tick()
}

func (c *controller) tick() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var ok bool
	if c.shard.ID == 0 {
		c.agent.Read(c.ctx, func(state *zongzi.State) {
			state.ShardIterate(func(s zongzi.Shard) bool {
				if s.Type == uri {
					c.shard = s
					return false
				}
				return true
			})
		})
	}
	if c.shard.ID > 0 {
		// Resolve replica clients
		c.agent.Read(c.ctx, func(state *zongzi.State) {
			if c.shard, ok = state.Shard(c.shard.ID); !ok {
				return
			}
			if c.shard.Updated <= c.index && c.lastHostID == state.HostID() {
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
			c.lastHostID = state.HostID()

			// Print snapshot
			buf := bytes.NewBufferString("")
			state.Save(buf)
			log.Print(buf.String())
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
