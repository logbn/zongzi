//go:build !unit

package zongzi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	updates     int
	update_time time.Duration
)

func TestAgent(t *testing.T) {
	// SetLogLevel(LogLevelDebug)
	basedir := `./tmp/zongzi-test`
	agents := []*Agent{}
	var err error
	var ctx = context.Background()
	os.RemoveAll(basedir)
	start := func(t *testing.T, peers []string, class string, notifyCommit bool, addresses ...[]string) {
		for i, addr := range addresses {
			a, err := NewAgent(`test001`, peers,
				WithApiAddress(addr[0]),
				WithGossipAddress(addr[2]),
				WithHostConfig(HostConfig{
					WALDir:         fmt.Sprintf(basedir+`/agent-%d/wal`, len(agents)),
					NodeHostDir:    fmt.Sprintf(basedir+`/agent-%d/raft`, len(agents)),
					RaftAddress:    addr[1],
					RTTMillisecond: 5,
					NotifyCommit:   notifyCommit,
				}),
				WithHostTags(
					fmt.Sprintf(`geo:zone=%d`, i%3),
					`node:class=`+class,
					`test:tag=1234`,
					`test:novalue`,
				))
			require.Nil(t, err)
			// a.log.SetLevel(LogLevelDebug)
			agents = append(agents, a)
			a.StateMachineRegister(`concurrent`, mockConcurrentSM)
			a.StateMachineRegister(`persistent`, mockPersistentSM)
			go func(a *Agent) {
				err = a.Start(ctx)
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
		start(t, peers, `concurrent`, false,
			[]string{`127.0.0.1:18011`, `127.0.0.1:18012`, `127.0.0.1:18013`},
			[]string{`127.0.0.1:18021`, `127.0.0.1:18022`, `127.0.0.1:18023`},
			[]string{`127.0.0.1:18031`, `127.0.0.1:18032`, `127.0.0.1:18033`},
			[]string{`127.0.0.1:18111`, `127.0.0.1:18112`, `127.0.0.1:18113`},
			[]string{`127.0.0.1:18121`, `127.0.0.1:18122`, `127.0.0.1:18123`},
			[]string{`127.0.0.1:18131`, `127.0.0.1:18132`, `127.0.0.1:18133`},
		)
	})
	t.Run(`join`, func(t *testing.T) {
		start(t, peers, `persistent`, true,
			[]string{`127.0.0.1:18041`, `127.0.0.1:18042`, `127.0.0.1:18043`},
			[]string{`127.0.0.1:18051`, `127.0.0.1:18052`, `127.0.0.1:18053`},
			[]string{`127.0.0.1:18061`, `127.0.0.1:18062`, `127.0.0.1:18063`},
			[]string{`127.0.0.1:18071`, `127.0.0.1:18072`, `127.0.0.1:18073`},
			[]string{`127.0.0.1:18081`, `127.0.0.1:18082`, `127.0.0.1:18083`},
			[]string{`127.0.0.1:18091`, `127.0.0.1:18092`, `127.0.0.1:18093`},
		)
		// 5 seconds for all hosts to see themselves with at least one active replica
		var replicas []Replica
		require.True(t, await(5, 100, func() bool {
			var replicaCount = 0
			replicas = replicas[:0]
			for j, a := range agents {
				agents[j].State(ctx, func(s *State) {
					s.ReplicaIterateByHostID(a.hostID(), func(r Replica) bool {
						replicas = append(replicas, r)
						if r.Status == ReplicaStatus_Active {
							replicaCount++
						}
						return true
					})
				})
			}
			return replicaCount == len(agents)
		}), `%+v`, replicas)
	})
	for _, sm := range []string{`concurrent`, `persistent`} {
		var shard Shard
		var created bool
		var othersm = `persistent`
		if sm == `persistent` {
			othersm = `concurrent`
		}
		t.Run(sm+` shard create`, func(t *testing.T) {
			shard, created, err = agents[0].ShardCreate(ctx, sm,
				WithPlacementVary(`geo:zone`),
				WithPlacementMembers(3, `node:class=`+sm),
				WithPlacementReplicas(sm, 3, `node:class=`+sm),
				WithPlacementReplicas(othersm, 6, `node:class=`+othersm),
				WithName(sm))
			require.Nil(t, err)
			require.True(t, created)
			var replicas []Replica
			// 10 seconds for replicas to be active on all hosts
			require.True(t, await(10, 100, func() bool {
				var replicaCount = 0
				replicas = replicas[:0]
				agents[0].State(ctx, func(s *State) {
					s.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
						replicas = append(replicas, r)
						if r.Status == ReplicaStatus_Active {
							replicaCount++
						}
						return true
					})
				})
				return replicaCount == len(agents)
			}), `%+v`, replicas)
		})
		for _, a := range agents {
			require.Nil(t, a.index(ctx, shard.ID))
		}
		for _, op := range []string{"update", "query", "watch"} {
			for _, linearity := range []string{"linear", "non-linear"} {
				t.Run(fmt.Sprintf(`%s %s %s host client`, sm, op, linearity), func(t *testing.T) {
					runAgentSubTest(t, agents, shard, op, linearity != "linear")
				})
				t.Run(fmt.Sprintf(`%s %s %s shard client`, sm, op, linearity), func(t *testing.T) {
					runAgentSubTestByShard(t, agents, shard, op, linearity != "linear")
				})
			}
		}
	}
	t.Run(`shard cover`, func(t *testing.T) {
		shard, created, err := agents[0].ShardCreate(ctx, `concurrent`,
			WithPlacementVary(`geo:zone`),
			WithPlacementMembers(3, `node:class=concurrent`),
			WithPlacementCover(`test:tag=1234`),
			WithName(`cover-test`))
		require.Nil(t, err)
		require.True(t, created)
		var replicas []Replica
		// 10 seconds for replicas to be active on all hosts
		require.True(t, await(10, 100, func() bool {
			var replicaCount = 0
			replicas = replicas[:0]
			agents[0].State(ctx, func(s *State) {
				s.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
					replicas = append(replicas, r)
					if r.Status == ReplicaStatus_Active {
						replicaCount++
					}
					return true
				})
			})
			return replicaCount == len(agents)
		}), `%+v`, replicas)
	})
	t.Run(`shard cover multi tag`, func(t *testing.T) {
		shard, created, err := agents[0].ShardCreate(ctx, `concurrent`,
			WithPlacementVary(`geo:zone`),
			WithPlacementMembers(3, `node:class=concurrent`),
			WithPlacementCover(`test:tag=1234`, `node:class=concurrent`),
			WithName(`cover-test-2`))
		require.Nil(t, err)
		require.True(t, created)
		var replicas []Replica
		// 10 seconds for replicas to be active on all hosts
		require.True(t, await(10, 100, func() bool {
			var replicaCount = 0
			replicas = replicas[:0]
			agents[0].State(ctx, func(s *State) {
				s.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
					replicas = append(replicas, r)
					if r.Status == ReplicaStatus_Active {
						replicaCount++
					}
					return true
				})
			})
			return replicaCount == 6
		}), `%+v`, replicas)
	})
	t.Run(`shard cover no value`, func(t *testing.T) {
		shard, created, err := agents[0].ShardCreate(ctx, `concurrent`,
			WithPlacementVary(`geo:zone`),
			WithPlacementMembers(3, `node:class=concurrent`),
			WithPlacementCover(`test:novalue`),
			WithName(`cover-test-3`))
		require.Nil(t, err)
		require.True(t, created)
		var replicas []Replica
		// 10 seconds for replicas to be active on all hosts
		require.True(t, await(10, 100, func() bool {
			var replicaCount = 0
			replicas = replicas[:0]
			agents[0].State(ctx, func(s *State) {
				s.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
					replicas = append(replicas, r)
					if r.Status == ReplicaStatus_Active {
						replicaCount++
					}
					return true
				})
			})
			return replicaCount == len(agents)
		}), `%+v`, replicas)
	})
	t.Run(`host restart`, func(t *testing.T) {
		t.Run(`stop`, func(t *testing.T) {
			agents[0].Stop()
			// 5 seconds for the host to transition to stopped
			require.True(t, await(5, 100, func() bool {
				return agents[0].Status() == AgentStatus_Stopped
			}), `%s`, mustJson(agents))
		})
		t.Run(`start`, func(t *testing.T) {
			agents[0].Start(ctx)
			// 5 seconds for the host to transition to active
			require.True(t, await(10, 100, func() bool {
				return agents[0].Status() == AgentStatus_Ready
			}), `%v`, agents[0].Status())
			require.True(t, await(5, 100, func() (success bool) {
				agents[0].State(ctx, func(s *State) {
					host, ok := s.Host(agents[0].hostID())
					success = ok && host.Status == HostStatus_Active
				})
				return
			}), `%s`, mustJson(agents))
		})
	})
	t.Run(`stop all`, func(t *testing.T) {
		for _, agent := range agents {
			agent.Stop()
			// 5 seconds for the host to transition to stopped
			require.True(t, await(5, 100, func() bool {
				return agent.Status() == AgentStatus_Stopped
			}), `%s`, mustJson(agents))
		}
	})
	fmt.Printf("UPDATES: %d\nAverage: %v\n", updates, update_time/time.Duration(updates))
}

