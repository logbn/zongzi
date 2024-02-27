package zongzi

import (
	"fmt"
	"strings"
)

type AgentOption func(*Agent) error

func WithApiAddress(advertiseAddress string, bindAddress ...string) AgentOption {
	return func(a *Agent) error {
		a.advertiseAddress = advertiseAddress
		if len(bindAddress) > 0 {
			a.bindAddress = bindAddress[0]
		} else {
			a.bindAddress = fmt.Sprintf("0.0.0.0:%s", strings.Split(advertiseAddress, ":")[1])
		}
		return nil
	}
}

func WithGossipAddress(advertiseAddress string, bindAddress ...string) AgentOption {
	return func(a *Agent) error {
		a.hostConfig.Gossip.AdvertiseAddress = advertiseAddress
		if len(bindAddress) > 0 {
			a.hostConfig.Gossip.BindAddress = bindAddress[0]
		} else {
			a.hostConfig.Gossip.BindAddress = fmt.Sprintf("0.0.0.0:%s", strings.Split(advertiseAddress, ":")[1])
		}
		return nil
	}
}

func WithRaftAddress(raftAddress string) AgentOption {
	return func(a *Agent) error {
		a.hostConfig.RaftAddress = raftAddress
		return nil
	}
}

func WithHostConfig(cfg HostConfig) AgentOption {
	return func(a *Agent) error {
		if len(cfg.Gossip.AdvertiseAddress) == 0 && len(a.hostConfig.Gossip.AdvertiseAddress) > 0 {
			cfg.Gossip.AdvertiseAddress = a.hostConfig.Gossip.AdvertiseAddress
		}
		if len(cfg.Gossip.BindAddress) == 0 && len(a.hostConfig.Gossip.BindAddress) > 0 {
			cfg.Gossip.BindAddress = a.hostConfig.Gossip.BindAddress
		}
		if len(cfg.RaftAddress) == 0 && len(a.hostConfig.RaftAddress) > 0 {
			cfg.RaftAddress = a.hostConfig.RaftAddress
		}
		cfg.Expert.LogDBFactory = DefaultHostConfig.Expert.LogDBFactory
		a.hostConfig = cfg
		return nil
	}
}

func WithRaftEventListener(listener RaftEventListener) AgentOption {
	return func(a *Agent) error {
		a.hostConfig.RaftEventListener = listener
		return nil
	}
}

func WithSystemEventListener(listener SystemEventListener) AgentOption {
	return func(a *Agent) error {
		a.hostConfig.SystemEventListener = listener
		return nil
	}
}

func WithHostTags(tags ...string) AgentOption {
	return func(a *Agent) error {
		a.hostTags = tags
		return nil
	}
}

func WithReplicaConfig(cfg ReplicaConfig) AgentOption {
	return func(a *Agent) error {
		a.replicaConfig = cfg
		return nil
	}
}

func WithShardController(c ShardController) AgentOption {
	return func(a *Agent) error {
		a.shardControllerManager.shardController = c
		return nil
	}
}
