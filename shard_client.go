package zongzi

import (
	"context"
)

// ShardClient can be used to interact with a shard regardless of its placement in the cluster
// Requests will be forwarded to the appropriate host based on ping
type ShardClient interface {
	Index(ctx context.Context) (err error)
	Apply(ctx context.Context, cmd []byte) (value uint64, data []byte, err error)
	Commit(ctx context.Context, cmd []byte) (err error)
	Read(ctx context.Context, query []byte, stale bool) (value uint64, data []byte, err error)
	Watch(ctx context.Context, query []byte, results chan<- *Result, stale bool) (err error)
}

// The shard client
type client struct {
	manager       *clientManager
	shardID       uint64
	retries       int
	writeToLeader bool
}

func newClient(manager *clientManager, shardID uint64, opts ...ClientOption) (c *client, err error) {
	c = &client{
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

func (c *client) Index(ctx context.Context) (err error) {
	c.manager.mutex.RLock()
	list, ok := c.manager.clientMember[c.shardID]
	c.manager.mutex.RUnlock()
	if !ok {
		err = ErrShardNotReady
		return
	}
	el := list.Front()
	for ; el != nil; el = el.Next() {
		err = el.Value.ReadIndex(ctx, c.shardID)
		if err == nil {
			break
		}
	}
	return
}

func (c *client) Apply(ctx context.Context, cmd []byte) (value uint64, data []byte, err error) {
	if c.writeToLeader {
		if client, ok := c.manager.clientLeader[c.shardID]; ok {
			return client.Apply(ctx, c.shardID, cmd)
		}
	}
	c.manager.mutex.RLock()
	list, ok := c.manager.clientMember[c.shardID]
	c.manager.mutex.RUnlock()
	if !ok {
		err = ErrShardNotReady
		return
	}
	for el := list.Front(); el != nil; el = el.Next() {
		value, data, err = el.Value.Apply(ctx, c.shardID, cmd)
		if err == nil {
			break
		}
	}
	return
}

func (c *client) Commit(ctx context.Context, cmd []byte) (err error) {
	if c.writeToLeader {
		if client, ok := c.manager.clientLeader[c.shardID]; ok {
			return client.Commit(ctx, c.shardID, cmd)
		}
	}
	c.manager.mutex.RLock()
	list, ok := c.manager.clientMember[c.shardID]
	c.manager.mutex.RUnlock()
	if !ok {
		err = ErrShardNotReady
		return
	}
	el := list.Front()
	for ; el != nil; el = el.Next() {
		err = el.Value.Commit(ctx, c.shardID, cmd)
		if err == nil {
			break
		}
	}
	return
}

func (c *client) Read(ctx context.Context, query []byte, stale bool) (value uint64, data []byte, err error) {
	var run bool
	if stale {
		c.manager.mutex.RLock()
		list, ok := c.manager.clientMember[c.shardID]
		c.manager.mutex.RUnlock()
		if !ok {
			err = ErrShardNotReady
			return
		}
		el := list.Front()
		for ; el != nil; el = el.Next() {
			run = true
			value, data, err = el.Value.Read(ctx, c.shardID, query, stale)
			if err == nil {
				break
			}
		}
		if run && err == nil {
			return
		}
	}
	c.manager.mutex.RLock()
	list, ok := c.manager.clientMember[c.shardID]
	c.manager.mutex.RUnlock()
	if !ok {
		err = ErrShardNotReady
		return
	}
	el := list.Front()
	for ; el != nil; el = el.Next() {
		value, data, err = el.Value.Read(ctx, c.shardID, query, stale)
		if err == nil {
			break
		}
	}
	return
}

func (c *client) Watch(ctx context.Context, query []byte, results chan<- *Result, stale bool) (err error) {
	var run bool
	if stale {
		c.manager.mutex.RLock()
		list, ok := c.manager.clientMember[c.shardID]
		c.manager.mutex.RUnlock()
		if !ok {
			err = ErrShardNotReady
			return
		}
		el := list.Front()
		for ; el != nil; el = el.Next() {
			run = true
			err = el.Value.Watch(ctx, c.shardID, query, results, stale)
			if err == nil {
				break
			}
		}
		if run && err == nil {
			return
		}
	}
	c.manager.mutex.RLock()
	list, ok := c.manager.clientMember[c.shardID]
	c.manager.mutex.RUnlock()
	if !ok {
		err = ErrShardNotReady
		return
	}
	el := list.Front()
	for ; el != nil; el = el.Next() {
		err = el.Value.Watch(ctx, c.shardID, query, results, stale)
		if err == nil {
			break
		}
	}
	return
}
