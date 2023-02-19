package zongzi

type fsmStore interface {
	// HostDelete deletes a host by id and all associated replicas, returning number of replicas deleted
	HostDelete(id string) uint64

	// HostFind returns a host by id
	HostFind(id string) *Host

	// HostList retrieves a list of hosts
	HostList() (hosts []*Host)

	// HostPut upserts a host, preserving replicas
	HostPut(h *Host)

	// ReplicaDelete deletes a replica by id
	ReplicaDelete(id uint64)

	// ReplicaFind returns a replica by id
	ReplicaFind(id uint64) *Replica

	// ReplicaList retrieves a list of all replicas
	ReplicaList() []*Replica

	// ReplicaPut upserts a replica
	ReplicaPut(r *Replica) error

	// ShardDelete deletes a shard by id and all associated replicas, returning number of replicas deleted
	ShardDelete(id uint64) uint64

	// ShardFind returns a shard by id
	ShardFind(id uint64) *Shard

	// ShardList retrieves a list of all shards
	ShardList() (shards []*Shard)

	// ShardPut upserts a shard, preserving replicas
	ShardPut(s *Shard)
}
