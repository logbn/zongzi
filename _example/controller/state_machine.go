package main

import (
	"fmt"
	"io"

	"github.com/logbn/zongzi"
)

const (
	StateMachineUri     = "zongzi://github.com/logbn/zongzi-examples/controller/default"
	StateMachineVersion = "v0.0.1"
)

func StateMachineFactory() zongzi.StateMachineFactory {
	return zongzi.StateMachineFactory(func(shardID, replicaID uint64) zongzi.StateMachine {
		return &stateMachine{
			shardID:   shardID,
			replicaID: replicaID,
		}
	})
}

type stateMachine struct {
	zongzi.StateMachine

	replicaID uint64
	shardID   uint64
}

func (fsm *stateMachine) Update(entries []zongzi.Entry) []zongzi.Entry {
	for _, entry := range entries {
		entry.Result.Value = 1
		entry.Result.Data = []byte(fmt.Sprintf("%s [%d:%d]", StateMachineUri, fsm.shardID, fsm.replicaID))
	}
	return entries
}

func (fsm *stateMachine) Lookup(entry zongzi.Entry) zongzi.Entry {
	entry.Result.Value = 1
	entry.Result.Data = []byte(fmt.Sprintf("%s [%d:%d]", StateMachineUri, fsm.shardID, fsm.replicaID))
	return entry
}

func (fsm *stateMachine) SaveSnapshot(cursor any, w io.Writer, close <-chan struct{}) (err error) {
	w.Write([]byte(fmt.Sprintf("%s %d", StateMachineUri, fsm.shardID)))
	return
}

func (fsm *stateMachine) RecoverFromSnapshot(r io.Reader, close <-chan struct{}) (err error) {
	_, err = io.ReadAll(r)
	return
}

func (fsm *stateMachine) Close() (err error)                               { return }
func (fsm *stateMachine) Open(stopc <-chan struct{}) (i uint64, err error) { return }
func (fsm *stateMachine) PrepareSnapshot() (cursor any, err error)         { return }
func (fsm *stateMachine) Sync() (err error)                                { return }
