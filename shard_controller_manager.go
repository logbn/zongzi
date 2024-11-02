package zongzi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

// The controllerManager creates and destroys replicas based on a shard tags.
type controllerManager struct {
	agent       *Agent
	clock       clock.Clock
	ctx         context.Context
	ctxCancel   context.CancelFunc
	index       uint64
	isLeader    bool
	lastHostID  string
	leaderIndex uint64
	log         Logger
	mutex       sync.RWMutex
	controller  Controller
	wg          sync.WaitGroup
}

func newControllerManager(agent *Agent) *controllerManager {
	return &controllerManager{
		log:        agent.log,
		agent:      agent,
		clock:      clock.New(),
		controller: newShardControllerDefault(agent),
	}
}

type Controller interface {
	Reconcile(*State, Shard, Controls) error
}

func (c *controllerManager) Start() (err error) {
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

func (c *controllerManager) tick() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var err error
	var hadErr bool
	var index uint64
	var updated = true
	var controls = newControls(c.agent)
	if c.isLeader {
		for updated {
			updated = false
			err = c.agent.State(c.ctx, func(state *State) {
				index = state.Index()
				state.ShardIterateUpdatedAfter(c.index, func(shard Shard) bool {
					select {
					case <-c.ctx.Done():
						return false
					default:
					}
					if shard.ID == 0 {
						return true
					}
					controls.updated = false
					err = c.controller.Reconcile(state, shard, controls)
					if err != nil {
						hadErr = true
						c.log.Warningf("Error resolving shard %d %s %s", shard.ID, shard.Name, err.Error())
						c.agent.tagsSet(shard, fmt.Sprintf(`zongzi:controller:error=%s`, err.Error()))
					} else if _, ok := shard.Tags[`zongzi:controller:error`]; ok {
						c.agent.tagsRemove(shard, `zongzi:controller:error`)
					}
					if controls.updated {
						// We break the iterator on update in order to catch a fresh snapshot for the next shard.
						// This ensures that changes applied during reconciliation of this shard will be visible to
						// reconciliation of the next shard.
						return false
					}
					return true
				})
				updated = controls.updated
			})
			if err != nil {
				hadErr = true
			}
		}
	}
	if !hadErr && index > c.index {
		c.log.Debugf("%s Finished processing %d", c.agent.hostID(), index)
		// c.agent.dumpState()
		c.index = index
	}
	return
}

func (c *controllerManager) LeaderUpdated(info LeaderInfo) {
	c.log.Infof("[%05d:%05d] LeaderUpdated: %05d", info.ShardID, info.ReplicaID, info.LeaderID)
	if info.ShardID == 0 {
		c.mutex.Lock()
		c.isLeader = info.LeaderID == info.ReplicaID
		c.mutex.Unlock()
		return
	}
}

func (c *controllerManager) Stop() {
	defer c.log.Infof(`Stopped controllerManager`)
	if c.ctxCancel != nil {
		c.ctxCancel()
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.index = 0
}
