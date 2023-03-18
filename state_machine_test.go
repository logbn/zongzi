package zongzi

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateMachineShim(t *testing.T) {
	var newShim = func() (*mockStateMachine, *stateMachineShim) {
		mock := &mockStateMachine{}
		return mock, &stateMachineShim{sm: mock}
	}
	t.Run(`Open`, func(t *testing.T) {
		mock, shim := newShim()
		mock.Open = func(stopc <-chan struct{}) (index uint64, err error) {
			return 1, nil
		}
		index, err := shim.Open(make(chan struct{}))
		assert.Equal(t, 1, index)
		assert.Nil(t, err)
	})
}

var _ StateMachine = (*mockStateMachine)(nil)

type mockStateMachine struct {
	mockOpen                func(stopc <-chan struct{}) (index uint64, err error)
	mockUpdate              func(commands []Entry) []Entry
	mockLookup              func(query Entry) Entry
	mockWatch               func(query Entry, resultChan chan<- Result, close <-chan struct{})
	mockSync                func() error
	mockPrepareSnapshot     func() (cursor any, err error)
	mockSaveSnapshot        func(cursor any, w io.Writer, close <-chan struct{}) error
	mockRecoverFromSnapshot func(r io.Reader, close <-chan struct{}) error
	mockClose               func() error
}

func (shim *mockStateMachine) Open(stopc <-chan struct{}) (index uint64, err error) {
	return shim.mockOpen(stopc)
}

func (shim *mockStateMachine) Update(commands []Entry) []Entry {
	return shim.mockUpdate(commands)
}

func (shim *mockStateMachine) Lookup(query Entry) Entry {
	return shim.mockLookup(query)
}

func (shim *mockStateMachine) Watch(query Entry, resultChan chan<- Result, close <-chan struct{}) {
	return shim.mockWatch(ctx, query, resultChan)
}

func (shim *mockStateMachine) Sync() error {
	return shim.mockSync()
}

func (shim *mockStateMachine) PrepareSnapshot() (cursor any, err error) {
	return shim.mockPrepareSnapshot()
}

func (shim *mockStateMachine) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error {
	return shim.mockSaveSnapshot(cursor, w, close)
}

func (shim *mockStateMachine) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error {
	return shim.mockRecoverFromSnapshot(r, close)
}

func (shim *mockStateMachine) Close() error {
	return shim.mockClose()
}
