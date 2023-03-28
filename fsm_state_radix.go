package zongzi

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/go-memdb"
)

type State interface {
	metaSet(key string, val uint64)
	metaSetIndex(index uint64)
	metaGet(key string) (val uint64)
	hostPut(h Host)
	hostDelete(h Host)
	hostTouch(id string, index uint64)
	shardPut(s Shard)
	shardDelete(s Shard)
	shardTouch(id, index uint64)
	shardIncr() uint64
	replicaPut(r Replica)
	replicaDelete(r Replica)
	replicaTouch(id, index uint64)
	replicaIncr() uint64

	withTxn(write bool) State
	commit()
	rollback()
	recover(r io.Reader) error

	Index() uint64
	HostGet(id string) (h Host, ok bool)
	HostIterate(fn func(h Host) bool)
	ShardGet(id uint64) (s Shard, ok bool)
	ShardIterate(fn func(s Shard) bool)
	ReplicaGet(id uint64) (r Replica, ok bool)
	ReplicaIterate(fn func(r Replica) bool)
	ReplicaIterateByHostID(hostID string, fn func(r Replica) bool)
	ReplicaIterateByShardID(shardID uint64, fn func(r Replica) bool)
	Save(w io.Writer) error
}

type fsmStateRadix struct {
	db  *memdb.MemDB
	txn *memdb.Txn
}

func newFsmStateRadix() *fsmStateRadix {
	db, err := memdb.NewMemDB(&memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"meta": &memdb.TableSchema{
				Name: "meta",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Key"},
					},
				},
			},
			"host": &memdb.TableSchema{
				Name: "host",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
				},
			},
			"shard": &memdb.TableSchema{
				Name: "shard",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.UintFieldIndex{Field: "ID"},
					},
				},
			},
			"replica": &memdb.TableSchema{
				Name: "replica",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.UintFieldIndex{Field: "ID"},
					},
					"HostID": &memdb.IndexSchema{
						Name:    "HostID",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "HostID"},
					},
					"ShardID": &memdb.IndexSchema{
						Name:    "ShardID",
						Unique:  false,
						Indexer: &memdb.UintFieldIndex{Field: "ShardID"},
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return &fsmStateRadix{
		db: db,
	}
}

type metaValue struct {
	Key string
	Val uint64
}

func (fsm *fsmStateRadix) withTxn(write bool) State {
	return &fsmStateRadix{fsm.db, fsm.db.Txn(write)}
}

func (fsm *fsmStateRadix) commit() {
	fsm.txn.Commit()
}

func (fsm *fsmStateRadix) rollback() {
	fsm.txn.Abort()
}

func (fsm *fsmStateRadix) metaSet(key string, val uint64) {
	fsm.txn.Insert(`meta`, metaValue{key, val})
}

func (fsm *fsmStateRadix) metaSetIndex(val uint64) {
	fsm.txn.Insert(`meta`, metaValue{`index`, val})
}

func (fsm *fsmStateRadix) metaGet(key string) (val uint64) {
	res, _ := fsm.txn.First(`meta`, `id`, key)
	if res != nil {
		val = res.(metaValue).Val
	}
	return
}

func (fsm *fsmStateRadix) Index() (val uint64) {
	res, _ := fsm.txn.First(`meta`, `id`, `index`)
	if res != nil {
		val = res.(metaValue).Val
	}
	return
}

func (fsm *fsmStateRadix) HostGet(id string) (h Host, ok bool) {
	res, _ := fsm.txn.First(`host`, `id`, id)
	if res != nil {
		h = res.(Host)
		ok = true
	}
	return
}

func (fsm *fsmStateRadix) hostPut(h Host) {
	fsm.txn.Insert(`host`, h)
}

func (fsm *fsmStateRadix) hostDelete(h Host) {
	fsm.txn.Delete(`host`, h)
}

func (fsm *fsmStateRadix) hostTouch(id string, index uint64) {
	if h, ok := fsm.HostGet(id); ok {
		h.Updated = index
		fsm.hostPut(h)
	}
}

func (fsm *fsmStateRadix) HostIterate(fn func(h Host) bool) {
	iter, _ := fsm.txn.Get(`host`, `id_prefix`, "")
	for {
		res := iter.Next()
		if res == nil || !fn(res.(Host)) {
			break
		}
	}
}

func (fsm *fsmStateRadix) ShardGet(id uint64) (s Shard, ok bool) {
	res, _ := fsm.txn.First(`shard`, `id`, id)
	if res != nil {
		s = res.(Shard)
		ok = true
	}
	return
}

func (fsm *fsmStateRadix) shardPut(s Shard) {
	fsm.txn.Insert(`shard`, s)

}

func (fsm *fsmStateRadix) shardDelete(s Shard) {
	fsm.txn.Delete(`shard`, s)
}

func (fsm *fsmStateRadix) shardTouch(id, index uint64) {
	if s, ok := fsm.ShardGet(id); ok {
		s.Updated = index
		fsm.shardPut(s)
	}
}

func (fsm *fsmStateRadix) shardIncr() uint64 {
	val := fsm.metaGet(`indexShard`)
	val++
	fsm.txn.Insert(`meta`, metaValue{`indexShard`, val})
	return val
}

