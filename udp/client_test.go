package udp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	t.Run("sig", func(t *testing.T) {
		var c *client
		t.Run("success", func(t *testing.T) {
			c = &client{secrets: []string{"test1"}}
			assert.True(t, c.sigVerify(c.sig("cmd", "a b"), "cmd", "a b"))
			assert.False(t, c.sigVerify(c.sig("cmd", "a b")+"cc", "cmd", "a b"))
		})
		t.Run("empty", func(t *testing.T) {
			c = &client{}
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
