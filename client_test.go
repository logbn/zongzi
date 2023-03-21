package zongzi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	a, _ := NewAgent("test", []string{"127.0.0.1:10801"})
	c := newClient(a)
	assert.NotNil(t, c)
}
