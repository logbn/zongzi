package zongzi

import (
	"encoding/json"
	"sync"

	"github.com/elliotchance/orderedmap/v2"
)

type State struct {
	Hosts        *orderedmap.OrderedMap[string, *Host]
	Index        uint64
	ReplicaIndex uint64
	Replicas     *orderedmap.OrderedMap[uint64, *Replica]
	ShardIndex   uint64
	Shards       *orderedmap.OrderedMap[uint64, *Shard]

	mutex sync.RWMutex
}

func newState() *State {
	return &State{
		Hosts:    orderedmap.NewOrderedMap[string, *Host](),
		Replicas: orderedmap.NewOrderedMap[uint64, *Replica](),
		Shards:   orderedmap.NewOrderedMap[uint64, *Shard](),
	}
}

func (s *State) GetSnapshot() *Snapshot {
	return &Snapshot{
		Hosts:        s.hostList(),
		Index:        s.Index,
		ReplicaIndex: s.ReplicaIndex,
		Replicas:     s.replicaList(),
		ShardIndex:   s.ShardIndex,
		Shards:       s.shardList(),
	}
}

func (s *State) MarshalJSON() ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return json.Marshal(s.GetSnapshot())
}

func (s *State) UnmarshalJSON(data []byte) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var snapshot Snapshot
	if err = json.Unmarshal(data, &snapshot); err != nil {
		return
	}
	s.Index = snapshot.Index
	s.ShardIndex = snapshot.ShardIndex
	s.ReplicaIndex = snapshot.ReplicaIndex
	for _, host := range snapshot.Hosts {
		s.Hosts.Set(host.ID, &host)
	}
	for _, shard := range snapshot.Shards {
		s.Shards.Set(shard.ID, &shard)
	}
	for _, replica := range snapshot.Replicas {
		replica.Host, _ = s.Hosts.Get(replica.HostID)
		replica.Host.Replicas = append(replica.Host.Replicas, &replica)
		replica.Shard, _ = s.Shards.Get(replica.ShardID)
		replica.Shard.Replicas = append(replica.Shard.Replicas, &replica)
		s.Replicas.Set(replica.ID, &replica)
	}
	return
}

func (s *State) hostList() []Host {
	var list = make([]Host, 0, s.Hosts.Len())
	for el := s.Hosts.Front(); el != nil; el = el.Next() {
		list = append(list, *el.Value)
	}
	return list
}

func (s *State) shardList() []Shard {
	var list = make([]Shard, 0, s.Shards.Len())
	for el := s.Shards.Front(); el != nil; el = el.Next() {
		list = append(list, *el.Value)
	}
	return list
}

func (s *State) replicaList() []Replica {
	var list = make([]Replica, 0, s.Replicas.Len())
	for el := s.Replicas.Front(); el != nil; el = el.Next() {
		list = append(list, *el.Value)
	}
	return list
}
