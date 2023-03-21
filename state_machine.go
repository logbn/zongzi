package zongzi

import (
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/statemachine"
)

type StateMachineFactory = func(shardID uint64, replicaID uint64) StateMachine

func stateMachineFactoryShim(fn StateMachineFactory) statemachine.CreateOnDiskStateMachineFunc {
	return statemachine.CreateOnDiskStateMachineFunc(func(shardID uint64, replicaID uint64) statemachine.IOnDiskStateMachine {
		return &stateMachineShim{fn(shardID, replicaID)}
	})
}

type StateMachine interface {
	Close() error
	Lookup(query Entry) Entry
	Open(stopc <-chan struct{}) (index uint64, err error)
	PrepareSnapshot() (cursor any, err error)
	RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error
	SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error
	Sync() error
	Update(entries []Entry) []Entry
	Watch(query Entry, result chan<- Result, close <-chan struct{})
}

var _ statemachine.IOnDiskStateMachine = (*stateMachineShim)(nil)

type stateMachineShim struct {
	sm StateMachine
}

func (shim *stateMachineShim) Close() error {
	return shim.sm.Close()
}

func (shim *stateMachineShim) Lookup(query any) (res any, err error) {
	if q, ok := query.(Entry); ok {
		res = shim.sm.Lookup(q)
		return
	}
	if w, ok := query.(watchQuery); ok {
		shim.sm.Watch(w.query, w.result, w.close)
		return
	}
	return
}

func (shim *stateMachineShim) Open(stopc <-chan struct{}) (index uint64, err error) {
	return shim.sm.Open(stopc)
}

func (shim *stateMachineShim) PrepareSnapshot() (cursor any, err error) {
	return shim.sm.PrepareSnapshot()
}

func (shim *stateMachineShim) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error {
	return shim.sm.RecoverFromSnapshot(r, close)
}

func (shim *stateMachineShim) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error {
	return shim.sm.SaveSnapshot(cursor, w, close)
}

func (shim *stateMachineShim) Sync() error {
	return shim.sm.Sync()
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
