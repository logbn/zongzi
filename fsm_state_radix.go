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
	Host(id string) (h Host, ok bool)
	HostIterate(fn func(h Host) bool)
	HostIterateByShardType(shardType string, fn func(h Host) bool)
	Shard(id uint64) (s Shard, ok bool)
	ShardIterate(fn func(s Shard) bool)
	ShardMembers(id uint64) map[uint64]string
	Replica(id uint64) (r Replica, ok bool)
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
						Indexer: &memdb.UUIDFieldIndex{Field: "ID"},
					},
					"RaftAddress": &memdb.IndexSchema{
						Name:         "RaftAddress",
						Unique:       false,
						AllowMissing: true,
						Indexer:      &memdb.StringFieldIndex{Field: "RaftAddress"},
					},
					"ShardTypes": &memdb.IndexSchema{
						Name:         "ShardTypes",
						Unique:       false,
						AllowMissing: true,
						Indexer:      &memdb.StringSliceFieldIndex{Field: "ShardTypes"},
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
						Indexer: &memdb.UUIDFieldIndex{Field: "HostID"},
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
	err := fsm.txn.Insert(`meta`, metaValue{key, val})
	if err != nil {
		panic(err)
	}
}

func (fsm *fsmStateRadix) metaSetIndex(val uint64) {
	err := fsm.txn.Insert(`meta`, metaValue{`index`, val})
	if err != nil {
		panic(err)
	}
}

func (fsm *fsmStateRadix) metaGet(key string) (val uint64) {
	res, _ := fsm.txn.First(`meta`, `id`, key)
	if res != nil {
		val = res.(metaValue).Val
	}
	return
}

func (fsm *fsmStateRadix) Index() (val uint64) {
	res, err := fsm.txn.First(`meta`, `id`, `index`)
	if err != nil {
		panic(err)
	}
	if res != nil {
		val = res.(metaValue).Val
	}
	return
}

func (fsm *fsmStateRadix) Host(id string) (h Host, ok bool) {
	res, err := fsm.txn.First(`host`, `id`, id)
	if err != nil {
		panic(err)
	}
	if res != nil {
		h = res.(Host)
		ok = true
	}
	return
}

func (fsm *fsmStateRadix) HostIterateByShardType(shardType string, fn func(h Host) bool) {
	iter, err := fsm.txn.Get(`host`, `ShardTypes`, shardType)
	if err != nil {
		panic(err)
	}
	for {
		res := iter.Next()
		if res == nil || !fn(res.(Host)) {
			break
		}
	}
}

func (fsm *fsmStateRadix) hostPut(h Host) {
	err := fsm.txn.Insert(`host`, h)
	if err != nil {
		panic(err)
	}
}

func (fsm *fsmStateRadix) hostDelete(h Host) {
	err := fsm.txn.Delete(`host`, h)
	if err != nil && err != memdb.ErrNotFound {
		panic(err)
	}
}

func (fsm *fsmStateRadix) hostTouch(id string, index uint64) {
	h, ok := fsm.Host(id)
	if !ok {
		fsm.HostIterate(func(h Host) bool {
			fmt.Println(h.ID, h.Updated)
			return true
		})
		panic(`Host not found "` + id + fmt.Sprintf(`" %#v`, h))
	}
	h.Updated = index
	fsm.hostPut(h)
}

func (fsm *fsmStateRadix) hostByRaftAddress(raftAddress string) (h Host, ok bool) {
	res, err := fsm.txn.First(`host`, `RaftAddress`, raftAddress)
	if err != nil {
		panic(err)
	}
	if res != nil {
		h = res.(Host)
		ok = true
	}
	return
}

func (fsm *fsmStateRadix) HostIterate(fn func(h Host) bool) {
	iter, err := fsm.txn.Get(`host`, `id_prefix`, "")
	if err != nil {
		panic(err)
	}
	for {
		res := iter.Next()
		if res == nil || !fn(res.(Host)) {
			break
		}
	}
}

func (fsm *fsmStateRadix) Shard(id uint64) (s Shard, ok bool) {
	res, err := fsm.txn.First(`shard`, `id`, id)
	if err != nil {
		panic(err)
	}
	if res != nil {
		s = res.(Shard)
		ok = true
	}
	return
}

