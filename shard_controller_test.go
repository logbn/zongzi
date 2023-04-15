package zongzi

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShardController(t *testing.T) {
	t.Run("reconcile", func(t *testing.T) {
		state := newFsmStateRadix()
		testHelperFillHosts(state, 3)
		state = state.withTxn(true)
		state.shardPut(Shard{ID: 0, Name: "zongzi"})
		state.metaSetIndex(1)
		state.commit()
		shard := Shard{
			ID:      1,
			Updated: 2,
			Status:  ShardStatus_New,
			Type:    "test",
			Tags:    map[string]string{},
		}
		t.Run("members", func(t *testing.T) {
			state = state.withTxn(true)
			defer state.commit()
			require.Nil(t, WithName("test-1")(&shard))
			require.Nil(t, WithPlacementMembers(3, "geo:region=us-west")(&shard))
			state.shardPut(shard)
			agent, err := NewAgent("test", nil)
			require.Nil(t, err)
			ctrl := shardController{
				agent: agent,
				cluster: &mockCluster{
					mockReplicaCreate: func(hostID string, shardID uint64, isNonVoting bool) (id uint64, err error) {
						i := state.Index()
						id = state.replicaIncr()
						state.metaSetIndex(i + 1)
						state.replicaPut(Replica{
							ID:          id,
							Updated:     i,
							HostID:      hostID,
							ShardID:     shardID,
							IsNonVoting: isNonVoting,
							Tags:        map[string]string{},
						})
						state.shardTouch(shardID, i)
						state.hostTouch(hostID, i)
						return
					},
				},
			}
			updated, err := ctrl.reconcile(state, shard)
			require.Nil(t, err)
			assert.True(t, updated)
			var n int
			state.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
				n++
				// t.Logf(`%#v`, r)
				return true
			})
			assert.Equal(t, 3, n)
		})
		testHelperFillHosts(state, 3)
		t.Run("replicas", func(t *testing.T) {
			state = state.withTxn(true)
			defer state.commit()
			require.Nil(t, WithName("test-1")(&shard))
			require.Nil(t, WithPlacementReplicas("group1", 3, "geo:region=us-west")(&shard))
			shard.Updated = state.Index() + 1
			state.shardPut(shard)
			agent, err := NewAgent("test", nil)
			require.Nil(t, err)
			ctrl := shardController{
				agent: agent,
				cluster: &mockCluster{
					mockReplicaCreate: func(hostID string, shardID uint64, isNonVoting bool) (id uint64, err error) {
						i := state.Index()
						id = state.replicaIncr()
						state.metaSetIndex(i + 1)
						state.replicaPut(Replica{
							ID:          id,
							Updated:     i,
							HostID:      hostID,
							ShardID:     shardID,
							IsNonVoting: isNonVoting,
							Tags:        map[string]string{},
						})
						state.shardTouch(shardID, i)
						state.hostTouch(hostID, i)
						return
					},
				},
			}
			updated, err := ctrl.reconcile(state, shard)
			require.Nil(t, err)
			assert.True(t, updated)
			var n int
			state.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
				n++
				// t.Logf(`%#v`, r)
				return true
			})
			assert.Equal(t, 6, n)
		})
	})
}

var uuidIncr uint64

func testHelperFillHosts(state *State, n uint64) {
	state = state.withTxn(true)
	defer state.commit()
	for i := uint64(1); i <= n; i++ {
		state.hostPut(Host{
			ID: fmt.Sprintf("8201349e-c113-4504-8d7e-25514c3c%04x", uuidIncr+i),
			Tags: map[string]string{
				"geo:region": "us-west",
				"geo:zone":   fmt.Sprintf("us-west-%d", i),
			},
		})
		state.replicaPut(Replica{
			ID:      i,
			HostID:  fmt.Sprintf("8201349e-c113-4504-8d7e-25514c3c%04x", uuidIncr+i),
			ShardID: 0,
			Tags:    map[string]string{},
		})
		uuidIncr++
	}
}

type mockCluster struct {
	mockReplicaCreate func(hostID string, shardID uint64, isNonVoting bool) (id uint64, err error)
}

func (m *mockCluster) ReplicaCreate(hostID string, shardID uint64, isNonVoting bool) (id uint64, err error) {
	return m.mockReplicaCreate(hostID, shardID, isNonVoting)
}
