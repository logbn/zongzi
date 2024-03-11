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

var (
	uri     = `zongzi://github.com/logbn/zongzi/_examples/kv1-fast`
	factory = func(shardID, replicaID uint64) zongzi.StateMachine {
		return &StateMachine{
			shardID:   shardID,
			replicaID: replicaID,
			data:      map[string]kvRecord{},
		}
	}
)

type StateMachine struct {
	zongzi.StateMachine

	replicaID uint64
	shardID   uint64
	data      map[string]kvRecord
	mutex     sync.RWMutex
}

func (fsm *StateMachine) Update(entries []zongzi.Entry) []zongzi.Entry {
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

func (fsm *StateMachine) Query(ctx context.Context, data []byte) (result *zongzi.Result) {
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

func (fsm *StateMachine) PrepareSnapshot() (cursor any, err error) { return }

func (fsm *StateMachine) SaveSnapshot(cursor any, w io.Writer, c zongzi.SnapshotFileCollection, close <-chan struct{}) (err error) {
	fsm.mutex.RLock()
	defer fsm.mutex.RUnlock()
	b, err := json.Marshal(fsm.data)
	if err == nil {
		_, err = io.Copy(w, bytes.NewReader(b))
	}
	return
}

func (fsm *StateMachine) RecoverFromSnapshot(r io.Reader, f []zongzi.SnapshotFile, close <-chan struct{}) (err error) {
	fsm.mutex.Lock()
	defer fsm.mutex.Unlock()
	data, err := io.ReadAll(r)
	if err != nil {
		return
	}
	log.Printf("Recovering (%d)", len(data))
	return json.Unmarshal(data, &fsm.data)
}

func (fsm *StateMachine) Close() (err error) { return }

const (
	ResultCodeFailure = iota
	ResultCodeSuccess
	ResultCodeVersionMismatch
	ResultCodeInvalidRecord
	ResultCodeNotFound

	cmdOpSet = iota
	cmdOpDel

	queryOpRead = iota
)

type kvQuery struct {
	Op  uint64 `json:"op"`
	Key string `json:"key"`
}

func (q *kvQuery) MustMarshalBinary() []byte {
	b, err := json.Marshal(q)
	if err != nil {
		panic(err)
	}
	return b
}

type kvCmd struct {
	Op     uint64   `json:"op"`
	Key    string   `json:"key"`
	Record kvRecord `json:"rec"`
}

func (c *kvCmd) MustMarshalBinary() []byte {
	b, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return b
}

type kvRecord struct {
	Ver uint64 `json:"ver"`
	Val string `json:"val"`
}
