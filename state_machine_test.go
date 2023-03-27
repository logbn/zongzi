package zongzi

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMachineFactoryShim(t *testing.T) {
	t.Run(`success`, func(t *testing.T) {
		shimFunc := stateMachineFactoryShim(func(shardID uint64, replicaID uint64) StateMachine {
			return &mockStateMachine{
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

func TestStateMachineShim(t *testing.T) {
	var newShim = func() (*mockStateMachine, *stateMachineShim) {
		mock := &mockStateMachine{}
		return mock, &stateMachineShim{sm: mock}
	}
	t.Run(`Update`, func(t *testing.T) {
		t.Run(`success`, func(t *testing.T) {
			mock, shim := newShim()
			mock.mockUpdate = func(e []Entry) []Entry {
				e[0].Result.Value = e[0].Index
				return e
			}
			res, err := shim.Update([]Entry{{Index: 2, Cmd: []byte(``)}})
			assert.Equal(t, uint64(2), res[0].Result.Value)
			assert.Nil(t, err)
		})
	})
	t.Run(`Query`, func(t *testing.T) {
		test := `test`
		t.Run(`success`, func(t *testing.T) {
			mock, shim := newShim()
			mock.mockQuery = func(ctx context.Context, data []byte) *Result {
				return &Result{
					Value: 1,
					Data:  data,
				}
			}
			ctx := context.Background()
			res, err := shim.Lookup(newLookupQuery(ctx, []byte(test)))
			require.Nil(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint64(1), res.(*Result).Value)
			assert.Equal(t, test, string(res.(*Result).Data))
		})
		t.Run(`cancel`, func(t *testing.T) {
			mock, shim := newShim()
			mock.mockQuery = func(ctx context.Context, data []byte) *Result {
				select {
				case <-ctx.Done():
					return &Result{Value: 10}
				default:
				}
				return nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			var res any
			var err error
			wg.Add(1)
			go func() {
				res, err = shim.Lookup(newLookupQuery(ctx, []byte(test)))
				wg.Done()
			}()
			cancel()
			wg.Wait()
			require.Nil(t, err)
			require.NotNil(t, res)
			assert.Equal(t, uint64(10), res.(*Result).Value)
		})
	})
	t.Run(`PrepareSnapshot`, func(t *testing.T) {
		test := `test`
		t.Run(`success`, func(t *testing.T) {
			mock, shim := newShim()
			mock.mockPrepareSnapshot = func() (cursor any, err error) {
				return test, nil
			}
			cursor, err := shim.PrepareSnapshot()
			require.Nil(t, err)
			require.NotNil(t, cursor)
			assert.Equal(t, test, cursor.(string))
		})
	})
	t.Run(`SaveSnapshot`, func(t *testing.T) {
		t.Run(`success`, func(t *testing.T) {
			mock, shim := newShim()
			var called bool
			mock.mockSaveSnapshot = func(cursor any, w io.Writer, c SnapshotFileCollection, close <-chan struct{}) error {
				called = true
				return nil
			}
			w := bytes.NewBuffer(nil)
			c := &mockSnapshotFileCollection{}
			close := make(chan struct{})
			err := shim.SaveSnapshot(1, w, c, close)
			require.Nil(t, err)
			assert.True(t, called)
		})
	})

}

var _ StateMachine = (*mockStateMachine)(nil)

type mockStateMachine struct {
	mockUpdate              func(commands []Entry) []Entry
	mockQuery               func(ctx context.Context, data []byte) *Result
	mockWatch               func(ctx context.Context, data []byte, result chan<- *Result)
	mockPrepareSnapshot     func() (cursor any, err error)
	mockSaveSnapshot        func(cursor any, w io.Writer, c SnapshotFileCollection, close <-chan struct{}) error
	mockRecoverFromSnapshot func(r io.Reader, f []SnapshotFile, close <-chan struct{}) error
	mockClose               func() error
}

func (shim *mockStateMachine) Update(commands []Entry) []Entry {
	return shim.mockUpdate(commands)
}

func (shim *mockStateMachine) Query(ctx context.Context, data []byte) *Result {
	return shim.mockQuery(ctx, data)
}

func (shim *mockStateMachine) Watch(ctx context.Context, data []byte, result chan<- *Result) {
	shim.mockWatch(ctx, data, result)
}

func (shim *mockStateMachine) PrepareSnapshot() (cursor any, err error) {
	return shim.mockPrepareSnapshot()
}

func (shim *mockStateMachine) SaveSnapshot(cursor any, w io.Writer, c SnapshotFileCollection, close <-chan struct{}) error {
	return shim.mockSaveSnapshot(cursor, w, c, close)
}

func (shim *mockStateMachine) RecoverFromSnapshot(r io.Reader, f []SnapshotFile, close <-chan struct{}) error {
	return shim.mockRecoverFromSnapshot(r, f, close)
}

func (shim *mockStateMachine) Close() error {
	return shim.mockClose()
}

type mockSnapshotFileCollection struct{}

func (*mockSnapshotFileCollection) AddFile(fileID uint64, path string, metadata []byte) {
	return
}
