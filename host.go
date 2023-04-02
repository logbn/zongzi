package zongzi

type Host struct {
	ID      string     `json:"id"`
	Meta    []byte     `json:"meta"`
	Status  HostStatus `json:"status"`
	Created uint64     `json:"created"`
	Updated uint64     `json:"updated"`

	ApiAddress  string   `json:"apiAddress"`
	RaftAddress string   `json:"raftAddress"`
	ShardTypes  []string `json:"shardTypes"`

	Ping uint64 `json:"-"`
}
