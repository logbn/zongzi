package zongzi

import (
	"iter"
	"strings"
)

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

func (s Shard) TagsPrefixed(prefix string) iter.Seq2[string, string] {
	return tagsPrefixed(s.Tags, prefix)
}

func (r Replica) TagsPrefixed(prefix string) iter.Seq2[string, string] {
	return tagsPrefixed(r.Tags, prefix)
}

func (h Host) TagsPrefixed(prefix string) iter.Seq2[string, string] {
	return tagsPrefixed(h.Tags, prefix)
}

func tagsPrefixed(tags map[string]string, prefix string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for k, v := range tags {
			if strings.HasPrefix(k, prefix) {
				yield(k, v)
			}
		}
	}
}
