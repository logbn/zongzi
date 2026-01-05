package zongzi

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMachinePersistentFactoryShim(t *testing.T) {
	t.Run(`success`, func(t *testing.T) {
		shimFunc := stateMachinePersistentFactoryShim(func(shardID uint64, replicaID uint64) StateMachinePersistent {
			return &mockStateMachinePersistent{
				mockUpdate: func(e []Entry) []Entry {
					e[0].Result.Value = shardID + e[0].Index
					return e
				},
			}
		})
		require.NotNil(t, shimFunc)
		shim := shimFunc(1, 1)
		require.NotNil(t, shim)
		res, err := shim.Update([]Entry{{Index: 1}})
		assert.Equal(t, uint64(2), res[0].Result.Value)
		assert.Nil(t, err)
	})
}

func TestStateMachinePersistentShim(t *testing.T) {
	var newShim = func() (*mockStateMachinePersistent, *stateMachinePersistentShim) {
		mock := &mockStateMachinePersistent{}
		return mock, &stateMachinePersistentShim{sm: mock}
	}
	t.Run(`Open`, func(t *testing.T) {
		mock, shim := newShim()
		mock.mockOpen = func(stopc <-chan struct{}) (index uint64, err error) {
			return 1, nil
		}
		index, err := shim.Open(make(chan struct{}))
		assert.Equal(t, uint64(1), index)
		assert.Nil(t, err)
	})
	t.Run(`Update`, func(t *testing.T) {
		mock, shim := newShim()
		mock.mockUpdate = func(e []Entry) []Entry {
			e[0].Result.Value = e[0].Index
			return e
		}
		res, err := shim.Update([]Entry{{Index: 2, Cmd: []byte(``)}})
		assert.Equal(t, uint64(2), res[0].Result.Value)
		assert.Nil(t, err)
	})
}

var _ StateMachinePersistent = (*mockStateMachinePersistent)(nil)

type mockStateMachinePersistent struct {
	mockOpen                func(stopc <-chan struct{}) (index uint64, err error)
	mockUpdate              func(commands []Entry) []Entry
	mockQuery               func(ctx context.Context, data []byte) *Result
	mockWatch               func(ctx context.Context, data []byte, result chan<- *Result)
	mockStream              func(ctx context.Context, in <-chan []byte, out chan<- *Result)
	mockPrepareSnapshot     func() (cursor any, err error)
	mockSaveSnapshot        func(cursor any, w io.Writer, close <-chan struct{}) error
	mockRecoverFromSnapshot func(r io.Reader, close <-chan struct{}) error
	mockSync                func() error
	mockClose               func() error
}

func (shim *mockStateMachinePersistent) Open(stopc <-chan struct{}) (index uint64, err error) {
	return shim.mockOpen(stopc)
}

func (shim *mockStateMachinePersistent) Update(commands []Entry) []Entry {
	return shim.mockUpdate(commands)
}

func (shim *mockStateMachinePersistent) Query(ctx context.Context, data []byte) *Result {
	return shim.mockQuery(ctx, data)
}

func (shim *mockStateMachinePersistent) Watch(ctx context.Context, data []byte, result chan<- *Result) {
	shim.mockWatch(ctx, data, result)
}

func (shim *mockStateMachinePersistent) Stream(ctx context.Context, in <-chan []byte, out chan<- *Result) {
	shim.mockStream(ctx, in, out)
}

func (shim *mockStateMachinePersistent) Sync() error {
	return shim.mockSync()
}

func (shim *mockStateMachinePersistent) PrepareSnapshot() (cursor any, err error) {
	return shim.mockPrepareSnapshot()
}

func (shim *mockStateMachinePersistent) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error {
	return shim.mockSaveSnapshot(cursor, w, close)
}

func (shim *mockStateMachinePersistent) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error {
	return shim.mockRecoverFromSnapshot(r, close)
}

func (shim *mockStateMachinePersistent) Close() error {
	return shim.mockClose()
}
