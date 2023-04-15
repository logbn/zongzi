package zongzi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

// The shard shardController creates and destroys replicas based on a shard's tags.
type shardController struct {
	agent       *Agent
	cluster     cluster
	ctx         context.Context
	ctxCancel   context.CancelFunc
	clock       clock.Clock
	isLeader    bool
	index       uint64
	lastHostID  string
	leaderIndex uint64
	mutex       sync.RWMutex
	wg          sync.WaitGroup
}

func newShardController(agent *Agent) *shardController {
	return &shardController{
		clock:   clock.New(),
		agent:   agent,
		cluster: agent,
	}
}

type cluster interface {
	replicaCreate(hostID string, shardID uint64, isNonVoting bool) (id uint64, err error)
	replicaDelete(replicaID uint64) (err error)
}

func (c *shardController) Start() (err error) {
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
				c.agent.log.Infof("Controller stopped")
				return
			case <-t.C:
				c.tick()
			}
		}
	}()
	return
}

func (c *shardController) tick() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var err error
	var hadErr bool
	var index uint64
	var updated = true
	if c.isLeader {
		for updated {
			updated = false
			err = c.agent.Read(c.ctx, func(state *State) {
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
					updated, err = c.reconcile(state, shard)
					if err != nil {
						hadErr = true
						c.agent.log.Warningf("Error resolving shard %d %s %s", shard.ID, shard.Name, err.Error())
						c.agent.tagsSet(shard, fmt.Sprintf(`placement:error=%s`, err.Error()))
					} else if _, ok := shard.Tags[`placement:error`]; ok {
						c.agent.tagsRemove(shard, `placement:error`)
					}
					if updated {
						// We break the iterator on update in order to catch a fresh snapshot for the next shard.
						// This ensures that changes applied during reconciliation of this shard will be visible to
						// reconciliation of the next shard. Otherwise we would stack all the replicas for all the
						// shards requiring reconciliation onto the first few hosts due to outdated host replica
						// counts. Updated must only be set to true if shard.Updated has been changed. Otherwise
						// reconcile will loop infinitely.
						return false
					}
					return true
				})
			})
			if err != nil {
				hadErr = true
			}
		}
	}
	// TODO - Update shard client pool
	if !hadErr && index > c.index {
		c.agent.log.Debugf("%s Finished processing %d", c.agent.HostID(), index)
		c.index = index
	}
	return
}

