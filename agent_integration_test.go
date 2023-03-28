//go:build !unit

package zongzi

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	run := func(t *testing.T, peers, apiAddr, gossipAddr, raftAddr []string) {
		for i := range apiAddr {
			a, err := NewAgent(`test001`, peers,
				WithApiAddress(apiAddr[i]),
				WithGossipAddress(gossipAddr[i]),
				WithHostConfig(HostConfig{
					WALDir:         fmt.Sprintf(basedir+`/agent-%d/wal`, len(agents)),
					NodeHostDir:    fmt.Sprintf(basedir+`/agent-%d/raft`, len(agents)),
					RaftAddress:    raftAddr[i],
					RTTMillisecond: 10,
				}))
			require.Nil(t, err)
			go func(a *Agent) {
				err := a.Start()
				assert.Nil(t, err, `%+v`, err)
			}(a)
			agents = append(agents, a)
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
		assert.True(t, good, `%+v`, agents)
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
		assert.True(t, good, `%+v`, agents)
	}
	peers := []string{`127.0.0.1:17101`, `127.0.0.1:17111`, `127.0.0.1:17121`}
	t.Run(`start`, func(t *testing.T) {
		run(t, peers,
			[]string{`127.0.0.1:17101`, `127.0.0.1:17111`, `127.0.0.1:17121`},
			[]string{`127.0.0.1:17102`, `127.0.0.1:17112`, `127.0.0.1:17122`},
			[]string{`127.0.0.1:17103`, `127.0.0.1:17113`, `127.0.0.1:17123`},
		)
	})
	t.Run(`join`, func(t *testing.T) {
		run(t, peers,
			[]string{`127.0.0.1:17131`, `127.0.0.1:17141`, `127.0.0.1:17151`},
			[]string{`127.0.0.1:17132`, `127.0.0.1:17142`, `127.0.0.1:17152`},
			[]string{`127.0.0.1:17133`, `127.0.0.1:17143`, `127.0.0.1:17153`},
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
		assert.True(t, good, `%+v`, replicas)
	})
	// Create shard
	// Add replicas
	// Assert initialized
}
