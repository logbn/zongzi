package zongzi

import (
	"os"

	"github.com/benbjohnson/clock"
	"github.com/lni/dragonboat/v4/logger"
)

type AgentMock interface {
	Agent
	GetClock() clock.Clock
}

type agentMock struct {
	agent
}

func NewAgentMock(cfg AgentConfig) (*agentMock, error) {
	clusterName := base36Encode(cfg.NodeHostConfig.DeploymentID)
	return &agentMock{
		agent: agent{
			log:         logger.GetLogger(magicPrefix),
			hostConfig:  cfg.NodeHostConfig,
			clusterName: clusterName,
			client:      newUDPClient(magicPrefix, cfg.NodeHostConfig.RaftAddress, clusterName),
			hostFS:      os.DirFS(cfg.NodeHostConfig.NodeHostDir),
			multicast:   cfg.Multicast,
			clock:       clock.NewMock(),
			peers:       map[string]string{},
			probes:      map[string]bool{},
		},
	}, nil
}

func (a *agentMock) GetClock() clock.Clock {
	return a.clock
}
