//go:build integration

package zongzi

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent(t *testing.T) {
	t.Run(`ready`, func(t *testing.T) {
		peers := []string{
			`127.0.0.1:17101`,
			`127.0.0.1:17111`,
			`127.0.0.1:17121`,
		}
		agents := make([]*Agent, 3)
		for i := range agents {
			a, err := NewAgent(`test001`, peers,
				WithApiAddress(fmt.Sprintf(`127.0.0.1:171%d1`, i)),
				WithGossipAddress(fmt.Sprintf(`127.0.0.1:171%d2`, i)),
				WithHostConfig(HostConfig{
					WALDir:         fmt.Sprintf("/var/lib/zongzi/agent-%d/wal", i),
					NodeHostDir:    fmt.Sprintf("/var/lib/zongzi/agent-%d/raft", i),
					RaftAddress:    fmt.Sprintf(`127.0.0.1:171%d3`, i),
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
