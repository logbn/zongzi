//go:build !unit

package zongzi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent(t *testing.T) {
	// SetLogLevel(LogLevelDebug)
	basedir := `/tmp/zongzi-test`
	agents := []*Agent{}
	defer func() {
		for _, a := range agents {
			a.Stop()
		}
	}()
	var err error
	os.RemoveAll(basedir)
	start := func(t *testing.T, peers []string, notifyCommit bool, addresses ...[]string) {
		for _, addr := range addresses {
			a, err := NewAgent(`test001`, peers,
				WithApiAddress(addr[0]),
				WithGossipAddress(addr[1]),
				WithHostConfig(HostConfig{
					WALDir:         fmt.Sprintf(basedir+`/agent-%d/wal`, len(agents)),
					NodeHostDir:    fmt.Sprintf(basedir+`/agent-%d/raft`, len(agents)),
					RaftAddress:    addr[2],
					RTTMillisecond: 5,
					NotifyCommit:   notifyCommit,
				}))
			require.Nil(t, err)
			// a.log.SetLevel(LogLevelDebug)
			agents = append(agents, a)
			a.RegisterStateMachine(`concurrent`, `v0.0.1`, mockConcurrentSM)
			a.RegisterPersistentStateMachine(`persistent`, `v0.0.1`, mockPersistentSM)
			go func(a *Agent) {
				err = a.Start()
				require.Nil(t, err, `%+v`, err)
			}(a)
		}
		// 10 seconds to start the cluster.
		require.True(t, await(10, 100, func() bool {
			for j := range agents {
				if agents[j].Status() != AgentStatus_Ready {
					return false
				}
			}
			return true
		}), `%#v`, agents)
	}
	peers := []string{
		`127.0.0.1:18011`,
		`127.0.0.1:18021`,
		`127.0.0.1:18031`,
	}
	t.Run(`start`, func(t *testing.T) {
		start(t, peers, false,
			[]string{`127.0.0.1:18011`, `127.0.0.1:18012`, `127.0.0.1:18013`},
			[]string{`127.0.0.1:18021`, `127.0.0.1:18022`, `127.0.0.1:18023`},
			[]string{`127.0.0.1:18031`, `127.0.0.1:18032`, `127.0.0.1:18033`},
		)
	})
	t.Run(`join`, func(t *testing.T) {
		start(t, peers, true,
			[]string{`127.0.0.1:18041`, `127.0.0.1:18042`, `127.0.0.1:18043`},
			[]string{`127.0.0.1:18051`, `127.0.0.1:18052`, `127.0.0.1:18053`},
			[]string{`127.0.0.1:18061`, `127.0.0.1:18062`, `127.0.0.1:18063`},
		)
		// 5 seconds for all hosts to see themselves with at least one active replica
		var replicas []Replica
		require.True(t, await(5, 100, func() bool {
			var replicaCount = 0
			replicas = replicas[:0]
			for j, a := range agents {
				agents[j].Read(func(s State) {
					s.ReplicaIterateByHostID(a.HostID(), func(r Replica) bool {
						replicas = append(replicas, r)
						if r.Status == ReplicaStatus_Active {
							replicaCount++
						}
						return true
					})
				}, true)
			}
			return replicaCount == len(agents)
		}), `%+v`, replicas)
	})
	for _, sm := range []string{`concurrent`, `persistent`} {
		var shard Shard
		t.Run(sm+` shard create`, func(t *testing.T) {
			shard, err = agents[0].CreateShard(sm, `v0.0.1`)
			require.Nil(t, err)
		})
		t.Run(sm+` replica create`, func(t *testing.T) {
			var replicaID uint64
			for i := 0; i < len(agents); i++ {
				replicaID, err = agents[0].CreateReplica(shard.ID, agents[i].HostID(), i > 2)
				require.Nil(t, err)
				require.NotEqual(t, 0, replicaID)
			}
			var replicas []Replica
			// 10 seconds for replicas to be active on all hosts
			require.True(t, await(10, 100, func() bool {
				var replicaCount = 0
				replicas = replicas[:0]
				agents[0].Read(func(s State) {
					s.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
						replicas = append(replicas, r)
						if r.Status == ReplicaStatus_Active {
							replicaCount++
						}
						return true
					})
				}, true)
				return replicaCount == len(agents)
			}), `%+v`, replicas)
		})
		for _, a := range agents {
			require.Nil(t, a.readIndex(shard.ID))
		}
		for _, op := range []string{"update", "query"} {
			for _, linearity := range []string{"linear", "non-linear"} {
				t.Run(fmt.Sprintf(`%s %s %s`, sm, op, linearity), func(t *testing.T) {
					runAgentSubTest(t, agents, shard, sm, op, linearity == "linear")
				})
			}
		}
	}
}

func runAgentSubTest(t *testing.T, agents []*Agent, shard Shard, sm, op string, linear bool) {
	var i = 0
	var nonvoting = 0
	var val uint64
	var err error
	agents[0].Read(func(s State) {
		s.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
			if r.IsNonVoting {
				nonvoting++
				return true
			}
			replicaClient := agents[0].GetReplicaClient(r.ID)
			require.NotNil(t, replicaClient)
			if op == "update" {
				val, _, err = replicaClient.Propose(raftCtx(), bytes.Repeat([]byte("test"), i+1), linear)
			} else {
				val, _, err = replicaClient.Query(raftCtx(), bytes.Repeat([]byte("test"), i+1), linear)
			}
			require.Nil(t, err, `%v, %v, %#v`, i, err, replicaClient)
			if op == "update" && !linear {
				assert.Equal(t, uint64(0), val)
			} else {
				assert.Equal(t, uint64((i+1)*4), val)
			}
			i++
			return true
		})
	}, true)
	assert.Equal(t, 3, nonvoting)
}

func await(d, n time.Duration, fn func() bool) bool {
	for i := 0; i < int(n); i++ {
		if fn() {
			return true
		}
		time.Sleep(d * time.Second / n)
	}
	return false
}

var mockConcurrentSM = func(shardID uint64, replicaID uint64) StateMachine {
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
}

var mockPersistentSM = func(shardID uint64, replicaID uint64) PersistentStateMachine {
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
}
