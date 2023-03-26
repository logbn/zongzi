package zongzi

import (
	"context"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/statemachine"
)

// PersistentStateMachineFactory is a function that returns a PersistentStateMachine
type PersistentStateMachineFactory = func(shardID uint64, replicaID uint64) PersistentStateMachine

func persistentStateMachineFactoryShim(fn PersistentStateMachineFactory) statemachine.CreateOnDiskStateMachineFunc {
	return statemachine.CreateOnDiskStateMachineFunc(func(shardID uint64, replicaID uint64) statemachine.IOnDiskStateMachine {
		return &persistentStateMachineShim{fn(shardID, replicaID)}
	})
}

// PersistentStateMachine is a StateMachine where the state is persisted to a medium (such as disk) that can survive
// restart. During compaction, calls to Snapshot are replaced with calls to Sync which effectively flushes state to
// the persistent medium. SaveSnapshot and RecoverFromSnapshot are used to replicate full on-disk state to new replicas.
type PersistentStateMachine interface {
	Open(stopc <-chan struct{}) (index uint64, err error)
	Update(entries []Entry) []Entry
	Lookup(ctx context.Context, query []byte) *Result
	Watch(ctx context.Context, query []byte, result chan<- *Result)
	PrepareSnapshot() (cursor any, err error)
	SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error
	RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error
	Sync() error
	Close() error
}

var _ statemachine.IOnDiskStateMachine = (*persistentStateMachineShim)(nil)

type persistentStateMachineShim struct {
	sm PersistentStateMachine
}

func (shim *persistentStateMachineShim) Open(stopc <-chan struct{}) (index uint64, err error) {
	return shim.sm.Open(stopc)
}

func (shim *persistentStateMachineShim) Update(entries []Entry) (responses []Entry, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(`%v`, r)
		}
	}()
	responses = shim.sm.Update(entries)
	return
}

func (shim *persistentStateMachineShim) Lookup(query any) (res any, err error) {
	if q, ok := query.(lookupQuery); ok {
		res = shim.sm.Lookup(q.ctx, q.data)
		return
	}
	if q, ok := query.(watchQuery); ok {
		shim.sm.Watch(q.ctx, q.data, q.result)
		return
	}
	return
}

func (shim *persistentStateMachineShim) PrepareSnapshot() (cursor any, err error) {
	return shim.sm.PrepareSnapshot()
}

func (shim *persistentStateMachineShim) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error {
	return shim.sm.SaveSnapshot(cursor, w, close)
}

func (shim *persistentStateMachineShim) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error {
	return shim.sm.RecoverFromSnapshot(r, close)
}

func (shim *persistentStateMachineShim) Sync() error {
	return shim.sm.Sync()
}

func (shim *persistentStateMachineShim) Close() error {
	return shim.sm.Close()
}