func mustJson(in any) string {
	b, _ := json.Marshal(in)
	return string(b)
}

func runAgentSubTest(t *testing.T, agents []*Agent, shard Shard, op string, stale bool) {
	var i = 0
	var err error
	var val uint64
	var nonvoting = 0
	agents[0].State(context.Background(), func(s *State) {
		s.ReplicaIterateByShardID(shard.ID, func(r Replica) bool {
			if op == "update" && r.IsNonVoting {
				nonvoting++
				return true
			}
			val = 0
			client := agents[0].hostClient(r.HostID)
			require.NotNil(t, client)
			if op == "update" && stale {
				start := time.Now()
				err = client.Commit(raftCtx(), shard.ID, bytes.Repeat([]byte("test"), i+1))
				update_time += time.Since(start)
				updates++
			} else if op == "update" && !stale {
				val, _, err = client.Apply(raftCtx(), shard.ID, bytes.Repeat([]byte("test"), i+1))
			} else if op == "query" {
				val, _, err = client.Read(raftCtx(), shard.ID, bytes.Repeat([]byte("test"), i+1), stale)
				assert.Nil(t, err)
			} else if op == "watch" {
				res := make(chan *Result)
				done := make(chan bool)
				n := uint64(0)
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					for {
						select {
						case result := <-res:
							val = result.Value + 1
							assert.Equal(t, n, result.Value)
							n++
						case <-done:
							wg.Done()
							return
						}
					}
				}()
				err = client.Watch(raftCtx(), shard.ID, bytes.Repeat([]byte("test"), i+1), res, stale)
				close(done)
				wg.Wait()
				assert.Equal(t, uint64((i+1)*4), n)
				assert.Nil(t, err)
			} else {
				t.Error("Invalid op" + op)
			}
			require.Nil(t, err, `%v, %v, %#v`, i, err, client)
			if op == "update" && stale {
				assert.Equal(t, uint64(0), val)
			} else {
				assert.Equal(t, uint64((i+1)*4), val)
			}
			i++
			return true
		})
	})
	if op == "update" {
		assert.Equal(t, 9, nonvoting)
	}
}

