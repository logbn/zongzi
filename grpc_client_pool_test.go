package zongzi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpcClientPool(t *testing.T) {
	t.Run(`success`, func(t *testing.T) {
		p := newGrpcClientPool(1, []string{})
		c := p.get("127.0.0.1:9000")
		require.NotNil(t, c)
		_, ok := c.(*grpcClientErr)
		assert.False(t, ok)
	})
	t.Run(`err`, func(t *testing.T) {
		p := newGrpcClientPool(1, []string{})
		c := p.get("test|__)Q(@101::::''2")
		require.NotNil(t, c)
		_, ok := c.(*grpcClientErr)
		assert.False(t, ok)
	})
}
