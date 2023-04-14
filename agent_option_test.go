package zongzi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithApiAddress(t *testing.T) {
	t.Run(`withBind`, func(t *testing.T) {
		a, err := NewAgent(`test001`, nil)
		require.Nil(t, err)
		o := WithApiAddress(`127.0.0.1:80000`)
		o(a)
		assert.Equal(t, `127.0.0.1:80000`, a.advertiseAddress)
		assert.Equal(t, `0.0.0.0:80000`, a.bindAddress)
	})
	t.Run(`withoutBind`, func(t *testing.T) {
		a, err := NewAgent(`test001`, nil)
		require.Nil(t, err)
		o := WithApiAddress(`127.0.0.1:80001`, `127.0.0.1:80002`)
		o(a)
		assert.Equal(t, `127.0.0.1:80001`, a.advertiseAddress)
		assert.Equal(t, `127.0.0.1:80002`, a.bindAddress)
	})
}

func TestWithGossipAddress(t *testing.T) {
	t.Run(`with-bind`, func(t *testing.T) {
		a, err := NewAgent(`test001`, nil)
		require.Nil(t, err)
		o := WithGossipAddress(`127.0.0.1:80000`)
		o(a)
		assert.Equal(t, `127.0.0.1:80000`, a.hostConfig.Gossip.AdvertiseAddress)
		assert.Equal(t, `0.0.0.0:80000`, a.hostConfig.Gossip.BindAddress)
	})
	t.Run(`without-bind`, func(t *testing.T) {
		a, err := NewAgent(`test001`, nil)
		require.Nil(t, err)
		o := WithGossipAddress(`127.0.0.1:80001`, `127.0.0.1:80002`)
		o(a)
		assert.Equal(t, `127.0.0.1:80001`, a.hostConfig.Gossip.AdvertiseAddress)
		assert.Equal(t, `127.0.0.1:80002`, a.hostConfig.Gossip.BindAddress)
	})
}

func TestWithHostConfig(t *testing.T) {
	t.Run(`gossip`, func(t *testing.T) {
		a, err := NewAgent(`test001`, nil)
		require.Nil(t, err)
		a.hostConfig.Gossip.Meta = []byte(`test1`)
		o := WithHostConfig(HostConfig{
			Gossip: GossipConfig{
				AdvertiseAddress: `127.0.0.1:80001`,
				BindAddress:      `127.0.0.1:80002`,
			},
		})
		o(a)
		assert.Equal(t, `127.0.0.1:80001`, string(a.hostConfig.Gossip.AdvertiseAddress))
		assert.Equal(t, `127.0.0.1:80002`, string(a.hostConfig.Gossip.BindAddress))
	})
	t.Run(`no-gossip`, func(t *testing.T) {
		a, err := NewAgent(`test001`, nil)
		require.Nil(t, err)
		aa := a.hostConfig.Gossip.AdvertiseAddress
		ba := a.hostConfig.Gossip.BindAddress
		o := WithHostConfig(HostConfig{})
		o(a)
		assert.Equal(t, aa, string(a.hostConfig.Gossip.AdvertiseAddress))
		assert.Equal(t, ba, string(a.hostConfig.Gossip.BindAddress))
	})
}

func TestWithHostTags(t *testing.T) {
	t.Run(`success`, func(t *testing.T) {
		a, err := NewAgent(`test001`, nil)
		require.Nil(t, err)
		a.hostConfig.Gossip.Meta = []byte(`test1`)
		o := WithHostTags(`test:tag=2`)
		o(a)
		assert.Equal(t, []string{`test:tag=2`}, a.hostTags)
	})
}
