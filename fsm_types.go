package zongzi

import (
	"encoding/json"
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

type Host struct {
	ID      string     `json:"id"`
	Meta    []byte     `json:"meta"`
	Status  HostStatus `json:"status"`
	Created uint64     `json:"created"`
	Updated uint64     `json:"updated"`

	ApiAddress string   `json:"apiAddress"`
	ShardTypes []string `json:"shardTypes"`
}

type Shard struct {
	ID      uint64      `json:"id"`
	Meta    []byte      `json:"meta"`
	Status  ShardStatus `json:"status"`
	Created uint64      `json:"created"`
	Updated uint64      `json:"updated"`

	Type    string `json:"type"`
	Version string `json:"version"`
}

type Replica struct {
	ID      uint64        `json:"id"`
	Meta    []byte        `json:"meta"`
	Status  ReplicaStatus `json:"status"`
	Created uint64        `json:"created"`
	Updated uint64        `json:"updated"`

	HostID      string `json:"hostID"`
	IsNonVoting bool   `json:"isNonVoting"`
	IsWitness   bool   `json:"isWitness"`
	ShardID     uint64 `json:"shardID"`
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

func newCmdReplicaPost(nhid string, shardID uint64, isNonVoting bool) (b []byte) {
	r := Replica{
		HostID:      nhid,
		IsNonVoting: isNonVoting,
		ShardID:     shardID,
		Status:      ReplicaStatus_New,
	}
	if isNonVoting {
		r.Status = ReplicaStatus_Joining
	}
	b, _ = json.Marshal(commandReplica{command{
		Action: command_action_post,
		Type:   command_type_replica,
	}, r})
	return
}

func newCmdReplicaPut(nhid string, shardID, replicaID uint64, isNonVoting bool) (b []byte) {
	b, _ = json.Marshal(commandReplica{command{
		Action: command_action_put,
		Type:   command_type_replica,
	}, Replica{
		HostID:      nhid,
		ID:          replicaID,
		ShardID:     shardID,
		IsNonVoting: isNonVoting,
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
