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

func StateMachineFactory() zongzi.SMFactory {
	return zongzi.SMFactory(func(shardID, replicaID uint64) zongzi.IStateMachine {
		return &stateMachine{
			shardID:   shardID,
			replicaID: replicaID,
		}
	})
}

type stateMachine struct {
	replicaID uint64
	shardID   uint64
}

func (fsm *stateMachine) Update(ent zongzi.Entry) (res zongzi.Result, err error) {
	res.Value = 1
	res.Data = []byte(fmt.Sprintf("%s [%d:%d]", shardType, fsm.shardID, fsm.replicaID))
	return
}

func (fsm *stateMachine) Lookup(e any) (val any, err error) {
	val = fmt.Sprintf("%s [%d:%d]", shardType, fsm.shardID, fsm.replicaID)
	return
}

func (fsm *stateMachine) PrepareSnapshot() (ctx any, err error) {
	return
}

func (fsm *stateMachine) SaveSnapshot(w io.Writer, sfc zongzi.ISnapshotFileCollection, stopc <-chan struct{}) (err error) {
	w.Write([]byte(fmt.Sprintf("%s %d", shardType, fsm.shardID)))
	return
}

func (fsm *stateMachine) RecoverFromSnapshot(r io.Reader, sfc []zongzi.SnapshotFile, stopc <-chan struct{}) (err error) {
	_, err = io.ReadAll(r)
	return
}

func (fsm *stateMachine) Close() (err error) {
	return
}