func (c *shardController) reconcile(state *State, shard Shard) (updated bool, err error) {
	c.agent.log.Debugf("Reconciling Shard %d", shard.ID)
	var (
		desired       = map[string]int{}
		filters       = map[string][]string{}
		found         = map[string]int{}
		matches       = map[string][]Host{}
		occupiedHosts = map[string]bool{}
		undesired     = []uint64{}
		vary          = map[string]bool{}
		varyCount     = map[string]map[string]int{}
		varyMatch     = map[string]map[string]int{}
	)
	for tagKey, tagValue := range shard.Tags {
		if !strings.HasPrefix(tagKey, "placement:") {
			continue
		}
		if tagKey == `placement:vary` {
			// ex: placement:vary=geo:zone
			vary[tagValue] = true
		} else if tagKey == `placement:member` {
			// ex: placement:member=3;geo:region=us-central1
			parts := strings.Split(tagValue, ";")
			i, err := strconv.Atoi(parts[0])
			if err != nil {
				c.agent.log.Warningf(`Invalid tag placement:member %s %s`, tagKey, err.Error())
				continue
			}
			desired[`member`] = i
			filters[`member`] = parts[1:]
		} else if strings.HasPrefix(tagKey, `placement:replica:`) {
			// ex: placement:replica:usc1=3;geo:region=us-central1
			// ex: placement:replica:usw1=3;geo:region=us-west1
			group := tagKey[len(`placement:replica:`):]
			if len(group) == 0 {
				c.agent.log.Warningf(`Invalid tag placement:replica - "%s"`, tagKey)
				continue
			}
			if group == `member` {
				c.agent.log.Warningf(`Invalid tag placement:replica - group name "member" is reserved.`)
				continue
			}
			parts := strings.Split(tagValue, ";")
			i, err := strconv.Atoi(parts[0])
			if err != nil {
				c.agent.log.Warningf(`Invalid tag placement:replica %s %s`, tagKey, err.Error())
				continue
			}
			desired[group] = i
			filters[group] = parts[1:]
		}
	}
	var varies bool
	var groups []string
	state.ReplicaIterateByShardID(shard.ID, func(replica Replica) bool {
		host, _ := state.Host(replica.HostID)
		groups = groups[:0]
		for group := range desired {
			if c.matchTagFilter(host.Tags, filters[group]) {
				varies = true
				for tag := range vary {
					if varyCount[group] == nil {
						varyCount[group] = map[string]int{}
					}
					v, ok := host.Tags[tag]
					if !ok {
						varies = false
						break
					}
					varyCount[group][fmt.Sprintf(`%s=%s`, tag, v)]++
				}
				if varies {
					groups = append(groups, group)
					found[group]++
				}
			}
			if len(groups) > 1 {
				c.agent.log.Infof(`Replica matched multiple groups [%05d:%05d]: %v`, shard.ID, replica.ID, groups)
			}
		}
		if len(groups) == 0 {
			c.agent.log.Debugf(`[%05d:%05d] Undesired \n%#v\n%#v`, shard.ID, replica.ID, shard.Tags, host.Tags)
			undesired = append(undesired, replica.ID)
		} else {
			occupiedHosts[replica.HostID] = true
		}
		return true
	})
	var excessReplicaCount = map[string]int{}
	var missingReplicaCount = map[string]int{}
	groups = groups[:0]
	for group, n := range desired {
		if found[group] > n {
			excessReplicaCount[group] = found[group] - n
		}
		if found[group] < n {
			missingReplicaCount[group] = n - found[group]
		}
		if group == `member` {
			// Always process the member group first
			groups = append([]string{group}, groups...)
		} else {
			groups = append(groups, group)
		}
	}
	var requiresRebalance = map[string]bool{}
	for group, tags := range varyCount {
		var min int
		var max int
		for _, n := range tags {
			if min == 0 || n < min {
				min = n
			}
			if max == 0 || n > max {
				max = n
			}
			if min-max < -1 || min-max > 1 {
				requiresRebalance[group] = true
			}
		}
	}
	// Early exit
	if len(missingReplicaCount) == 0 &&
		len(excessReplicaCount) == 0 &&
		len(requiresRebalance) == 0 &&
		len(undesired) == 0 {
		return
	}
	// Delete undesired replicas
	for _, replicaID := range undesired {
		if err = c.cluster.replicaDelete(replicaID); err != nil {
			c.agent.log.Errorf(`Error deleting replica: %s`, err.Error())
			return
		}
		updated = true
		// Early exit just simplifies the logic
		return
	}
	// Find matching hosts for each group
	state.HostIterate(func(host Host) bool {
		if _, ok := occupiedHosts[host.ID]; ok {
			return true
		}
		for group, n := range desired {
			if found[group] == n {
				continue
			}
			if c.matchTagFilter(host.Tags, filters[group]) {
				matches[group] = append(matches[group], host)
				for tag := range vary {
					v, _ := host.Tags[tag]
					if _, ok := varyMatch[group]; !ok {
						varyMatch[group] = map[string]int{}
					}
					varyMatch[group][fmt.Sprintf(`%s=%s`, tag, v)]++
				}
			}
		}
		return true
	})
	// TODO - Rebalance (maybe belongs in its own controller)
	// for group := range requiresRebalance {}

	// TODO - Remove excess replicas (deciding which to remove while retaining balance)
	// for group, n := range excessReplicaCount {}

	// Add missing replicas
	for _, group := range groups {
		var n = missingReplicaCount[group]
		for i := 0; i < n; i++ {
			if len(matches[group]) == 0 {
				err = fmt.Errorf(`No more matching hosts`)
				break
			}
			// Find the vary tag values with the fewest replicas
			var varyTagValues = map[string]string{}
			var varyTagCounts = map[string]int{}
			for tag, replicaCount := range varyCount[group] {
				if varyMatch[group][tag] == 0 {
					// Don't even try this vary tag because it has no remaining hosts.
					continue
				}
				k, v, ok := c.parseTag(tag)
				if !ok {
					c.agent.log.Errorf(`Invalid tag %s`, tag)
					continue
				}
				if varyTagCounts[k] == 0 || varyTagCounts[k] < replicaCount {
					varyTagValues[k] = v
					varyTagCounts[k] = replicaCount
				}
			}
			if len(varyTagValues) < len(varyCount[group]) {
				err = fmt.Errorf(`Failed to find an available host matching vary criteria`)
				break
			}
			// TODO - Ensure a host is available with the full tag set rather than each individually to avoid pathological edge case w/ multiple vary tags.
			// Build vary tag set with fewest replicas
			var varyTags []string
			for tagKey, replicaCount := range varyTagCounts {
				if replicaCount > 0 && group == `member` {
					// The smallest vary tag value has a member already. If this group is the member group then
					// we give up because we never schedule members in a way that violates the vary policy.
					err = fmt.Errorf(`Unable to find host that satisfies member vary policy for shard`)
					break
				}
				varyTags = append(varyTags, fmt.Sprintf(`%s=%s`, tagKey, varyTagValues[tagKey]))
			}
			var success bool
			var replicaID uint64
			for j, host := range matches[group] {
				if _, ok := occupiedHosts[host.ID]; ok {
					// Host already occupied
					continue
				}
				if c.matchTagFilter(host.Tags, varyTags) {
					replicaID, err = c.cluster.replicaCreate(host.ID, shard.ID, group != `member`)
					if err != nil {
						return
					}
					updated = true
					for _, varyTag := range varyTags {
						tagKey, _, _ := c.parseTag(varyTag)
						varyMatch[group][tagKey]--
						varyCount[group][varyTag]++
					}
					c.agent.log.Infof(`[%05d:%05d] Created replica for %s`, shard.ID, replicaID, shard.Name)
					matches[group] = append(matches[group][:j], matches[group][j+1:]...)
					occupiedHosts[host.ID] = true
					success = true
					break
				}
			}
			if !success {
				err = fmt.Errorf(`Unable to find host with matching vary tag set %+v`, varyTags)
				break
			}
		}
		if err != nil {
			c.agent.log.Warningf("[%05d] (%s/%s) %s", shard.ID, shard.Name, group, err.Error())
			err = nil
		}
	}
	return
}

func (c *shardController) parseTag(tag string) (k, v string, ok bool) {
	i := strings.Index(tag, "=")
	if i < 1 {
		return
	}
	return tag[:i], tag[i+1:], true
}

func (c *shardController) matchTagFilter(src map[string]string, tags []string) bool {
	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}
		k, v, ok := c.parseTag(tag)
		if !ok {
			c.agent.log.Warningf(`Invalid tag filter: %s`, tag)
			continue
		}
		if _, ok = src[k]; !ok {
			// Tag key not present
			return false
		}
		if src[k] != v {
			// Tag value does not match
			return false
		}
	}
	return true
}

func (c *shardController) LeaderUpdated(info LeaderInfo) {
	c.agent.log.Infof("[%05d:%05d] LeaderUpdated: %05d", info.ShardID, info.ReplicaID, info.LeaderID)
	if info.ShardID == 0 {
		c.mutex.Lock()
		c.isLeader = info.LeaderID == info.ReplicaID
		c.mutex.Unlock()
		return
	}
}

func (c *shardController) Stop() {
	defer c.agent.log.Infof(`Stopped shardController`)
	if c.ctxCancel != nil {
		c.ctxCancel()
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.index = 0
}
