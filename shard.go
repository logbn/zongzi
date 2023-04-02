package zongzi

type Shard struct {
	ID      uint64      `json:"id"`
	Meta    []byte      `json:"meta"`
	Status  ShardStatus `json:"status"`
	Created uint64      `json:"created"`
	Updated uint64      `json:"updated"`

	Type    string `json:"type"`
	Version string `json:"version"`
}
