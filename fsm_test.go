package zongzi

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/lni/dragonboat/v4/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsmFactory(t *testing.T) {
	a, _ := NewAgent("test", []string{})
	f := fsmFactory(a)
	require.NotNil(t, f)
	ism := f(1, 2)
	require.NotNil(t, ism)
	assert.Equal(t, uint64(1), ism.(*fsm).shardID)
	assert.Equal(t, uint64(2), ism.(*fsm).replicaID)
	assert.Equal(t, ism.(*fsm), a.fsm)
}

func TestFsm(t *testing.T) {
	a := &Agent{
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
				entries, err := fsm.Update([]Entry{{Cmd: testHost}})
				require.Nil(t, err)
				require.Equal(t, 1, fsm.state.Hosts.Len())
				require.Equal(t, uint64(1), entries[0].Result.Value)
				item, ok := fsm.state.Hosts.Get(hostID)
				require.True(t, ok)
				require.NotNil(t, item)
				assert.Equal(t, hostID, item.ID)
				assert.Equal(t, "test-raft-addr", item.ApiAddress)
			})
			t.Run("delete", func(t *testing.T) {
				t.Run("empty", func(t *testing.T) {
					entries, err := fsm.Update([]Entry{{Cmd: newCmdHostDel(hostID)}})
					require.Nil(t, err)
					require.Equal(t, 0, fsm.state.Hosts.Len())
					require.Equal(t, uint64(1), entries[0].Result.Value)
					item, ok := fsm.state.Hosts.Get(hostID)
					require.False(t, ok)
					require.Nil(t, item)
				})
				t.Run("with-replicas", func(t *testing.T) {
					fsm.Update([]Entry{{Cmd: newCmdShardPut(shardID, shardType)}})
					fsm.Update([]Entry{{Cmd: testHost}})
					host, ok := fsm.state.Hosts.Get(hostID)
					require.True(t, ok)
					require.NotNil(t, host)
					for i := uint64(1); i < 5; i++ {
						entries, err := fsm.Update([]Entry{{Cmd: newCmdReplicaPut(hostID, shardID, i, false)}})
						require.Nil(t, err)
						require.Equal(t, i, entries[0].Result.Value)
					}
					require.Equal(t, 4, fsm.state.Replicas.Len())
					assert.Equal(t, 4, len(host.Replicas))
					entries, err := fsm.Update([]Entry{{Cmd: newCmdHostDel(hostID)}})
					require.Nil(t, err)
					require.Equal(t, 0, fsm.state.Hosts.Len())
					require.Equal(t, 0, fsm.state.Replicas.Len())
					require.Equal(t, uint64(1), entries[0].Result.Value)
					assert.Equal(t, 0, len(host.Replicas))
				})
			})
		})
		t.Run("shard", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			_, err := fsm.Update([]Entry{{Cmd: testHost}})
			require.Nil(t, err)
			require.Equal(t, 1, fsm.state.Hosts.Len())
			host, ok := fsm.state.Hosts.Get(hostID)
			require.True(t, ok)
			require.NotNil(t, host)
			t.Run("put", func(t *testing.T) {
				t.Run("new", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{
						Index: 10,
						Cmd:   newCmdShardPut(0, shardType),
					}})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.Shards.Len())
					require.Equal(t, uint64(10), res[0].Result.Value)
					item, ok := fsm.state.Shards.Get(10)
					require.True(t, ok)
					require.NotNil(t, item)
					assert.Equal(t, uint64(10), item.ID)
					assert.Equal(t, shardType, item.Type)
				})
				t.Run("existing", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{
						Index: 5,
						Cmd:   newCmdShardPut(shardID, shardType),
					}})
					require.Nil(t, err)
					require.Equal(t, 2, fsm.state.Shards.Len())
					require.Equal(t, shardID, res[0].Result.Value)
					item, ok := fsm.state.Shards.Get(shardID)
					require.True(t, ok)
					require.NotNil(t, item)
					assert.Equal(t, shardID, item.ID)
					assert.Equal(t, shardType, item.Type)
				})
			})
			t.Run("delete", func(t *testing.T) {
				t.Run("empty", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{Cmd: newCmdShardDel(10)}})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.Shards.Len())
					require.Equal(t, uint64(1), res[0].Result.Value)
					item, ok := fsm.state.Shards.Get(10)
					require.False(t, ok)
					require.Nil(t, item)
				})
				t.Run("with-replicas", func(t *testing.T) {
					var i uint64
					for i = 1; i < 5; i++ {
						res, err := fsm.Update([]Entry{{
							Index: i,
							Cmd:   newCmdReplicaPut(hostID, shardID, i, false),
						}})
						require.Nil(t, err)
						require.Equal(t, i, res[0].Result.Value)
					}
					require.Equal(t, 4, fsm.state.Replicas.Len())
					assert.Equal(t, 4, len(host.Replicas))
					res, err := fsm.Update([]Entry{{Cmd: newCmdShardDel(shardID)}})
					require.Nil(t, err)
					require.Equal(t, 0, fsm.state.Shards.Len())
					require.Equal(t, 0, fsm.state.Replicas.Len())
					require.Equal(t, uint64(1), res[0].Result.Value)
					assert.Equal(t, 0, len(host.Replicas))
				})
			})
		})
		t.Run("replica", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			_, err := fsm.Update([]Entry{{Cmd: testHost}})
			require.Nil(t, err)
			host, ok := fsm.state.Hosts.Get(hostID)
			require.True(t, ok)
			require.NotNil(t, host)
			_, err = fsm.Update([]Entry{{Cmd: newCmdShardPut(shardID, shardType)}})
			require.Nil(t, err)
			shard, ok := fsm.state.Shards.Get(shardID)
			require.True(t, ok)
			require.NotNil(t, shard)
			t.Run("put", func(t *testing.T) {
				t.Run("new", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{
						Index: 10,
						Cmd:   newCmdReplicaPut(hostID, shardID, 0, false),
					}})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.Replicas.Len())
					require.Equal(t, uint64(10), res[0].Result.Value)
					item, ok := fsm.state.Replicas.Get(10)
					require.True(t, ok)
					require.NotNil(t, item)
					assert.Equal(t, uint64(10), item.ID)
					assert.Equal(t, hostID, item.HostID)
					assert.False(t, item.IsNonVoting)
					assert.False(t, item.IsWitness)
					require.Equal(t, 1, len(host.Replicas))
					replica := host.Replicas[0]
					assert.True(t, ok)
					assert.Equal(t, shardID, replica.ShardID)
				})
				t.Run("existing", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{
						Index: 5,
						Cmd:   newCmdReplicaPut(hostID, shardID, 1, false),
					}})
					require.Nil(t, err)
					require.Equal(t, 2, fsm.state.Replicas.Len())
					require.Equal(t, uint64(1), res[0].Result.Value)
					item, ok := fsm.state.Replicas.Get(1)
					require.True(t, ok)
					require.NotNil(t, item)
					assert.Equal(t, uint64(1), item.ID)
					assert.Equal(t, hostID, item.HostID)
					assert.False(t, item.IsNonVoting)
					assert.False(t, item.IsWitness)
					require.Equal(t, 1, len(host.Replicas))
					replica := host.Replicas[0]
					assert.True(t, ok)
					assert.Equal(t, shardID, replica.ShardID)
				})
				t.Run("host-not-exist", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{
						Index: 20,
						Cmd:   newCmdReplicaPut(`shambles`, shardID, 0, false),
					}})
					require.Nil(t, err)
					require.Equal(t, 2, fsm.state.Replicas.Len())
					require.Equal(t, uint64(0), res[0].Result.Value)
				})
				t.Run("shard-not-exist", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{
						Index: 30,
						Cmd:   newCmdReplicaPut(hostID, 0, 0, false),
					}})
					require.Nil(t, err)
					require.Equal(t, 2, fsm.state.Replicas.Len())
					require.Equal(t, uint64(0), res[0].Result.Value)
				})
			})
			t.Run("delete", func(t *testing.T) {
				t.Run("success", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{Cmd: newCmdReplicaDel(10)}})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.Replicas.Len())
					require.Equal(t, uint64(1), res[0].Result.Value)
					item, ok := fsm.state.Replicas.Get(10)
					require.False(t, ok)
					require.Nil(t, item)
				})
				t.Run("nonexistant", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{Cmd: newCmdReplicaDel(10)}})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.Replicas.Len())
					require.Equal(t, uint64(1), res[0].Result.Value)
					item, ok := fsm.state.Replicas.Get(10)
					require.False(t, ok)
					require.Nil(t, item)
				})
			})
			t.Run("put-preserves-replicas", func(t *testing.T) {
				t.Run("host", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{Cmd: testHost}})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.Hosts.Len())
					require.Equal(t, uint64(1), res[0].Result.Value)
					require.Equal(t, 1, len(host.Replicas))
					replica := host.Replicas[0]
					assert.True(t, ok)
					assert.Equal(t, shardID, replica.ShardID)
				})
				t.Run("shard", func(t *testing.T) {
					res, err := fsm.Update([]Entry{{
						Cmd: newCmdShardPut(shardID, shardType),
					}})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.Shards.Len())
					require.Equal(t, shardID, res[0].Result.Value)
					require.Equal(t, 1, len(host.Replicas))
					replica := shard.Replicas[0]
					assert.True(t, ok)
					assert.Equal(t, hostID, replica.ID)
				})
			})
		})
		t.Run("unknown", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			t.Run("action", func(t *testing.T) {
				for _, ctype := range []string{
					command_type_host,
					command_type_replica,
					command_type_shard,
				} {
					cmd, err := json.Marshal(commandHost{command{
						Action: "squash",
						Type:   ctype,
					}, Host{
						ID: hostID,
					}})
					require.Nil(t, err)
					res, err := fsm.Update([]Entry{{Cmd: cmd}})
					require.NotNil(t, err)
					require.Equal(t, uint64(0), res[0].Result.Value)
					require.Equal(t, []byte(nil), res[0].Result.Data)
				}
			})
			t.Run("type", func(t *testing.T) {
				for _, action := range []string{
					command_action_put,
					command_action_del,
				} {
					cmd, err := json.Marshal(commandHost{command{
						Action: action,
						Type:   "banana",
					}, Host{
						ID: hostID,
					}})
					require.Nil(t, err)
					res, err := fsm.Update([]Entry{{Cmd: cmd}})
					require.NotNil(t, err)
					require.Equal(t, uint64(0), res[0].Result.Value)
					require.Equal(t, []byte(nil), res[0].Result.Data)
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
				res, err := fsm.Update([]Entry{{Cmd: []byte(cmd)}})
				require.NotNil(t, err, cmd)
				require.Equal(t, uint64(0), res[0].Result.Value)
				require.Equal(t, []byte(nil), res[0].Result.Data)
			}
		})
	})
	fsm1 := f(1, 2).(*fsm)
	var bb bytes.Buffer
	t.Run("SaveSnapshot", func(t *testing.T) {
		require.Nil(t, fsm1.SaveSnapshot(&bb, nil, nil))
	})
	fsm2 := f(1, 2).(*fsm)
	t.Run("RecoverFromSnapshot", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			require.Nil(t, fsm2.RecoverFromSnapshot(&bb, nil))
			assert.Equal(t, uint64(418), fsm2.state.Index)
			assert.Equal(t, 1, fsm2.state.Hosts.Len())
			assert.Equal(t, 1, fsm2.state.Shards.Len())
			assert.Equal(t, 1, fsm2.state.Replicas.Len())
		})
		t.Run("invalid", func(t *testing.T) {
			require.NotNil(t, fsm2.RecoverFromSnapshot(&bb, nil))
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
