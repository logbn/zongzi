//go:build integration

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
	agents := make([]*Agent, 3)
	t.Run(`start`, func(t *testing.T) {
		os.RemoveAll(basedir)
		var (
			apiAddr    = []string{`127.0.0.1:17101`, `127.0.0.1:17111`, `127.0.0.1:17121`}
			gossipAddr = []string{`127.0.0.1:17102`, `127.0.0.1:17112`, `127.0.0.1:17122`}
			raftAddr   = []string{`127.0.0.1:17103`, `127.0.0.1:17113`, `127.0.0.1:17123`}
		)
		for i := range agents {
			a, err := NewAgent(`test001`, apiAddr,
				WithApiAddress(apiAddr[i]),
				WithGossipAddress(gossipAddr[i]),
				WithHostConfig(HostConfig{
					WALDir:         fmt.Sprintf(basedir+`/agent-%d/wal`, i),
					NodeHostDir:    fmt.Sprintf(basedir+`/agent-%d/raft`, i),
					RaftAddress:    raftAddr[i],
					RTTMillisecond: 100,
				}))
			require.Nil(t, err)
			go func(a *Agent) {
				err := a.Start()
				assert.Nil(t, err)
			}(a)
			defer a.Stop()
			agents[i] = a
		}
		var good bool
		// 10 seconds to start the cluster.
		for i := 0; i < 100; i++ {
			good = true
			for j := range agents {
				good = good && agents[j].GetStatus() == AgentStatus_Ready
			}
			if good {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		assert.True(t, good, `%v %v %v`, agents[0].GetStatus(), agents[1].GetStatus(), agents[2].GetStatus())
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
		assert.True(t, good, `%v %v %v`, agents[0].controller.index, agents[1].controller.index, agents[2].controller.index)
	})
}