func (fsm *fsmStateRadix) ShardIterate(fn func(s Shard) bool) {
	iter, err := fsm.txn.LowerBound(`shard`, `id`, uint64(0))
	if err != nil {
		panic(err)
	}
	for {
		res := iter.Next()
		if res == nil || !fn(res.(Shard)) {
			break
		}
	}
}

func (fsm *fsmStateRadix) ReplicaGet(id uint64) (r Replica, ok bool) {
	res, _ := fsm.txn.First(`replica`, `id`, id)
	if res != nil {
		r = res.(Replica)
		ok = true
	}
	return
}

func (fsm *fsmStateRadix) replicaPut(r Replica) {
	fsm.txn.Insert(`replica`, r)
}

func (fsm *fsmStateRadix) replicaDelete(r Replica) {
	fsm.txn.Delete(`replica`, r)
}

func (fsm *fsmStateRadix) replicaTouch(id, index uint64) {
	if r, ok := fsm.ReplicaGet(id); ok {
		r.Updated = index
		fsm.replicaPut(r)
	}
}

func (fsm *fsmStateRadix) replicaIncr() uint64 {
	val := fsm.metaGet(`indexReplica`)
	val++
	fsm.txn.Insert(`meta`, metaValue{`indexReplica`, val})
	return val
}

func (fsm *fsmStateRadix) ReplicaIterate(fn func(r Replica) bool) {
	iter, err := fsm.txn.LowerBound(`replica`, `id`, uint64(0))
	if err != nil {
		panic(err)
	}
	for {
		res := iter.Next()
		if res == nil || !fn(res.(Replica)) {
			break
		}
	}
}

func (fsm *fsmStateRadix) ReplicaIterateByShardID(shardID uint64, fn func(r Replica) bool) {
	iter, err := fsm.txn.Get(`replica`, `ShardID`, shardID)
	if err != nil {
		panic(err)
	}
	for {
		res := iter.Next()
		if res == nil || !fn(res.(Replica)) {
			break
		}
	}
}

func (fsm *fsmStateRadix) ReplicaIterateByHostID(hostID string, fn func(r Replica) bool) {
	iter, err := fsm.txn.Get(`replica`, `HostID`, hostID)
	if err != nil {
		panic(err)
	}
	for {
		res := iter.Next()
		if res == nil || !fn(res.(Replica)) {
			break
		}
	}
}

type fsmStateMetaHeader struct {
	Index        uint64 `json:"index"`
	IndexShard   uint64 `json:"indexShard"`
	IndexReplica uint64 `json:"indexReplica"`
	CountHost    uint64 `json:"countHost"`
	CountShard   uint64 `json:"countShard"`
	CountReplica uint64 `json:"countReplica"`
}

func (f *fsmStateRadix) Save(w io.Writer) error {
	fsm := f.withTxn(false)
	// TODO - Maintain counts in meta so we don't have to recompute on every snapshot
	var hostCount uint64
	fsm.HostIterate(func(Host) bool {
		hostCount++
		return true
	})
	var shardCount uint64
	fsm.ShardIterate(func(Shard) bool {
		shardCount++
		return true
	})
	var replicaCount uint64
	fsm.ReplicaIterate(func(Replica) bool {
		replicaCount++
		return true
	})
	b, _ := json.Marshal(fsmStateMetaHeader{
		Index:        fsm.metaGet(`index`),
		IndexShard:   fsm.metaGet(`indexShard`),
		IndexReplica: fsm.metaGet(`indexReplica`),
		CountHost:    hostCount,
		CountShard:   shardCount,
		CountReplica: replicaCount,
	})
	w.Write(append(b, '\n'))
	fsm.HostIterate(func(h Host) bool {
		b, _ := json.Marshal(h)
		w.Write(append(b, '\n'))
		return true
	})
	fsm.ShardIterate(func(s Shard) bool {
		b, _ := json.Marshal(s)
		w.Write(append(b, '\n'))
		return true
	})
	fsm.ReplicaIterate(func(r Replica) bool {
		b, _ := json.Marshal(r)
		w.Write(append(b, '\n'))
		return true
	})
	return nil
}

func (f *fsmStateRadix) recover(r io.Reader) (err error) {
	fsm := f.withTxn(true)
	defer func() {
		if err == nil {
			fsm.commit()
		} else {
			fsm.rollback()
		}
	}()
	decoder := json.NewDecoder(r)
	var header fsmStateMetaHeader
	if err := decoder.Decode(&header); err != nil {
		return err
	}
	fsm.metaSet(`index`, header.Index)
	fsm.metaSet(`indexShard`, header.IndexShard)
	fsm.metaSet(`indexReplica`, header.IndexReplica)
	var i uint64
	for i = 0; i < header.CountHost; i++ {
		var h Host
		if err = decoder.Decode(&h); err != nil {
			return fmt.Errorf("parse host: %w", err)
		}
		fsm.hostPut(h)
	}
	for i = 0; i < header.CountShard; i++ {
		var s Shard
		if err = decoder.Decode(&s); err != nil {
			return fmt.Errorf("parse shard: %w", err)
		}
		fsm.shardPut(s)
	}
	for i = 0; i < header.CountReplica; i++ {
		var r Replica
		if err = decoder.Decode(&r); err != nil {
			return fmt.Errorf("parse replica: %w", err)
		}
		fsm.replicaPut(r)
	}
	return nil
}
