package zongzi

import (
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
	t.Run("Update", func(t *testing.T) {
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
		t.Run("host", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			t.Run("put", func(t *testing.T) {
				res, err := fsm.Update(dbsm.Entry{Cmd: testHost})
				require.Nil(t, err)
				require.Equal(t, 1, fsm.state.hosts.Len())
				require.Equal(t, uint64(1), res.Value)
				item, ok := fsm.state.hosts.Get(hostID)
				require.True(t, ok)
				assert.Equal(t, hostID, item.ID)
				assert.Equal(t, "test-raft-addr", item.RaftAddr)
			})
			t.Run("delete", func(t *testing.T) {
				res, err := fsm.Update(dbsm.Entry{Cmd: newCmdHostDel(hostID)})
				require.Nil(t, err)
				require.Equal(t, 0, fsm.state.hosts.Len())
				require.Equal(t, uint64(1), res.Value)
				item, ok := fsm.state.hosts.Get(hostID)
				assert.Nil(t, item)
				assert.False(t, ok)
			})
		})
		t.Run("shard", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			_, err := fsm.Update(dbsm.Entry{Cmd: testHost})
			require.Nil(t, err)
			require.Equal(t, 1, fsm.state.hosts.Len())
			t.Run("put", func(t *testing.T) {
				t.Run("new", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 10,
						Cmd:   newCmdShardPut(0, shardType),
					})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.shards.Len())
					require.Equal(t, uint64(10), res.Value)
					item, ok := fsm.state.shards.Get(10)
					require.True(t, ok)
					assert.Equal(t, uint64(10), item.ID)
					assert.Equal(t, shardType, item.Type)
				})
				t.Run("existing", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 5,
						Cmd:   newCmdShardPut(shardID, shardType),
					})
					require.Nil(t, err)
					require.Equal(t, 2, fsm.state.shards.Len())
					require.Equal(t, shardID, res.Value)
					item, ok := fsm.state.shards.Get(shardID)
					require.True(t, ok)
					assert.Equal(t, shardID, item.ID)
					assert.Equal(t, shardType, item.Type)
				})
			})
			t.Run("delete", func(t *testing.T) {
				res, err := fsm.Update(dbsm.Entry{Cmd: newCmdShardDel(10)})
				require.Nil(t, err)
				require.Equal(t, 1, fsm.state.shards.Len())
				require.Equal(t, uint64(1), res.Value)
				item, ok := fsm.state.shards.Get(10)
				assert.Nil(t, item)
				assert.False(t, ok)
			})
		})
		t.Run("replica", func(t *testing.T) {
			fsm := f(1, 2).(*fsm)
			_, err := fsm.Update(dbsm.Entry{Cmd: testHost})
			require.Nil(t, err)
			host, ok := fsm.state.hosts.Get(hostID)
			require.NotNil(t, host)
			require.True(t, ok)
			_, err = fsm.Update(dbsm.Entry{
				Index: 5,
				Cmd:   newCmdShardPut(shardID, shardType),
			})
			require.Nil(t, err)
			shard, ok := fsm.state.shards.Get(shardID)
			require.NotNil(t, shard)
			require.True(t, ok)
			t.Run("put", func(t *testing.T) {
				t.Run("new", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 10,
						Cmd:   newCmdReplicaPut(hostID, shardID, 0, false),
					})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.replicas.Len())
					require.Equal(t, uint64(10), res.Value)
					item, ok := fsm.state.replicas.Get(10)
					require.True(t, ok)
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
					require.Equal(t, 2, fsm.state.replicas.Len())
					require.Equal(t, uint64(1), res.Value)
					item, ok := fsm.state.replicas.Get(1)
					require.True(t, ok)
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
					require.Equal(t, 2, fsm.state.replicas.Len())
					require.Equal(t, uint64(0), res.Value)
				})
				t.Run("shard-not-exist", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 30,
						Cmd:   newCmdReplicaPut(hostID, 0, 0, false),
					})
					require.Nil(t, err)
					require.Equal(t, 2, fsm.state.replicas.Len())
					require.Equal(t, uint64(0), res.Value)
				})
			})
			t.Run("delete", func(t *testing.T) {
				res, err := fsm.Update(dbsm.Entry{Cmd: newCmdReplicaDel(10)})
				require.Nil(t, err)
				require.Equal(t, 1, fsm.state.replicas.Len())
				require.Equal(t, uint64(1), res.Value)
				item, ok := fsm.state.replicas.Get(10)
				assert.Nil(t, item)
				assert.False(t, ok)
			})
			t.Run("put-preserves-replicas", func(t *testing.T) {
				t.Run("host", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{Cmd: testHost})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.hosts.Len())
					require.Equal(t, uint64(1), res.Value)
					id, ok := host.Replicas[uint64(1)]
					assert.True(t, ok)
					assert.Equal(t, shardID, id)
				})
				t.Run("shard", func(t *testing.T) {
					res, err := fsm.Update(dbsm.Entry{
						Index: 5,
						Cmd:   newCmdShardPut(shardID, shardType),
					})
					require.Nil(t, err)
					require.Equal(t, 1, fsm.state.shards.Len())
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
					require.Nil(t, err)
					require.Equal(t, uint64(0), res.Value)
					require.Equal(t, []byte(nil), res.Data)
				}
			})
			t.Run("type", func(t *testing.T) {
				for _, action := range []string{
					cmd_action_put,
					cmd_action_del,
					cmd_action_set_status,
				} {
					cmd, err := json.Marshal(cmdHost{cmd{
						Action: action,
						Type:   "banana",
					}, Host{
						ID: hostID,
					}})
					require.Nil(t, err)
					res, err := fsm.Update(dbsm.Entry{Cmd: cmd})
					require.Nil(t, err)
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
}

type nullLogger struct{}

func (nullLogger) SetLevel(logger.LogLevel)                    {}
func (nullLogger) Debugf(format string, args ...interface{})   {}
func (nullLogger) Infof(format string, args ...interface{})    {}
func (nullLogger) Warningf(format string, args ...interface{}) {}
func (nullLogger) Errorf(format string, args ...interface{})   {}
func (nullLogger) Panicf(format string, args ...interface{})   {}
