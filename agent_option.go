package zongzi

import (
	"fmt"
	"strings"
)

type AgentOption func(*Agent) error

func AgentOptionApiAddress(advertiseAddress string, bindAddress ...string) AgentOption {
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

func AgentOptionGossipAddress(advertiseAddress string, bindAddress ...string) AgentOption {
	return func(a *Agent) error {
		a.configHost.Gossip.AdvertiseAddress = advertiseAddress
		if len(bindAddress) > 0 {
			a.configHost.Gossip.BindAddress = bindAddress[0]
		} else {
			a.configHost.Gossip.BindAddress = fmt.Sprintf("0.0.0.0:%s", strings.Split(advertiseAddress, ":")[1])
		}
		return nil
	}
}

func AgentOptionHostConfig(cfg HostConfig) AgentOption {
	return func(a *Agent) error {
		if len(cfg.Gossip.AdvertiseAddress) == 0 && len(a.configHost.Gossip.AdvertiseAddress) > 0 {
			cfg.Gossip.AdvertiseAddress = a.configHost.Gossip.AdvertiseAddress
		}
		if len(cfg.Gossip.BindAddress) == 0 && len(a.configHost.Gossip.BindAddress) > 0 {
			cfg.Gossip.BindAddress = a.configHost.Gossip.BindAddress
		}
		if len(cfg.Gossip.Meta) == 0 && len(a.configHost.Gossip.Meta) > 0 {
			cfg.Gossip.Meta = a.configHost.Gossip.Meta
		}
		a.configHost = cfg
		return nil
	}
}

func AgentOptionMeta(meta []byte) AgentOption {
	return func(a *Agent) error {
		a.configHost.Gossip.Meta = meta
		return nil
	}
}

func AgentOptionReplicaConfig(cfg ReplicaConfig) AgentOption {
	return func(a *Agent) error {
		a.configPrime = cfg
		return nil
	}
}
