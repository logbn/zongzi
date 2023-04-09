package zongzi

import (
	"encoding/json"
	"strings"
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
	command_action_tags_set      = "tags-set"
	command_action_tags_setnx    = "tags-setnx"
	command_action_tags_remove   = "tags-remove"
)

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

func newCmdHostPut(nhid, apiAddr, raftAddr string, tagList []string, status HostStatus, shardTypes []string) (b []byte) {
	b, _ = json.Marshal(commandHost{command{
		Action: command_action_put,
		Type:   command_type_host,
	}, Host{
		ApiAddress:  apiAddr,
		ID:          nhid,
		Tags:        tagMapFromList(tagList),
		RaftAddress: raftAddr,
		ShardTypes:  shardTypes,
		Status:      status,
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

func newCmdShardPost(shardType string, tags map[string]string) (b []byte) {
	b, _ = json.Marshal(commandShard{command{
		Action: command_action_post,
		Type:   command_type_shard,
	}, Shard{
		Status: ShardStatus_New,
		Type:   shardType,
		Tags:   tags,
	}})
	return
}

func newCmdShardPut(shardID uint64, shardType string) (b []byte) {
	b, _ = json.Marshal(commandShard{command{
		Action: command_action_put,
		Type:   command_type_shard,
	}, Shard{
		ID:     shardID,
		Status: ShardStatus_New,
		Type:   shardType,
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
	b, _ = json.Marshal(commandReplica{command{
		Action: command_action_post,
		Type:   command_type_replica,
	}, Replica{
		HostID:      nhid,
		IsNonVoting: isNonVoting,
		ShardID:     shardID,
		Status:      ReplicaStatus_New,
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

func newCmdTagsSetNX(subject any, tagList ...string) []byte {
	return newCmdTags(command_action_tags_setnx, subject, tagList)
}

func newCmdTagsSet(subject any, tagList ...string) []byte {
	return newCmdTags(command_action_tags_set, subject, tagList)
}

func newCmdTagsRemove(subject any, tagList ...string) []byte {
	return newCmdTags(command_action_tags_remove, subject, tagList)
}

func newCmdTags(action string, subject any, tagList []string) (b []byte) {
	switch subject.(type) {
	case Host:
		b, _ = json.Marshal(commandHost{command{
			Action: action,
			Type:   command_type_host,
		}, Host{
			ID:   subject.(Host).ID,
			Tags: tagMapFromList(tagList),
		}})
	case Shard:
		b, _ = json.Marshal(commandShard{command{
			Action: action,
			Type:   command_type_shard,
		}, Shard{
			ID:   subject.(Shard).ID,
			Tags: tagMapFromList(tagList),
		}})
	case Replica:
		b, _ = json.Marshal(commandReplica{command{
			Action: action,
			Type:   command_type_replica,
		}, Replica{
			ID:   subject.(Replica).ID,
			Tags: tagMapFromList(tagList),
		}})
	}
	return
}

// tagMapFromList converts a list of tags to a map of key (namespace:predicate) and value
//
//	["geo:region=us-west-1"] -> {"geo:region": "us-west-1"}
func tagMapFromList(tagList []string) map[string]string {
	var m = map[string]string{}
	for _, ts := range tagList {
		if i := strings.Index(ts, "="); i >= 0 {
			m[ts[:i]] = ts[i:]
		} else {
			m[ts] = ""
		}
	}
	return m
}
