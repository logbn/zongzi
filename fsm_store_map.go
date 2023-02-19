package zongzi

import (
	"github.com/elliotchance/orderedmap/v2"
)

func newFsmStoreMap() *fsmStoreMap {
	return &fsmStoreMap{
		hosts:    orderedmap.NewOrderedMap[string, *Host](),
		shards:   orderedmap.NewOrderedMap[uint64, *Shard](),
		replicas: orderedmap.NewOrderedMap[uint64, *Replica](),
	}
}

type fsmStoreMap struct {
	hosts    *orderedmap.OrderedMap[string, *Host]
	replicas *orderedmap.OrderedMap[uint64, *Replica]
	shards   *orderedmap.OrderedMap[uint64, *Shard]
}

func (s *fsmStoreMap) HostDelete(id string) (n uint64) {
	if host, ok := s.hosts.Get(id); ok {
		for id := range host.Replicas {
			s.ReplicaDelete(id)
			n++
		}
	}
	s.hosts.Delete(id)
	return
}

func (s *fsmStoreMap) HostFind(id string) (v *Host) {
	v, _ = s.hosts.Get(id)
	return
}

func (s *fsmStoreMap) HostList() (hosts []*Host) {
	for el := s.hosts.Front(); el != nil; el = el.Next() {
		hosts = append(hosts, el.Value)
	}
	return hosts
}

func (s *fsmStoreMap) HostPut(new *Host) {
	if old, ok := s.hosts.Get(new.ID); ok {
		new.Replicas = old.Replicas
	}
	s.hosts.Set(new.ID, new)
}

func (s *fsmStoreMap) ShardDelete(id uint64) (n uint64) {
	if old, ok := s.shards.Get(id); ok {
		for id := range old.Replicas {
			s.ReplicaDelete(id)
			n++
		}
	}
	s.shards.Delete(id)
	return
}

func (s *fsmStoreMap) ShardFind(id uint64) (v *Shard) {
	v, _ = s.shards.Get(id)
	return
}

func (s *fsmStoreMap) ShardList() (shards []*Shard) {
	for el := s.shards.Front(); el != nil; el = el.Next() {
		shards = append(shards, el.Value)
	}
	return shards
}

func (s *fsmStoreMap) ShardPut(new *Shard) {
	if old, ok := s.shards.Get(new.ID); ok {
		new.Replicas = old.Replicas
	}
	s.shards.Set(new.ID, new)
}

func (s *fsmStoreMap) ReplicaDelete(id uint64) {
	if old, ok := s.replicas.Get(id); ok {
		if host, ok := s.hosts.Get(old.HostID); ok {
			delete(host.Replicas, id)
		}
		if shard, ok := s.shards.Get(old.ShardID); ok {
			delete(shard.Replicas, id)
		}
	}
	s.replicas.Delete(id)
	return
}

func (s *fsmStoreMap) ReplicaFind(id uint64) (v *Replica) {
	v, _ = s.replicas.Get(id)
	return
}

func (s *fsmStoreMap) ReplicaList() (replicas []*Replica) {
	for el := s.replicas.Front(); el != nil; el = el.Next() {
		replicas = append(replicas, el.Value)
	}
	return replicas
}

func (s *fsmStoreMap) ReplicaPut(new *Replica) error {
	host, ok := s.hosts.Get(new.HostID)
	if !ok {
		return fsmErrHostNotFound
	}
	shard, ok := s.shards.Get(new.ShardID)
	if !ok {
		return fsmErrShardNotFound
	}
	host.Replicas[new.ID] = new.ShardID
	shard.Replicas[new.ID] = new.HostID
	s.replicas.Set(new.ID, new)
	return nil
}
