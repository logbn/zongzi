package main

import (
	"bytes"
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
		ctx   context.Context
		err   error
		index uint64
		shard zongzi.Shard
	)
	ctx, c.cancel = context.WithCancel(context.Background())
	go func() {
		t := c.clock.Ticker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				prevIndex := index
				c.agent.Read(func(state zongzi.State) {
					if state.Index() <= index {
						return
					}
					if shard.ID == 0 {
						state.ShardIterate(func(s zongzi.Shard) bool {
							if s.Type == StateMachineUri && s.Status != zongzi.ShardStatus_Closed {
								shard = s
								return false
							}
							return true
						})
						// Shard does not yet exist. Create it.
						if shard.ID == 0 {
							shard, err = c.agent.CreateShard(StateMachineUri, StateMachineVersion)
							if err != nil {
								return
							}
						}
					}
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
							if r.ShardID == shard.ID {
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
							if _, err = c.agent.CreateReplica(shard.ID, h.ID, false); err != nil {
								return true
							}
						} else {
							if _, err = c.agent.CreateReplica(shard.ID, h.ID, true); err != nil {
								return true
							}
						}
						return true
					})
					index = state.Index()
				})
				if index != prevIndex {
					c.agent.Read(func(state zongzi.State) {
						buf := bytes.NewBufferString("")
						state.Save(buf)
						log.Print(buf.String())
					})
				}
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
