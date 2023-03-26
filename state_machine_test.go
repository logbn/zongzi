package zongzi

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateMachineConcurrentShim(t *testing.T) {
	var newShim = func() (*mockStateMachineConcurrent, *stateMachineConcurrentShim) {
		mock := &mockStateMachineConcurrent{}
		return mock, &stateMachineConcurrentShim{sm: mock}
	}
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

var _ StateMachine = (*mockStateMachineConcurrent)(nil)

type mockStateMachineConcurrent struct {
	mockUpdate              func(commands []Entry) []Entry
	mockLookup              func(ctx context.Context, query []byte) *Result
	mockWatch               func(ctx context.Context, query []byte, result chan<- *Result)
	mockPrepareSnapshot     func() (cursor any, err error)
	mockSaveSnapshot        func(cursor any, w io.Writer, c SnapshotFileCollection, close <-chan struct{}) error
	mockRecoverFromSnapshot func(r io.Reader, f []SnapshotFile, close <-chan struct{}) error
	mockClose               func() error
}

func (shim *mockStateMachineConcurrent) Update(commands []Entry) []Entry {
	return shim.mockUpdate(commands)
}

func (shim *mockStateMachineConcurrent) Lookup(ctx context.Context, query []byte) *Result {
	return shim.mockLookup(ctx, query)
}

func (shim *mockStateMachineConcurrent) Watch(ctx context.Context, query []byte, result chan<- *Result) {
	shim.mockWatch(ctx, query, result)
}

func (shim *mockStateMachineConcurrent) PrepareSnapshot() (cursor any, err error) {
	return shim.mockPrepareSnapshot()
}

func (shim *mockStateMachineConcurrent) SaveSnapshot(cursor any, w io.Writer, c SnapshotFileCollection, close <-chan struct{}) error {
	return shim.mockSaveSnapshot(cursor, w, c, close)
}

func (shim *mockStateMachineConcurrent) RecoverFromSnapshot(r io.Reader, f []SnapshotFile, close <-chan struct{}) error {
	return shim.mockRecoverFromSnapshot(r, f, close)
}

func (shim *mockStateMachineConcurrent) Close() error {
	return shim.mockClose()
}
