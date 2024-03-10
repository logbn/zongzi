package zongzi

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/elliotchance/orderedmap/v2"
)

// The shardClientManager creates and destroys replicas based on a shard tags.
type shardClientManager struct {
	agent           *Agent
	clock           clock.Clock
	ctx             context.Context
	ctxCancel       context.CancelFunc
	clientHost      map[string]HostClient
	clientMember    map[uint64]*orderedmap.OrderedMap[int64, HostClient]
	clientReplica   map[uint64]*orderedmap.OrderedMap[int64, HostClient]
	index           uint64
	log             Logger
	mutex           sync.RWMutex
	shardController ShardController
	wg              sync.WaitGroup
}

func newShardClientManager(agent *Agent) *shardClientManager {
	return &shardClientManager{
		log:           agent.log,
		agent:         agent,
		clock:         clock.New(),
		clientHost:    map[string]HostClient{},
		clientMember:  map[uint64]*orderedmap.OrderedMap[int64, HostClient]{},
		clientReplica: map[uint64]*orderedmap.OrderedMap[int64, HostClient]{},
	}
}

func (c *shardClientManager) Start() (err error) {
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
				c.log.Infof("Shard controller manager stopped")
				return
			case <-t.C:
				c.tick()
			}
		}
	}()
	return
}

type hostClientPing struct {
	ping   int64
	client HostClient
}

func (c *shardClientManager) tick() {
	var err error
	var index uint64
	var start = c.clock.Now()
	var shardCount int
	var replicaCount int
	var pings = map[string]time.Duration{}
	err = c.agent.Read(c.ctx, func(state *State) {
		state.ShardIterateUpdatedAfter(c.index, func(shard Shard) bool {
			shardCount++
			index = shard.Updated
			members := []hostClientPing{}
			replicas := []hostClientPing{}
			state.ReplicaIterateByShardID(shard.ID, func(replica Replica) bool {
				replicaCount++
				client, ok := c.clientHost[replica.HostID]
				if !ok {
					client = c.agent.HostClient(replica.HostID)
					c.clientHost[replica.HostID] = client
				}
				ping, ok := pings[replica.HostID]
				if !ok {
					ctx, cancel := context.WithTimeout(c.ctx, time.Second)
					defer cancel()
					ping, err = client.Ping(ctx)
					if err != nil {
						c.log.Warningf(`Unable to ping host in shard client manager: %s`, err.Error())
						return true
					}
					pings[replica.HostID] = ping
				}
				if replica.IsNonVoting {
					replicas = append(replicas, hostClientPing{ping.Nanoseconds(), client})
				} else {
					members = append(members, hostClientPing{ping.Nanoseconds(), client})
				}
				return true
			})
			slices.SortFunc(members, func(a, b hostClientPing) int { return cmp.Compare(a.ping, b.ping) })
			slices.SortFunc(replicas, func(a, b hostClientPing) int { return cmp.Compare(a.ping, b.ping) })
			newMembers := orderedmap.NewOrderedMap[int64, HostClient]()
			for _, item := range members {
				newMembers.Set(item.ping, item.client)
			}
			newReplicas := orderedmap.NewOrderedMap[int64, HostClient]()
			for _, item := range replicas {
				newReplicas.Set(item.ping, item.client)
			}
			c.mutex.Lock()
			c.clientMember[shard.ID] = newMembers
			c.clientReplica[shard.ID] = newReplicas
			c.mutex.Unlock()
			return true
		})
	}, true)
	if err == nil && shardCount > 0 {
		c.log.Infof("%s Shard client manager updated. hosts: %d shards: %d replicas: %d time: %vms", c.agent.HostID(), len(pings), shardCount, replicaCount, int(c.clock.Since(start)/time.Millisecond))
		c.index = index
	}
	return
}

func (c *shardClientManager) Stop() {
	defer c.log.Infof(`Stopped shardClientManager`)
	if c.ctxCancel != nil {
		c.ctxCancel()
	}
	c.wg.Wait()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.index = 0
}
