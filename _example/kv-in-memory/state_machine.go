package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"sync"

	"github.com/logbn/zongzi"
)

func stateMachineFactory() zongzi.StateMachineFactory {
	return zongzi.StateMachineFactory(func(shardID, replicaID uint64) zongzi.StateMachine {
		return &stateMachine{
			shardID:   shardID,
			replicaID: replicaID,
			data:      map[string]kvRecord{},
		}
	})
}

type stateMachine struct {
	zongzi.StateMachine

	replicaID uint64
	shardID   uint64
	data      map[string]kvRecord
	mutex     sync.RWMutex
}

func (fsm *stateMachine) Update(entries []zongzi.Entry) []zongzi.Entry {
	fsm.mutex.Lock()
	defer fsm.mutex.Unlock()
	for i, entry := range entries {
		var cmd kvCmd
		if err := json.Unmarshal(entry.Cmd, &cmd); err != nil {
			entries[i].Result.Value = ResultCodeInvalidRecord
			continue
		}
		switch cmd.Op {
		case cmdOpSet:
			if old, ok := fsm.data[cmd.Key]; ok {
				// Reject entries with mismatched versions
				if old.Ver != cmd.Record.Ver {
					b, _ := json.Marshal(old)
					entries[i].Result.Value = ResultCodeVersionMismatch
					entries[i].Result.Data = b
					continue
				}
			}
			cmd.Record.Ver = entry.Index
			fsm.data[cmd.Key] = cmd.Record
			b, _ := json.Marshal(cmd.Record)
			entries[i].Result.Value = ResultCodeSuccess
			entries[i].Result.Data = b
		case cmdOpDel:
			if cmd.Record.Ver > 0 {
				if old, ok := fsm.data[cmd.Key]; ok {
					// Reject entries with mismatched versions
					if old.Ver != cmd.Record.Ver {
						b, _ := json.Marshal(old)
						entries[i].Result.Value = ResultCodeVersionMismatch
						entries[i].Result.Data = b
						continue
					}
				}
			}
			entries[i].Result.Value = ResultCodeSuccess
			delete(fsm.data, cmd.Key)
		}
	}
	return entries
}

func (fsm *stateMachine) Query(ctx context.Context, data []byte) (result *zongzi.Result) {
	fsm.mutex.RLock()
	defer fsm.mutex.RUnlock()
	var query kvQuery
	result = zongzi.GetResult()
	if err := json.Unmarshal(data, &query); err != nil {
		result.Value = ResultCodeInvalidRecord
		return
	}
	switch query.Op {
	case queryOpRead:
		if record, ok := fsm.data[query.Key]; ok {
			b, _ := json.Marshal(record)
			result.Value = ResultCodeSuccess
			result.Data = b
		} else {
			result.Value = ResultCodeNotFound
		}
	}
	return
}

func (fsm *stateMachine) PrepareSnapshot() (cursor any, err error) { return }

func (fsm *stateMachine) SaveSnapshot(cursor any, w io.Writer, c zongzi.SnapshotFileCollection, close <-chan struct{}) (err error) {
	fsm.mutex.RLock()
	defer fsm.mutex.RUnlock()
	b, err := json.Marshal(fsm.data)
	if err == nil {
		_, err = io.Copy(w, bytes.NewReader(b))
	}
	return
}

func (fsm *stateMachine) RecoverFromSnapshot(r io.Reader, f []zongzi.SnapshotFile, close <-chan struct{}) (err error) {
	fsm.mutex.Lock()
	defer fsm.mutex.Unlock()
	data, err := io.ReadAll(r)
	if err != nil {
		return
	}
	log.Printf("Recovering (%d)", len(data))
	return json.Unmarshal(data, &fsm.data)
}

func (fsm *stateMachine) Close() (err error) { return }