func runAgentSubTestByShard(t *testing.T, agents []*Agent, shard Shard, op string, stale bool) {
	var i = 0
	var err error
	var val uint64
	for _, a := range agents {
		val = 0
		client := a.Client(shard.ID)
		require.NotNil(t, client)
		if op == "update" && stale {
			err = client.Commit(raftCtx(), bytes.Repeat([]byte("test"), i+1))
		} else if op == "update" && !stale {
			val, _, err = client.Apply(raftCtx(), bytes.Repeat([]byte("test"), i+1))
		} else if op == "query" {
			val, _, err = client.Read(raftCtx(), bytes.Repeat([]byte("test"), i+1), stale)
		} else if op == "watch" {
			res := make(chan *Result)
			done := make(chan bool)
			n := uint64(0)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				for {
					select {
					case result := <-res:
						val = result.Value + 1
						assert.Equal(t, n, result.Value)
						n++
					case <-done:
						wg.Done()
						return
					}
				}
			}()
			err = client.Watch(raftCtx(), bytes.Repeat([]byte("test"), i+1), res, stale)
			close(done)
			wg.Wait()
			assert.Equal(t, uint64((i+1)*4), n)
			assert.Nil(t, err)
		} else {
			t.Error("Invalid op" + op)
		}
		require.Nil(t, err, `%v, %v, %#v`, i, err, client)
		if op == "update" && stale {
			assert.Equal(t, uint64(0), val)
		} else {
			assert.Equal(t, uint64((i+1)*4), val)
		}
		i++
	}
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
		mockWatch: func(ctx context.Context, data []byte, results chan<- *Result) {
			for i := 0; i < len(data); i++ {
				results <- &Result{Value: uint64(i)}
			}
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

var idx = map[string]uint64{}
var mutex sync.Mutex

var mockPersistentSM = func(shardID uint64, replicaID uint64) StateMachinePersistent {
	var id = fmt.Sprintf(`%d-%d`, shardID, replicaID)
	return &mockStateMachinePersistent{
		mockOpen: func(stopc <-chan struct{}) (index uint64, err error) {
			return idx[id], nil
		},
		mockUpdate: func(e []Entry) []Entry {
			for i := range e {
				e[i].Result.Value = uint64(len(e[i].Cmd))
				mutex.Lock()
				idx[id] = e[i].Index
				mutex.Unlock()
			}
			return e
		},
		mockQuery: func(ctx context.Context, data []byte) *Result {
			return &Result{Value: uint64(len(data))}
		},
		mockWatch: func(ctx context.Context, data []byte, results chan<- *Result) {
			for i := 0; i < len(data); i++ {
				results <- &Result{Value: uint64(i)}
			}
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
