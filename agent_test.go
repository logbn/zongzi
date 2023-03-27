package zongzi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	t.Run(`success`, func(t *testing.T) {
		for _, testcase := range []string{
			`test001`,
			`0`,
			`1wejoirajen1`,
		} {
			a, err := NewAgent(testcase, nil)
			assert.Nil(t, err)
			assert.NotNil(t, a)
		}
	})
	t.Run(`failure`, func(t *testing.T) {
		for _, testcase := range []string{
			`test-001`,
			`0000000000000000`,
			``,
		} {
			a, err := NewAgent(testcase, nil)
			assert.NotNil(t, err)
			assert.Nil(t, a)
		}
	})
}
