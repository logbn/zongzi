//go:build !unit

package zongzi

import (
	"fmt"
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
			go func(a *Agent) {
				for i := 0; i < 5; i++ {
					err := a.Start()
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
	// Create shard
	// Add replicas
	// Assert initialized
}
