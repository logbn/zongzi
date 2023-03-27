package main

import (
	"context"
	"fmt"
	"io"

	"github.com/logbn/zongzi"
)

const (
	StateMachineUri     = "github.com/logbn/zongzi-examples/controller"
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

func (fsm *stateMachine) Query(ctx context.Context, query []byte) *zongzi.Result {
	result := zongzi.GetResult()
	result.Value = 1
	result.Data = []byte(fmt.Sprintf("%s [%d:%d]", StateMachineUri, fsm.shardID, fsm.replicaID))
	return result
}

func (fsm *stateMachine) PrepareSnapshot() (cursor any, err error) { return }

func (fsm *stateMachine) SaveSnapshot(cursor any, w io.Writer, c zongzi.SnapshotFileCollection, close <-chan struct{}) (err error) {
	w.Write([]byte(fmt.Sprintf("%s %d", StateMachineUri, fsm.shardID)))
	return
}

func (fsm *stateMachine) RecoverFromSnapshot(r io.Reader, f []zongzi.SnapshotFile, close <-chan struct{}) (err error) {
	_, err = io.ReadAll(r)
	return
}

func (fsm *stateMachine) Close() (err error) { return }
