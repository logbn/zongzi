package zongzi

import (
	"encoding/json"
	"sync"
)

const (
	command_type_host     = "host"
	command_type_replica  = "replica"
	command_type_shard    = "shard"
	command_type_snapshot = "snapshot"

	command_action_del           = "del"
	command_action_put           = "put"
	command_action_post          = "post"
	command_action_status_update = "status-update"
)

type Snapshot struct {
	Hosts        []Host
	Index        uint64
	ReplicaIndex uint64
	Replicas     []Replica
	ShardIndex   uint64
	Shards       []Shard

	mu sync.RWMutex
}

type Host struct {
	ID      string `json:"id"`
	Created uint64 `json:"created"`
	Updated uint64 `json:"updated"`
	Meta    []byte `json:"meta"`

	ApiAddress string     `json:"apiAddress"`
	ShardTypes []string   `json:"shardTypes"`
	Status     HostStatus `json:"status"`

	Replicas []*Replica `json:"-"`
}

type Shard struct {
	ID      uint64 `json:"id"`
	Created uint64 `json:"created"`
	Updated uint64 `json:"updated"`
	Meta    []byte `json:"meta"`

	Status  ShardStatus `json:"status"`
	Type    string      `json:"type"`
	Version string      `json:"version"`

	Replicas []*Replica `json:"-"`
}

func (s *Shard) Members() (res map[uint64]string) {
	res = map[uint64]string{}
	for _, replica := range s.Replicas {
		if replica.IsNonVoting || replica.IsWitness {
			continue
		}
		res[replica.ID] = replica.HostID
	}
	return
}

type Replica struct {
	ID      uint64 `json:"id"`
	Created uint64 `json:"created"`
	Updated uint64 `json:"updated"`
	Meta    []byte `json:"meta"`

	HostID      string        `json:"hostID"`
	IsNonVoting bool          `json:"isNonVoting"`
	IsWitness   bool          `json:"isWitness"`
	ShardID     uint64        `json:"shardID"`
	Status      ReplicaStatus `json:"status"`

	Host  *Host  `json:"-"`
	Shard *Shard `json:"-"`
}

type command struct {
	Action string `json:"action"`
	Type   string `json:"type"`
}

type commandHost struct {
	command
	Host Host `json:"host"`
}

type commandShard struct {
	command
	Shard Shard `json:"shard"`
}

type commandReplica struct {
	command
	Replica Replica `json:"replica"`
}

func newCmdHostPut(nhid, apiAddr string, meta []byte, status HostStatus, shardTypes []string) (b []byte) {
	b, _ = json.Marshal(commandHost{command{
		Action: command_action_put,
		Type:   command_type_host,
	}, Host{
		ApiAddress: apiAddr,
		ID:         nhid,
		Meta:       meta,
		ShardTypes: shardTypes,
		Status:     status,
	}})
	return
}

func newCmdHostDel(nhid string) (b []byte) {
	b, _ = json.Marshal(commandHost{command{
		Action: command_action_del,
		Type:   command_type_host,
	}, Host{
		ID: nhid,
	}})
	return
}

func newCmdShardPost(shardType, shardVersion string) (b []byte) {
	b, _ = json.Marshal(commandShard{command{
		Action: command_action_post,
		Type:   command_type_shard,
	}, Shard{
		Status:  ShardStatus_New,
		Type:    shardType,
		Version: shardVersion,
	}})
	return
}

func newCmdShardPut(shardID uint64, shardType, shardVersion string) (b []byte) {
	b, _ = json.Marshal(commandShard{command{
		Action: command_action_put,
		Type:   command_type_shard,
	}, Shard{
		ID:      shardID,
		Status:  ShardStatus_New,
		Type:    shardType,
		Version: shardVersion,
	}})
	return
}

func newCmdShardDel(shardID uint64) (b []byte) {
	b, _ = json.Marshal(commandShard{command{
		Action: command_action_del,
		Type:   command_type_shard,
	}, Shard{
		ID: shardID,
	}})
	return
}

func newCmdReplicaPut(nhid string, shardID, replicaID uint64, isNonVoting bool) (b []byte) {
	b, _ = json.Marshal(commandReplica{command{
		Action: command_action_put,
		Type:   command_type_replica,
	}, Replica{
		HostID:      nhid,
		ID:          replicaID,
		IsNonVoting: isNonVoting,
		ShardID:     shardID,
		Status:      ReplicaStatus_New,
	}})
	return
}

func newCmdReplicaDel(replicaID uint64) (b []byte) {
	b, _ = json.Marshal(commandReplica{command{
		Action: command_action_del,
		Type:   command_type_replica,
	}, Replica{
		ID: replicaID,
	}})
	return
}

func newCmdReplicaUpdateStatus(replicaID uint64, status ReplicaStatus) (b []byte) {
	b, _ = json.Marshal(commandReplica{command{
		Action: command_action_status_update,
		Type:   command_type_replica,
	}, Replica{
		ID:     replicaID,
		Status: status,
	}})
	return
}
