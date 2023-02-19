package zongzi

import (
	"encoding/json"

	"github.com/elliotchance/orderedmap/v2"
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

type Snapshot struct {
	Hosts    []*Host
	Index    uint64
	Replicas []*Replica
	Shards   []*Shard
}

type fsmState struct {
	hosts    *orderedmap.OrderedMap[string, *Host]
	replicas *orderedmap.OrderedMap[uint64, *Replica]
	shards   *orderedmap.OrderedMap[uint64, *Shard]
}

func (s fsmState) listHosts() (hosts []*Host) {
	for el := s.hosts.Front(); el != nil; el = el.Next() {
		hosts = append(hosts, el.Value)
	}
	return hosts
}

func (s fsmState) listShards() (shards []*Shard) {
	for el := s.shards.Front(); el != nil; el = el.Next() {
		shards = append(shards, el.Value)
	}
	return shards
}

func (s fsmState) listReplicas() (replicas []*Replica) {
	for el := s.replicas.Front(); el != nil; el = el.Next() {
		replicas = append(replicas, el.Value)
	}
	return replicas
}

type Host struct {
	ID         string            `json:"id"`
	Meta       []byte            `json:"meta"`
	RaftAddr   string            `json:"raft_address"`
	Replicas   map[uint64]uint64 `json:"replicas"` // replicaID: shardID
	ShardTypes []string          `json:"shardTypes"`
	Status     HostStatus        `json:"status"`
}

type Shard struct {
	ID       uint64            `json:"id"`
	Replicas map[uint64]string `json:"replicas"` // replicaID: nodehostID
	Status   ShardStatus       `json:"status"`
	Type     string            `json:"type"`
	Version  string            `json:"version"`
}

type Replica struct {
	HostID      string        `json:"hostID"`
	ID          uint64        `json:"id"`
	IsNonVoting bool          `json:"isNonVoting"`
	IsWitness   bool          `json:"isWitness"`
	ShardID     uint64        `json:"shardID"`
	Status      ReplicaStatus `json:"status"`
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

type cmd_SetReplicaStatus struct {
	cmd
	ID     uint64
	Status ReplicaStatus
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

type queryReplica struct {
	query
	Replica Replica `json:"replica"`
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

func newQueryReplicaGet(nhid string, shardID uint64) queryReplica {
	return queryReplica{query{
		Type:   cmd_type_replica,
		Action: query_action_get,
	}, Replica{
		HostID:  nhid,
		ShardID: shardID,
	}}
}

func newQuerySnapshotGet() querySnapshot {
	return querySnapshot{query{
		Type:   cmd_type_snapshot,
		Action: query_action_get,
	}}
}
