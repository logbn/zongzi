package zongzi

import (
	"context"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/statemachine"
)

// StateMachinePersistentFactory is a function that returns a StateMachinePersistent
type StateMachinePersistentFactory = func(shardID uint64, replicaID uint64) StateMachinePersistent

func stateMachinePersistentFactoryShim(fn StateMachinePersistentFactory) statemachine.CreateOnDiskStateMachineFunc {
	return func(shardID uint64, replicaID uint64) statemachine.IOnDiskStateMachine {
		return &stateMachinePersistentShim{fn(shardID, replicaID)}
	}
}

// StateMachinePersistent is a StateMachine where the state is persisted to a medium (such as disk) that can survive
// restart. During compaction, calls to Snapshot are replaced with calls to Sync which effectively flushes state to
// the persistent medium. SaveSnapshot and RecoverFromSnapshot are used to replicate full on-disk state to new replicas.
type StateMachinePersistent interface {
	Open(stopc <-chan struct{}) (index uint64, err error)
	Update(entries []Entry) []Entry
	Query(ctx context.Context, query []byte) *Result
	Watch(ctx context.Context, query []byte, result chan<- *Result)
	PrepareSnapshot() (cursor any, err error)
	SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error
	RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error
	Sync() error
	Close() error
}

var _ statemachine.IOnDiskStateMachine = (*stateMachinePersistentShim)(nil)

type stateMachinePersistentShim struct {
	sm StateMachinePersistent
}

func (shim *stateMachinePersistentShim) Open(stopc <-chan struct{}) (index uint64, err error) {
	return shim.sm.Open(stopc)
}

func (shim *stateMachinePersistentShim) Update(entries []Entry) (responses []Entry, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(`%v`, r)
		}
	}()
	responses = shim.sm.Update(entries)
	return
}

func (shim *stateMachinePersistentShim) Lookup(query any) (res any, err error) {
	if q, ok := query.(*lookupQuery); ok {
		res = shim.sm.Query(q.ctx, q.data)
		return
	}
	if q, ok := query.(*watchQuery); ok {
		shim.sm.Watch(q.ctx, q.data, q.result)
		return
	}
	return
}

func (shim *stateMachinePersistentShim) PrepareSnapshot() (cursor any, err error) {
	return shim.sm.PrepareSnapshot()
}

func (shim *stateMachinePersistentShim) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error {
	return shim.sm.SaveSnapshot(cursor, w, close)
}

func (shim *stateMachinePersistentShim) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error {
	return shim.sm.RecoverFromSnapshot(r, close)
}

func (shim *stateMachinePersistentShim) Sync() error {
	return shim.sm.Sync()
}

func (shim *stateMachinePersistentShim) Close() error {
	return shim.sm.Close()
}
