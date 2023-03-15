package zongzi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lni/dragonboat/v4/logger"
)

func TestNewClient(t *testing.T) {
	c := NewClient(
		nullLogger{},
		"test-magic-prefix",
		"127.0.0.1:10801",
		"testcluster001",
		[]string{"secret"},
	)
	assert.NotNil(t, c)
}

func TestClient(t *testing.T) {
	var c *client
	t.Run("validate", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			c = &client{}
			res, err := c.Validate("CMD", []string{"test"}, 1)
			assert.Nil(t, err)
			assert.Nil(t, res)
		})
		t.Run("failure", func(t *testing.T) {
			c = &client{}
			res, err := c.Validate("CMD", []string{"test"}, 2)
			require.NotNil(t, err)
			assert.Contains(t, err.Error(), "CMD")
			require.Equal(t, 1, len(res))
			assert.Equal(t, "CMD_INVALID", res[0])
		})
	})
	t.Run("sig", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			c = &client{secrets: []string{"test1"}}
			assert.True(t, c.sigVerify(c.sig("cmd", "a b"), "cmd", "a b"))
			assert.False(t, c.sigVerify(c.sig("cmd", "a b")+"cc", "cmd", "a b"))
			assert.False(t, c.sigVerify(c.sig("cmd", "a b c"), "cmd", "a b"))
		})
		t.Run("empty", func(t *testing.T) {
			c = &client{}
			assert.Equal(t, "-", c.sig("cmd", "a b"))
			assert.True(t, c.sigVerify("-", "cmd", "a b"))
			assert.False(t, c.sigVerify("cc", "cmd", "a b"))
			c = &client{secrets: []string{"test1"}}
			assert.False(t, c.sigVerify("-", "cmd", "a b"))
		})
		t.Run("rotation", func(t *testing.T) {
			c = &client{secrets: []string{"test1"}}
			sig1 := c.sig("cmd", "a b")
			c = &client{secrets: []string{"test2"}}
			sig2 := c.sig("cmd", "a b")
			c = &client{secrets: []string{"test1", "test2"}}
			assert.True(t, c.sigVerify(sig1, "cmd", "a b"))
			assert.True(t, c.sigVerify(sig2, "cmd", "a b"))
		})
	})
}

type nullLogger struct{}

func (nullLogger) SetLevel(logger.LogLevel)                    {}
func (nullLogger) Debugf(format string, args ...interface{})   {}
func (nullLogger) Infof(format string, args ...interface{})    {}
func (nullLogger) Warningf(format string, args ...interface{}) {}
func (nullLogger) Errorf(format string, args ...interface{})   {}
func (nullLogger) Panicf(format string, args ...interface{})   {}
