package main

import (
	"encoding/json"
)

const (
	stateMachineUri     = "github.com/logbn/zongzi-examples/kv-in-memory"
	stateMachineVersion = "v0.0.1"

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
	Op     uint64 `json:"op"`
	Key    string `json:"key"`
	Record kvRecord
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
