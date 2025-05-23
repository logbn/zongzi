package zongzi

type Host struct {
	ID      string            `json:"id"`
	Created uint64            `json:"created"`
	Updated uint64            `json:"updated"`
	Status  HostStatus        `json:"status"`
	Tags    map[string]string `json:"tags"`

	ApiAddress  string   `json:"apiAddress"`
	RaftAddress string   `json:"raftAddress"`
	ShardTypes  []string `json:"shardTypes"`
}

type Shard struct {
	ID      uint64            `json:"id"`
	Created uint64            `json:"created"`
	Updated uint64            `json:"updated"`
	Status  ShardStatus       `json:"status"`
	Tags    map[string]string `json:"tags"`

	Leader uint64 `json:"leader"`
	Name   string `json:"name"`
	Term   uint64 `json:"term"`
	Type   string `json:"type"`
}

type Replica struct {
	ID      uint64            `json:"id"`
	Created uint64            `json:"created"`
	Updated uint64            `json:"updated"`
	Status  ReplicaStatus     `json:"status"`
	Tags    map[string]string `json:"tags"`

	HostID      string `json:"hostID"`
	IsNonVoting bool   `json:"isNonVoting"`
	IsWitness   bool   `json:"isWitness"`
	ShardID     uint64 `json:"shardID"`
}
