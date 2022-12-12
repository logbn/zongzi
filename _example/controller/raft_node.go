package main

import (
	"fmt"
	"io"

	"github.com/logbn/zongzi"
)

func raftNodeFactory() zongzi.CreateStateMachineFunc {
	return zongzi.CreateStateMachineFunc(func(shardID, replicaID uint64) zongzi.IStateMachine {
		return &raftNode{
			shardID:   shardID,
			replicaID: replicaID,
		}
	})
}

type raftNode struct {
	shardID   uint64
	replicaID uint64
}

func (fsm *raftNode) Update(ent zongzi.Entry) (res zongzi.Result, err error) {
	res.Value = 1
	res.Data = []byte(fmt.Sprintf("%s [%d:%d]", shardType, fsm.shardID, fsm.replicaID))
	return
}

func (fsm *raftNode) Lookup(e any) (val any, err error) {
	val = fmt.Sprintf("%s [%d:%d]", shardType, fsm.shardID, fsm.replicaID)
	return
}

func (fsm *raftNode) PrepareSnapshot() (ctx any, err error) {
	return
}

func (fsm *raftNode) SaveSnapshot(w io.Writer, sfc zongzi.ISnapshotFileCollection, stopc <-chan struct{}) (err error) {
	w.Write([]byte(fmt.Sprintf("%s %d", shardType, fsm.shardID)))
	return
}

func (fsm *raftNode) RecoverFromSnapshot(r io.Reader, sfc []zongzi.SnapshotFile, stopc <-chan struct{}) (err error) {
	_, err = io.ReadAll(r)
	return
}

func (fsm *raftNode) Close() (err error) {
	return
}