func (fsm *fsmStateRadix) shardPut(s Shard) {
	err := fsm.txn.Insert(`shard`, s)
	if err != nil {
		panic(err)
	}
}

func (fsm *fsmStateRadix) shardDelete(s Shard) {
	err := fsm.txn.Delete(`shard`, s)
	if err != nil && err != memdb.ErrNotFound {
		panic(err)
	}
}

func (fsm *fsmStateRadix) shardTouch(id, index uint64) {
	if s, ok := fsm.Shard(id); ok {
		s.Updated = index
		fsm.shardPut(s)
	}
}

func (fsm *fsmStateRadix) shardIncr() uint64 {
	val := fsm.metaGet(`shardID`)
	val++
	fsm.txn.Insert(`meta`, metaValue{`shardID`, val})
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

func (fsm *fsmStateRadix) ShardMembers(id uint64) map[uint64]string {
	members := map[uint64]string{}
	fsm.ReplicaIterateByShardID(id, func(r Replica) bool {
		if !r.IsNonVoting {
			members[r.ID] = r.HostID
		}
		return true
	})
	return members
}

func (fsm *fsmStateRadix) Replica(id uint64) (r Replica, ok bool) {
	res, err := fsm.txn.First(`replica`, `id`, id)
	if err != nil {
		panic(err)
	}
	if res != nil {
		r = res.(Replica)
		ok = true
	}
	return
}

func (fsm *fsmStateRadix) replicaPut(r Replica) {
	err := fsm.txn.Insert(`replica`, r)
	if err != nil {
		panic(err)
	}
}

func (fsm *fsmStateRadix) replicaDelete(r Replica) {
	err := fsm.txn.Delete(`replica`, r)
	if err != nil && err != memdb.ErrNotFound {
		panic(err)
	}
}

func (fsm *fsmStateRadix) replicaTouch(id, index uint64) {
	if r, ok := fsm.Replica(id); ok {
		r.Updated = index
		fsm.replicaPut(r)
	}
}

func (fsm *fsmStateRadix) replicaIncr() uint64 {
	val := fsm.metaGet(`replicaID`)
	val++
	fsm.txn.Insert(`meta`, metaValue{`replicaID`, val})
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
	Index     uint64 `json:"index"`
	ShardID   uint64 `json:"shardID"`
	ReplicaID uint64 `json:"replicaID"`
	Hosts     uint64 `json:"hosts"`
	Shards    uint64 `json:"shards"`
	Replicas  uint64 `json:"replicas"`
}

func (f *fsmStateRadix) Save(w io.Writer) error {
	fsm := f.withTxn(false)
	header := fsmStateMetaHeader{
		Index:     fsm.metaGet(`index`),
		ShardID:   fsm.metaGet(`shardID`),
		ReplicaID: fsm.metaGet(`replicaID`),
	}
	fsm.HostIterate(func(Host) bool {
		header.Hosts++
		return true
	})
	fsm.ShardIterate(func(Shard) bool {
		header.Shards++
		return true
	})
	fsm.ReplicaIterate(func(Replica) bool {
		header.Replicas++
		return true
	})
	b, _ := json.Marshal(header)
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
	defer fsm.commit()
	decoder := json.NewDecoder(r)
	var header fsmStateMetaHeader
	if err := decoder.Decode(&header); err != nil {
		return err
	}
	fsm.metaSet(`index`, header.Index)
	fsm.metaSet(`shardID`, header.ShardID)
	fsm.metaSet(`replicaID`, header.ReplicaID)
	var i uint64
	for i = 0; i < header.Hosts; i++ {
		var h Host
		if err = decoder.Decode(&h); err != nil {
			return fmt.Errorf("parse host: %w", err)
		}
		fsm.hostPut(h)
	}
	for i = 0; i < header.Shards; i++ {
		var s Shard
		if err = decoder.Decode(&s); err != nil {
			return fmt.Errorf("parse shard: %w", err)
		}
		fsm.shardPut(s)
	}
	for i = 0; i < header.Replicas; i++ {
		var r Replica
		if err = decoder.Decode(&r); err != nil {
			return fmt.Errorf("parse replica: %w", err)
		}
		fsm.replicaPut(r)
	}
	return nil
}
