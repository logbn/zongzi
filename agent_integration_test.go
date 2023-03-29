//go:build !unit

package zongzi

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent(t *testing.T) {
	basedir := `/tmp/zongzi-test`
	agents := []*Agent{}
	defer func() {
		for _, a := range agents {
			a.Stop()
		}
	}()
	var err error
	os.RemoveAll(basedir)
	run := func(t *testing.T, peers []string, addresses ...[]string) {
		for _, addr := range addresses {
			a, err := NewAgent(`test001`, peers,
				WithApiAddress(addr[0]),
				WithGossipAddress(addr[1]),
				WithHostConfig(HostConfig{
					WALDir:         fmt.Sprintf(basedir+`/agent-%d/wal`, len(agents)),
					NodeHostDir:    fmt.Sprintf(basedir+`/agent-%d/raft`, len(agents)),
					RaftAddress:    addr[2],
					RTTMillisecond: 5,
				}))
			require.Nil(t, err)
			agents = append(agents, a)
			a.RegisterStateMachine(`test`, `v0.0.1`, func(shardID uint64, replicaID uint64) StateMachine {
				return &mockStateMachine{
					mockUpdate: func(e []Entry) []Entry {
						for i := range e {
							e[i].Result.Value = uint64(len(e[i].Cmd))
						}
						return e
					},
					mockQuery: func(ctx context.Context, data []byte) *Result {
						return &Result{Value: uint64(len(data))}
					},
					mockPrepareSnapshot: func() (cursor any, err error) {
						return
					},
					mockSaveSnapshot: func(cursor any, w io.Writer, c SnapshotFileCollection, close <-chan struct{}) error {
						return nil
					},
					mockRecoverFromSnapshot: func(r io.Reader, f []SnapshotFile, close <-chan struct{}) error {
						return nil
					},
					mockClose: func() error {
						return nil
					},
				}
			})
			a.RegisterPersistentStateMachine(`test-persistent`, `v0.0.1`, func(shardID uint64, replicaID uint64) PersistentStateMachine {
				var idx uint64
				return &mockPersistentStateMachine{
					mockOpen: func(stopc <-chan struct{}) (index uint64, err error) {
						return idx, nil
					},
					mockUpdate: func(e []Entry) []Entry {
						for i := range e {
							e[i].Result.Value = uint64(len(e[i].Cmd))
							idx = e[i].Index
						}
						return e
					},
					mockQuery: func(ctx context.Context, data []byte) *Result {
						return &Result{Value: uint64(len(data))}
					},
					mockPrepareSnapshot: func() (cursor any, err error) {
						return
					},
					mockSaveSnapshot: func(cursor any, w io.Writer, close <-chan struct{}) error {
						return nil
					},
					mockRecoverFromSnapshot: func(r io.Reader, close <-chan struct{}) error {
						return nil
					},
					mockSync: func() error {
						return nil
					},
					mockClose: func() error {
						return nil
					},
				}
			})
			go func(a *Agent) {
				for i := 0; i < 5; i++ {
					err = a.Start()
					require.Nil(t, err, `%#v`, err)
					if err == nil {
						break
					}
				}
				require.Nil(t, err, `%+v`, err)
			}(a)
		}
		var good bool
		// 10 seconds to start the cluster.
		for i := 0; i < 100; i++ {
			good = true
			for j := range agents {
				good = good && agents[j].Status() == AgentStatus_Ready
			}
			if good {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.True(t, good, `%+v`, agents)
		// 10 seconds for all host controllers to complete their first run.
		for i := 0; i < 100; i++ {
			good = true
			for j := range agents {
				good = good && agents[j].controller.index > 0
			}
			if good {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.True(t, good, `%#v`, agents)
	}
	peers := []string{
		`127.0.0.1:18011`,
		`127.0.0.1:18021`,
		`127.0.0.1:18031`,
	}
	t.Run(`start`, func(t *testing.T) {
		run(t, peers,
			[]string{`127.0.0.1:18011`, `127.0.0.1:18012`, `127.0.0.1:18013`},
			[]string{`127.0.0.1:18021`, `127.0.0.1:18022`, `127.0.0.1:18023`},
			[]string{`127.0.0.1:18031`, `127.0.0.1:18032`, `127.0.0.1:18033`},
		)
	})
	t.Run(`join`, func(t *testing.T) {
		run(t, peers,
			[]string{`127.0.0.1:18041`, `127.0.0.1:18042`, `127.0.0.1:18043`},
			[]string{`127.0.0.1:18051`, `127.0.0.1:18052`, `127.0.0.1:18053`},
			[]string{`127.0.0.1:18061`, `127.0.0.1:18062`, `127.0.0.1:18063`},
		)
		// 5 seconds for all hosts to see themselves with at least one active replica
		var good bool
		var replicas []Replica
		for i := 0; i < 100; i++ {
			good = true
			replicas = replicas[:0]
			for j, a := range agents {
				agents[j].Read(func(s State) {
					var replicaCount = 0
					s.ReplicaIterateByHostID(a.HostID(), func(r Replica) bool {
						replicas = append(replicas, r)
						if r.Status == ReplicaStatus_Active {
							replicaCount++
						}
						return true
					})
					good = good && replicaCount > 0
				}, true)
			}
			if good {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.True(t, good, `%+v`, replicas)
	})
	var shard Shard
	t.Run(`create shard`, func(t *testing.T) {
		shard, err = agents[0].CreateShard(`test`, `v0.0.1`)
		require.Nil(t, err)
		t.Logf(`%#v`, shard)
	})
	require.Nil(t, err)
	t.Run(`create replicas`, func(t *testing.T) {
		var replicaID uint64
		for i := 0; i < len(agents); i++ {
			replicaID, err = agents[i].CreateReplica(shard.ID, agents[i].HostID(), i > 2)
			require.Nil(t, err)
			require.NotEqual(t, 0, replicaID)
		}
		var replicaCount int
		var replicas []Replica
		// 10 seconds for replicas to be active on all hosts
		for i := 0; i < 100; i++ {
			replicaCount = 0
			replicas = replicas[:0]
			agents[0].Read(func(s State) {
				s.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
					replicas = append(replicas, r)
					if r.Status == ReplicaStatus_Active {
						replicaCount++
					}
					return true
				})
			})
			if replicaCount == len(agents) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.Equal(t, len(agents), replicaCount, `%#v`, replicas)
	})
	t.Run(`create persistent shard`, func(t *testing.T) {
		shard, err = agents[0].CreateShard(`test`, `v0.0.1`)
		require.Nil(t, err)
		t.Logf(`%#v`, shard)
	})
	require.Nil(t, err)
	t.Run(`create persistent replicas`, func(t *testing.T) {
		var replicaID uint64
		for i := 0; i < len(agents); i++ {
			replicaID, err = agents[i].CreateReplica(shard.ID, agents[i].HostID(), i > 2)
			require.Nil(t, err)
			require.NotEqual(t, 0, replicaID)
		}
		var replicaCount int
		var replicas []Replica
		// 10 seconds for replicas to be active on all hosts
		for i := 0; i < 100; i++ {
			replicaCount = 0
			replicas = replicas[:0]
			agents[0].Read(func(s State) {
				s.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
					replicas = append(replicas, r)
					if r.Status == ReplicaStatus_Active {
						replicaCount++
					}
					return true
				})
			})
			if replicaCount == len(agents) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.Equal(t, len(agents), replicaCount, `%#v`, replicas)
	})
}
