package zongzi

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/lni/dragonboat/v4/logger"
	dbsm "github.com/lni/dragonboat/v4/statemachine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsmFactory(t *testing.T) {
	a := &agent{}
	f := fsmFactory(a)
	require.NotNil(t, f)
	ism := f(1, 2)
	require.NotNil(t, ism)
	assert.Equal(t, uint64(1), ism.(*fsm).shardID)
	assert.Equal(t, uint64(2), ism.(*fsm).replicaID)
	assert.Equal(t, ism.(*fsm), a.fsm)
}

func TestFsm(t *testing.T) {
	a := &agent{
		log: nullLogger{},
	}
	f := fsmFactory(a)
	hostID := "test-nh-id-1"
	shardID := uint64(2)
	shardType := `banana`
	testHost := newCmdHostPut(
		hostID,
		"test-raft-addr",
		[]byte("test-meta"),
		HostStatus_New,
		[]string{"test-shard-type-1", "test-shard-type-2"},
	)
	t.Run("Update", func(t *testing.T) {
		t.Run("host", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			t.Run("put", func(t *testing.T) {
				res, err := fsm.Update(dbsm.Entry{Cmd: testHost})
				require.Nil(t, err)
				require.Equal(t, 1, len(fsm.store.HostList()))
				require.Equal(t, uint64(1), res.Value)
				item := fsm.store.HostFind(hostID)
				require.NotNil(t, item)
				assert.Equal(t, hostID, item.ID)
				assert.Equal(t, "test-raft-addr", item.RaftAddr)
			})
			t.Run("delete", func(t *testing.T) {
				t.Run("empty", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{Cmd: newCmdHostDel(hostID)})
					require.Nil(t, err)
					require.Equal(t, 0, len(fsm.store.HostList()))
					require.Equal(t, uint64(1), res.Value)
					item := fsm.store.HostFind(hostID)
					assert.Nil(t, item)
				})
				t.Run("with-replicas", func(t *testing.T) {
					fsm.Update(dbsm.Entry{Cmd: newCmdShardPut(shardID, shardType)})
					fsm.Update(dbsm.Entry{Cmd: testHost})
					host := fsm.store.HostFind(hostID)
					require.NotNil(t, host)
					for i := uint64(1); i < 5; i++ {
						res, err := fsm.Update(dbsm.Entry{Cmd: newCmdReplicaPut(hostID, shardID, i, false)})
						require.Nil(t, err)
						require.Equal(t, i, res.Value)
					}
					require.Equal(t, 4, len(fsm.store.ReplicaList()))
					assert.Equal(t, 4, len(host.Replicas))
					res, err := fsm.Update(dbsm.Entry{Cmd: newCmdHostDel(hostID)})
					require.Nil(t, err)
					require.Equal(t, 0, len(fsm.store.HostList()))
					require.Equal(t, 0, len(fsm.store.ReplicaList()))
					require.Equal(t, uint64(1), res.Value)
					assert.Equal(t, 0, len(host.Replicas))
				})
			})
		})
		t.Run("shard", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			_, err := fsm.Update(dbsm.Entry{Cmd: testHost})
			require.Nil(t, err)
			require.Equal(t, 1, len(fsm.store.HostList()))
			host := fsm.store.HostFind(hostID)
			require.NotNil(t, host)
			t.Run("put", func(t *testing.T) {
				t.Run("new", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 10,
						Cmd:   newCmdShardPut(0, shardType),
					})
					require.Nil(t, err)
					require.Equal(t, 1, len(fsm.store.ShardList()))
					require.Equal(t, uint64(10), res.Value)
					item := fsm.store.ShardFind(10)
					require.NotNil(t, item)
					assert.Equal(t, uint64(10), item.ID)
					assert.Equal(t, shardType, item.Type)
				})
				t.Run("existing", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 5,
						Cmd:   newCmdShardPut(shardID, shardType),
					})
					require.Nil(t, err)
					require.Equal(t, 2, len(fsm.store.ShardList()))
					require.Equal(t, shardID, res.Value)
					item := fsm.store.ShardFind(shardID)
					require.NotNil(t, item)
					assert.Equal(t, shardID, item.ID)
					assert.Equal(t, shardType, item.Type)
				})
			})
			t.Run("delete", func(t *testing.T) {
				t.Run("empty", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{Cmd: newCmdShardDel(10)})
					require.Nil(t, err)
					require.Equal(t, 1, len(fsm.store.ShardList()))
					require.Equal(t, uint64(1), res.Value)
					item := fsm.store.ShardFind(10)
					assert.Nil(t, item)
				})
				t.Run("with-replicas", func(t *testing.T) {
					var i uint64
					for i = 1; i < 5; i++ {
						res, err := fsm.Update(dbsm.Entry{
							Index: i,
							Cmd:   newCmdReplicaPut(hostID, shardID, i, false),
						})
						require.Nil(t, err)
						require.Equal(t, i, res.Value)
					}
					require.Equal(t, 4, len(fsm.store.ReplicaList()))
					assert.Equal(t, 4, len(host.Replicas))
					res, err := fsm.Update(dbsm.Entry{Cmd: newCmdShardDel(shardID)})
					require.Nil(t, err)
					require.Equal(t, 0, len(fsm.store.ShardList()))
					require.Equal(t, 0, len(fsm.store.ReplicaList()))
					require.Equal(t, uint64(1), res.Value)
					assert.Equal(t, 0, len(host.Replicas))
				})
			})
		})
		t.Run("replica", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			_, err := fsm.Update(dbsm.Entry{Cmd: testHost})
			require.Nil(t, err)
			host := fsm.store.HostFind(hostID)
			require.NotNil(t, host)
			_, err = fsm.Update(dbsm.Entry{Cmd: newCmdShardPut(shardID, shardType)})
			require.Nil(t, err)
			shard := fsm.store.ShardFind(shardID)
			require.NotNil(t, shard)
			t.Run("put", func(t *testing.T) {
				t.Run("new", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 10,
						Cmd:   newCmdReplicaPut(hostID, shardID, 0, false),
					})
					require.Nil(t, err)
					require.Equal(t, 1, len(fsm.store.ReplicaList()))
					require.Equal(t, uint64(10), res.Value)
					item := fsm.store.ReplicaFind(10)
					require.NotNil(t, item)
					assert.Equal(t, uint64(10), item.ID)
					assert.Equal(t, hostID, item.HostID)
					assert.False(t, item.IsNonVoting)
					assert.False(t, item.IsWitness)
					id, ok := host.Replicas[item.ID]
					assert.True(t, ok)
					assert.Equal(t, shardID, id)
				})
				t.Run("existing", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 5,
						Cmd:   newCmdReplicaPut(hostID, shardID, 1, false),
					})
					require.Nil(t, err)
					require.Equal(t, 2, len(fsm.store.ReplicaList()))
					require.Equal(t, uint64(1), res.Value)
					item := fsm.store.ReplicaFind(1)
					require.NotNil(t, item)
					assert.Equal(t, uint64(1), item.ID)
					assert.Equal(t, hostID, item.HostID)
					assert.False(t, item.IsNonVoting)
					assert.False(t, item.IsWitness)
					id, ok := host.Replicas[item.ID]
					assert.True(t, ok)
					assert.Equal(t, shardID, id)
				})
				t.Run("host-not-exist", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 20,
						Cmd:   newCmdReplicaPut(`shambles`, shardID, 0, false),
					})
					require.Nil(t, err)
					require.Equal(t, 2, len(fsm.store.ReplicaList()))
					require.Equal(t, uint64(0), res.Value)
				})
				t.Run("shard-not-exist", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 30,
						Cmd:   newCmdReplicaPut(hostID, 0, 0, false),
					})
					require.Nil(t, err)
					require.Equal(t, 2, len(fsm.store.ReplicaList()))
					require.Equal(t, uint64(0), res.Value)
				})
			})
			t.Run("delete", func(t *testing.T) {
				t.Run("success", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{Cmd: newCmdReplicaDel(10)})
					require.Nil(t, err)
					require.Equal(t, 1, len(fsm.store.ReplicaList()))
					require.Equal(t, uint64(1), res.Value)
					item := fsm.store.ReplicaFind(10)
					require.Nil(t, item)
				})
				t.Run("nonexistant", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{Cmd: newCmdReplicaDel(10)})
					require.Nil(t, err)
					require.Equal(t, 1, len(fsm.store.ReplicaList()))
					require.Equal(t, uint64(1), res.Value)
					item := fsm.store.ReplicaFind(10)
					require.Nil(t, item)
				})
			})
			t.Run("put-preserves-replicas", func(t *testing.T) {
				t.Run("host", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{Cmd: testHost})
					require.Nil(t, err)
					require.Equal(t, 1, len(fsm.store.HostList()))
					require.Equal(t, uint64(1), res.Value)
					id, ok := host.Replicas[uint64(1)]
					assert.True(t, ok)
					assert.Equal(t, shardID, id)
				})
				t.Run("shard", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Cmd: newCmdShardPut(shardID, shardType),
					})
					require.Nil(t, err)
					require.Equal(t, 1, len(fsm.store.ShardList()))
					require.Equal(t, shardID, res.Value)
					id, ok := shard.Replicas[uint64(1)]
					assert.True(t, ok)
					assert.Equal(t, hostID, id)
				})
			})
		})
		t.Run("unknown", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			t.Run("action", func(t *testing.T) {
				for _, ctype := range []string{
					cmd_type_host,
					cmd_type_replica,
					cmd_type_shard,
				} {
					cmd, err := json.Marshal(cmdHost{cmd{
						Action: "squash",
						Type:   ctype,
					}, Host{
						ID: hostID,
					}})
					require.Nil(t, err)
					res, err := fsm.Update(dbsm.Entry{Cmd: cmd})
					require.NotNil(t, err)
					require.Equal(t, uint64(0), res.Value)
					require.Equal(t, []byte(nil), res.Data)
				}
			})
			t.Run("type", func(t *testing.T) {
				for _, action := range []string{
					cmd_action_put,
					cmd_action_del,
				} {
					cmd, err := json.Marshal(cmdHost{cmd{
						Action: action,
						Type:   "banana",
					}, Host{
						ID: hostID,
					}})
					require.Nil(t, err)
					res, err := fsm.Update(dbsm.Entry{Cmd: cmd})
					require.NotNil(t, err)
					require.Equal(t, uint64(0), res.Value)
					require.Equal(t, []byte(nil), res.Data)
				}
			})
		})
		t.Run("invalid", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			for _, cmd := range []string{
				`1`,
				`{"type":"host", "host":1}`,
				`{"type":"shard", "shard":[]}`,
				`{"type":"replica", "replica":false}`,
			} {
				res, err := fsm.Update(dbsm.Entry{Cmd: []byte(cmd)})
				require.NotNil(t, err, cmd)
				require.Equal(t, uint64(0), res.Value)
				require.Equal(t, []byte(nil), res.Data)
			}
		})
	})
	fsm1 := f(1, 2).(*fsm)
	t.Run("Query", func(t *testing.T) {
		t.Run("host", func(t *testing.T) {
			_, err := fsm1.Update(dbsm.Entry{Cmd: testHost})
			require.Nil(t, err)
			require.Equal(t, 1, len(fsm1.store.HostList()))
			t.Run("get", func(t *testing.T) {
				t.Run("found", func(t *testing.T) {
					val, err := fsm1.Lookup(newQueryHostGet(hostID))
					require.Nil(t, err)
					require.NotNil(t, val)
					assert.Equal(t, hostID, val.(Host).ID)
					assert.Equal(t, "test-raft-addr", val.(Host).RaftAddr)
				})
				t.Run("not-found", func(t *testing.T) {
					val, err := fsm1.Lookup(newQueryHostGet(`salami`))
					require.Nil(t, err)
					require.Nil(t, val)
				})
			})
			t.Run("unknown", func(t *testing.T) {
				val, err := fsm1.Lookup(queryHost{query{
					Action: `salami`,
				}, Host{}})
				require.NotNil(t, err)
				require.Nil(t, val)
			})
		})
		t.Run("snapshot", func(t *testing.T) {
			_, err := fsm1.Update(dbsm.Entry{Cmd: newCmdShardPut(shardID, shardType)})
			require.Nil(t, err)
			require.Equal(t, 1, len(fsm1.store.ShardList()))
			_, err = fsm1.Update(dbsm.Entry{
				Index: 418,
				Cmd:   newCmdReplicaPut(hostID, shardID, 1, false),
			})
			require.Nil(t, err)
			require.Equal(t, 1, len(fsm1.store.ReplicaList()))
			t.Run("get", func(t *testing.T) {
				val, err := fsm1.Lookup(newQuerySnapshotGet())
				require.Nil(t, err)
				require.NotNil(t, val)
				assert.Equal(t, uint64(418), val.(*Snapshot).Index)
				assert.Equal(t, 1, len(val.(*Snapshot).Hosts))
				assert.Equal(t, 1, len(val.(*Snapshot).Shards))
				assert.Equal(t, 1, len(val.(*Snapshot).Replicas))
			})
			t.Run("unknown", func(t *testing.T) {
				val, err := fsm1.Lookup(querySnapshot{query{
					Action: `salami`,
				}})
				require.NotNil(t, err)
				require.Nil(t, val)
			})
		})
		t.Run("invalid", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			val, err := fsm.Lookup(`salami`)
			require.NotNil(t, err)
			require.Nil(t, val)
		})
	})
	var bb bytes.Buffer
	t.Run("SaveSnapshot", func(t *testing.T) {
		require.Nil(t, fsm1.SaveSnapshot(&bb, nil, nil))
	})
	fsm2 := f(1, 2).(*fsm)
	t.Run("RecoverFromSnapshot", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			require.Nil(t, fsm2.RecoverFromSnapshot(&bb, nil, nil))
			val, err := fsm2.Lookup(newQuerySnapshotGet())
			require.Nil(t, err)
			require.NotNil(t, val)
			assert.Equal(t, uint64(418), val.(*Snapshot).Index)
			assert.Equal(t, 1, len(val.(*Snapshot).Hosts))
			assert.Equal(t, 1, len(val.(*Snapshot).Shards))
			assert.Equal(t, 1, len(val.(*Snapshot).Replicas))
		})
		t.Run("invalid", func(t *testing.T) {
			require.NotNil(t, fsm2.RecoverFromSnapshot(&bb, nil, nil))
		})
	})
	t.Run("Close", func(t *testing.T) {
		fsm := f(1, 2).(*fsm)
		require.Nil(t, fsm.Close())
	})
}

type nullLogger struct{}

func (nullLogger) SetLevel(logger.LogLevel)                    {}
func (nullLogger) Debugf(format string, args ...interface{})   {}
func (nullLogger) Infof(format string, args ...interface{})    {}
func (nullLogger) Warningf(format string, args ...interface{}) {}
func (nullLogger) Errorf(format string, args ...interface{})   {}
func (nullLogger) Panicf(format string, args ...interface{})   {}
