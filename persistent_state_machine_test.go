package zongzi

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPersistentStateMachineShim(t *testing.T) {
	var newShim = func() (*mockPersistentStateMachine, *persistentStateMachineShim) {
		mock := &mockPersistentStateMachine{}
		return mock, &persistentStateMachineShim{sm: mock}
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

var _ PersistentStateMachine = (*mockPersistentStateMachine)(nil)

type mockPersistentStateMachine struct {
	mockOpen                func(stopc <-chan struct{}) (index uint64, err error)
	mockUpdate              func(commands []Entry) []Entry
	mockLookup              func(ctx context.Context, query []byte) *Result
	mockWatch               func(ctx context.Context, query []byte, result chan<- *Result)
	mockSync                func() error
	mockPrepareSnapshot     func() (cursor any, err error)
	mockSaveSnapshot        func(cursor any, w io.Writer, close <-chan struct{}) error
	mockRecoverFromSnapshot func(r io.Reader, close <-chan struct{}) error
	mockClose               func() error
}

func (shim *mockPersistentStateMachine) Open(stopc <-chan struct{}) (index uint64, err error) {
	return shim.mockOpen(stopc)
}

func (shim *mockPersistentStateMachine) Update(commands []Entry) []Entry {
	return shim.mockUpdate(commands)
}

func (shim *mockPersistentStateMachine) Lookup(ctx context.Context, query []byte) *Result {
	return shim.mockLookup(ctx, query)
}

func (shim *mockPersistentStateMachine) Watch(ctx context.Context, query []byte, result chan<- *Result) {
	shim.mockWatch(ctx, query, result)
}

func (shim *mockPersistentStateMachine) Sync() error {
	return shim.mockSync()
}

func (shim *mockPersistentStateMachine) PrepareSnapshot() (cursor any, err error) {
	return shim.mockPrepareSnapshot()
}

func (shim *mockPersistentStateMachine) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error {
	return shim.mockSaveSnapshot(cursor, w, close)
}

func (shim *mockPersistentStateMachine) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error {
	return shim.mockRecoverFromSnapshot(r, close)
}

func (shim *mockPersistentStateMachine) Close() error {
	return shim.mockClose()
}
