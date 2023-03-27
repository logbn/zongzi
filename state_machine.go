package zongzi

import (
	"context"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/statemachine"
)

// StateMachineFactory is a function that returns a StateMachine
type StateMachineFactory = func(shardID uint64, replicaID uint64) StateMachine

func stateMachineFactoryShim(fn StateMachineFactory) statemachine.CreateConcurrentStateMachineFunc {
	return func(shardID uint64, replicaID uint64) statemachine.IConcurrentStateMachine {
		return &stateMachineShim{fn(shardID, replicaID)}
	}
}

// StateMachine is a deterministic finite state machine. Snapshots are requested during log compaction to ensure that
// the in-memory state can be recovered following a restart. If you expect a dataset larger than memory, a persistent
// state machine may be more appropriate.
//
// Lookup may be called concurrently with Update and SaveSnapshot. It is the caller's responsibility to ensure that
// snapshots are generated using snapshot isolation. This can be achieved using Multi Version Concurrency Control
// (MVCC). A simple mutex can also be used if blocking writes during read is acceptable.
type StateMachine interface {
	Update(entries []Entry) []Entry
	Query(ctx context.Context, query []byte) *Result
	Watch(ctx context.Context, query []byte, result chan<- *Result)
	PrepareSnapshot() (cursor any, err error)
	SaveSnapshot(cursor any, w io.Writer, c SnapshotFileCollection, close <-chan struct{}) error
	RecoverFromSnapshot(r io.Reader, f []SnapshotFile, close <-chan struct{}) error
	Close() error
}

var _ statemachine.IConcurrentStateMachine = (*stateMachineShim)(nil)

type stateMachineShim struct {
	sm StateMachine
}

func (shim *stateMachineShim) Update(entries []Entry) (responses []Entry, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(`%v`, r)
		}
	}()
	responses = shim.sm.Update(entries)
	return
}

func (shim *stateMachineShim) Lookup(query any) (res any, err error) {
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

func (shim *stateMachineShim) PrepareSnapshot() (cursor any, err error) {
	return shim.sm.PrepareSnapshot()
}

func (shim *stateMachineShim) SaveSnapshot(cursor any, w io.Writer, c SnapshotFileCollection, close <-chan struct{}) error {
	return shim.sm.SaveSnapshot(cursor, w, c, close)
}

func (shim *stateMachineShim) RecoverFromSnapshot(r io.Reader, f []SnapshotFile, close <-chan struct{}) error {
	return shim.sm.RecoverFromSnapshot(r, f, close)
}

func (shim *stateMachineShim) Close() error {
	return shim.sm.Close()
}
