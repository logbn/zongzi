package zongzi

import (
	"context"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4/statemachine"
)

type StateMachineFactory = func(shardID uint64, replicaID uint64) StateMachine

func stateMachineFactoryShim(fn SMFactory) statemachine.CreateOnDiskStateMachineFunc {
	return statemachine.CreateOnDiskStateMachineFunc(func(shardID uint64, replicaID uint64) statemachine.IOnDiskStateMachine {
		return &stateMachineShim{fn(shardID, replicaID)}
	}
}

type StateMachine interface {
	Open(stopc <-chan struct{}) (index uint64, err error)
	Update(entries []Entry) []Entry
	Lookup(query Entry) Entry
	Watch(query Entry, resultChan chan<- Result, close <-chan struct{})
	Sync() error
	PrepareSnapshot() (cursor any, err error)
	SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error
	RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error
	Close() error
}

var _ statemachine.IOnDiskStateMachine = (*stateMachineShim)(nil)

type stateMachineShim struct {
	sm StateMachine
}

func (shim *stateMachineShim) Open(stopc <-chan struct{}) (index uint64, err error) {
	return shim.sm.Open(stopc)
}

func (shim *stateMachineShim) Update(entries []Entry) (responses []Entry, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(r)
		}
	}()
	responses = shim.sm.Update(entries)
	return
}

func (shim *stateMachineShim) Lookup(query interface{}) (interface{}, error) {
	if q, ok := query.(Entry); ok {
		return shim.sm.Lookup(q), nil
	}
	if w, ok := query.(Watch); ok {
		return shim.sm.Watch(w.ctx, w.query, w.resultChan), nil
	}
	return
}

func (shim *stateMachineShim) Sync() error {
	return shim.sm.Sync()
}

func (shim *stateMachineShim) PrepareSnapshot() (cursor any, err error) {
	return shim.sm.PrepareSnapshot()
}

func (shim *stateMachineShim) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) error {
	return shim.sm.SaveSnapshot(cursor, w, close)
}

func (shim *stateMachineShim) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) error {
	return shim.sm.RecoverFromSnapshot(r, close)
}

func (shim *stateMachineShim) Close() error {
	return shim.sm.Close()
}
