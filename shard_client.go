package zongzi

import (
	"context"
)

// ShardClient can be used to interact with a shard regardless of its placement in the cluster
// Requests will be forwarded to the appropriate host based on ping
type ShardClient interface {
	Apply(ctx context.Context, cmd []byte) (value uint64, data []byte, err error)
	Commit(ctx context.Context, cmd []byte) (err error)
	Read(ctx context.Context, query []byte, stale ...bool) (value uint64, data []byte, err error)
	Watch(ctx context.Context, query []byte, results chan<- *Result, stale ...bool) (err error)
}

// The shardClient
type shardClient struct {
	manager *shardClientManager
	shardID uint64
	retries int
}

func newShardClient(manager *shardClientManager, shardID uint64, opts ...ShardClientOption) (c *shardClient, err error) {
	c = &shardClient{
		manager: manager,
		shardID: shardID,
	}
	for _, fn := range opts {
		if err = fn(c); err != nil {
			return
		}
	}
	return
}

func (c *shardClient) Apply(ctx context.Context, cmd []byte) (value uint64, data []byte, err error) {
	c.manager.mutex.RLock()
	el := c.manager.clientMember[c.shardID].Front()
	c.manager.mutex.RUnlock()
	for ; el != nil; el = el.Next() {
		value, data, err = el.Value.Apply(ctx, c.shardID, cmd)
		if err == nil {
			break
		}
	}
	return
}

func (c *shardClient) Commit(ctx context.Context, cmd []byte) (err error) {
	c.manager.mutex.RLock()
	el := c.manager.clientMember[c.shardID].Front()
	c.manager.mutex.RUnlock()
	for ; el != nil; el = el.Next() {
		err = el.Value.Commit(ctx, c.shardID, cmd)
		if err == nil {
			break
		}
	}
	return
}

func (c *shardClient) Read(ctx context.Context, query []byte, stale ...bool) (value uint64, data []byte, err error) {
	var run bool
	if len(stale) > 0 && stale[0] {
		c.manager.mutex.RLock()
		el := c.manager.clientReplica[c.shardID].Front()
		c.manager.mutex.RUnlock()
		for ; el != nil; el = el.Next() {
			run = true
			value, data, err = el.Value.Read(ctx, c.shardID, query)
			if err == nil {
				break
			}
		}
		if run && err == nil {
			return
		}
	}
	c.manager.mutex.RLock()
	el := c.manager.clientMember[c.shardID].Front()
	c.manager.mutex.RUnlock()
	for ; el != nil; el = el.Next() {
		value, data, err = el.Value.Read(ctx, c.shardID, query)
		if err == nil {
			break
		}
	}
	return
}

func (c *shardClient) Watch(ctx context.Context, query []byte, results chan<- *Result, stale ...bool) (err error) {
	var run bool
	if len(stale) > 0 && stale[0] {
		c.manager.mutex.RLock()
		el := c.manager.clientReplica[c.shardID].Front()
		c.manager.mutex.RUnlock()
		for ; el != nil; el = el.Next() {
			run = true
			err = el.Value.Watch(ctx, c.shardID, query, results, stale...)
			if err == nil {
				break
			}
		}
		if run && err == nil {
			return
		}
	}
	c.manager.mutex.RLock()
	el := c.manager.clientMember[c.shardID].Front()
	c.manager.mutex.RUnlock()
	for ; el != nil; el = el.Next() {
		err = el.Value.Watch(ctx, c.shardID, query, results, stale...)
		if err == nil {
			break
		}
	}
	return
}