package zongzi

import (
	"encoding/json"
	"fmt"
	"sync"
)

const (
	cmd_type_host     = "host"
	cmd_type_replica  = "replica"
	cmd_type_shard    = "shard"
	cmd_type_snapshot = "snapshot"

	cmd_action_del        = "del"
	cmd_action_put        = "put"
	cmd_action_set_status = "set-status"

	query_action_get = "get"

	cmd_result_failure uint64 = 0
	cmd_result_success uint64 = 1
)

var (
	fsmErrHostNotFound  = fmt.Errorf(`Host not found`)
	fsmErrShardNotFound = fmt.Errorf(`Shard not found`)
)

type Snapshot struct {
	Hosts    []*Host
	Index    uint64
	Replicas []*Replica
	Shards   []*Shard
}

type Host struct {
	ID         string            `json:"id"`
	Meta       []byte            `json:"meta"`
	RaftAddr   string            `json:"raft_address"`
	Replicas   map[uint64]uint64 `json:"replicas"` // replicaID: shardID
	ShardTypes []string          `json:"shardTypes"`
	Status     HostStatus        `json:"status"`

	mu sync.RWMutex
}

type Shard struct {
	ID       uint64            `json:"id"`
	Replicas map[uint64]string `json:"replicas"` // replicaID: nodehostID
	Status   ShardStatus       `json:"status"`
	Type     string            `json:"type"`
	Version  string            `json:"version"`

	mu sync.RWMutex
}

type Replica struct {
	HostID      string        `json:"hostID"`
	ID          uint64        `json:"id"`
	IsNonVoting bool          `json:"isNonVoting"`
	IsWitness   bool          `json:"isWitness"`
	ShardID     uint64        `json:"shardID"`
	Status      ReplicaStatus `json:"status"`

	mu sync.RWMutex
}

type cmd struct {
	Action string `json:"action"`
	Type   string `json:"type"`
}

type cmdHost struct {
	cmd
	Host Host `json:"host"`
}

type cmdShard struct {
	cmd
	Shard Shard `json:"shard"`
}

type cmdReplica struct {
	cmd
	Replica Replica `json:"replica"`
}

func newCmdSetReplicaStatus(id uint64, status ReplicaStatus) (b []byte) {
	b, _ = json.Marshal(cmdReplica{cmd{
		Action: cmd_action_set_status,
		Type:   cmd_type_replica,
	}, Replica{
		ID:     id,
		Status: status,
	}})
	return
}

func newCmdHostPut(nhid, raftAddr string, meta []byte, status HostStatus, shardTypes []string) (b []byte) {
	b, _ = json.Marshal(cmdHost{cmd{
		Action: cmd_action_put,
		Type:   cmd_type_host,
	}, Host{
		ID:         nhid,
		Meta:       meta,
		RaftAddr:   raftAddr,
		ShardTypes: shardTypes,
		Status:     status,
	}})
	return
}

func newCmdHostDel(nhid string) (b []byte) {
	b, _ = json.Marshal(cmdHost{cmd{
		Action: cmd_action_del,
		Type:   cmd_type_host,
	}, Host{
		ID: nhid,
	}})
	return
}

func newCmdShardPut(shardID uint64, shardType string) (b []byte) {
	b, _ = json.Marshal(cmdShard{cmd{
		Action: cmd_action_put,
		Type:   cmd_type_shard,
	}, Shard{
		ID:     shardID,
		Status: ShardStatus_New,
		Type:   shardType,
	}})
	return
}

func newCmdShardDel(shardID uint64) (b []byte) {
	b, _ = json.Marshal(cmdShard{cmd{
		Action: cmd_action_del,
		Type:   cmd_type_shard,
	}, Shard{
		ID: shardID,
	}})
	return
}

func newCmdReplicaPut(nhid string, shardID, replicaID uint64, isNonVoting bool) (b []byte) {
	b, _ = json.Marshal(cmdReplica{cmd{
		Action: cmd_action_put,
		Type:   cmd_type_replica,
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
	b, _ = json.Marshal(cmdReplica{cmd{
		Action: cmd_action_del,
		Type:   cmd_type_replica,
	}, Replica{
		ID: replicaID,
	}})
	return
}

type query struct {
	Type   string `json:"type"`
	Action string `json:"action"`
}

type queryHost struct {
	query
	Host Host `json:"host"`
}

type querySnapshot struct {
	query
}

func newQueryHostGet(nhid string) queryHost {
	return queryHost{query{
		Type:   cmd_type_host,
		Action: query_action_get,
	}, Host{
		ID: nhid,
	}}
}

func newQuerySnapshotGet() querySnapshot {
	return querySnapshot{query{
		Type:   cmd_type_snapshot,
		Action: query_action_get,
	}}
}
