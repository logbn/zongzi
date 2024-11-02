package zongzi

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/go-memdb"
)

// State represents internal state.
// All public methods are read only.
type State struct {
	db  *memdb.MemDB
	txn *memdb.Txn
}

func newFsmStateRadix() *State {
	metaSchema := &memdb.TableSchema{
		Name: "meta",
		Indexes: map[string]*memdb.IndexSchema{
			"id": {
				Name:    "id",
				Unique:  true,
				Indexer: &memdb.StringFieldIndex{Field: "Key"},
			},
		},
	}
	hostSchema := &memdb.TableSchema{
		Name: "host",
		Indexes: map[string]*memdb.IndexSchema{
			"id": {
				Name:    "id",
				Unique:  true,
				Indexer: &memdb.UUIDFieldIndex{Field: "ID"},
			},
			"RaftAddress": {
				Name:         "RaftAddress",
				Unique:       false,
				AllowMissing: true,
				Indexer:      &memdb.StringFieldIndex{Field: "RaftAddress"},
			},
			"Tags": {
				Name:         "Tags",
				Unique:       false,
				AllowMissing: true,
				Indexer:      &memdb.StringMapFieldIndex{Field: "Tags"},
			},
			"Updated": {
				Name:         "Updated",
				Unique:       false,
				AllowMissing: true,
				Indexer:      &memdb.UintFieldIndex{Field: "Updated"},
			},
		},
	}
	shardSchema := &memdb.TableSchema{
		Name: "shard",
		Indexes: map[string]*memdb.IndexSchema{
			"id": {
				Name:    "id",
				Unique:  true,
				Indexer: &memdb.UintFieldIndex{Field: "ID"},
			},
			"Name": {
				Name:         "Name",
				Unique:       true,
				AllowMissing: true,
				Indexer:      &memdb.StringFieldIndex{Field: "Name"},
			},
			"Tags": {
				Name:         "Tags",
				Unique:       false,
				AllowMissing: true,
				Indexer:      &memdb.StringMapFieldIndex{Field: "Tags"},
			},
			"Updated": {
				Name:         "Updated",
				Unique:       false,
				AllowMissing: true,
				Indexer:      &memdb.UintFieldIndex{Field: "Updated"},
			},
		},
	}
	replicaSchema := &memdb.TableSchema{
		Name: "replica",
		Indexes: map[string]*memdb.IndexSchema{
			"id": {
				Name:    "id",
				Unique:  true,
				Indexer: &memdb.UintFieldIndex{Field: "ID"},
			},
			"HostID": {
				Name:    "HostID",
				Unique:  false,
				Indexer: &memdb.UUIDFieldIndex{Field: "HostID"},
			},
			"ShardID": {
				Name:    "ShardID",
				Unique:  false,
				Indexer: &memdb.UintFieldIndex{Field: "ShardID"},
			},
			"Tags": {
				Name:         "Tags",
				Unique:       false,
				AllowMissing: true,
				Indexer:      &memdb.StringMapFieldIndex{Field: "Tags"},
			},
		},
	}
	db, err := memdb.NewMemDB(&memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"meta":    metaSchema,
			"host":    hostSchema,
			"shard":   shardSchema,
			"replica": replicaSchema,
		},
	})
	if err != nil {
		panic(err)
	}
	return &State{
		db: db,
	}
}

type metaValue struct {
	Key  string
	Val  uint64
	Data []byte
}

func (fsm *State) withTxn(write bool) *State {
	return &State{fsm.db, fsm.db.Txn(write)}
}

func (fsm *State) commit() {
	fsm.txn.Commit()
}

func (fsm *State) rollback() {
	fsm.txn.Abort()
}

func (fsm *State) metaSet(key string, val uint64, data []byte) {
	err := fsm.txn.Insert(`meta`, metaValue{key, val, data})
	if err != nil {
		panic(err)
	}
}

func (fsm *State) metaSetIndex(val uint64) {
	err := fsm.txn.Insert(`meta`, metaValue{`index`, val, nil})
	if err != nil {
		panic(err)
	}
}

func (fsm *State) metaGet(key string) (val uint64, data []byte) {
	res, _ := fsm.txn.First(`meta`, `id`, key)
	if res != nil {
		val = res.(metaValue).Val
		data = res.(metaValue).Data
	}
	return
}

func (fsm *State) metaGetVal(key string) (val uint64) {
	val, _ = fsm.metaGet(key)
	return
}

func (fsm *State) metaGetDataString(key string) string {
	_, b := fsm.metaGet(key)
	return string(b)
}

func (fsm *State) Index() (val uint64) {
	return fsm.metaGetVal(`index`)
}

func (fsm *State) setHostID(id string) {
	fsm.metaSet(`hostID`, 0, []byte(id))
}

// HostID returns the ID of the last created Host
func (fsm *State) HostID() string {
	return fsm.metaGetDataString(`hostID`)
}

func (fsm *State) shardIncr() uint64 {
	id := fsm.metaGetVal(`shardID`)
	id++
	fsm.metaSet(`shardID`, id, nil)
	return id
}

// ShardID returns the ID of the last created Shard
func (fsm *State) ShardID() uint64 {
	return fsm.metaGetVal(`shardID`)
}

func (fsm *State) replicaIncr() uint64 {
	id := fsm.metaGetVal(`replicaID`)
	id++
	fsm.metaSet(`replicaID`, id, nil)
	return id
}

// ReplicaID returns the ID of the last created Replica
func (fsm *State) ReplicaID() uint64 {
	return fsm.metaGetVal(`replicaID`)
}

// Host returns the host with the specified ID or ok false if not found.
func (fsm *State) Host(id string) (h Host, ok bool) {
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

// HostIterate executes a callback for every host in the cluster.
// Return true to continue iterating, false to stop.
//
//	var hostCount int
//	agent.Read(func(s *zongzi.State) {
//	    s.HostIterate(func(h Host) bool {
//	        hostCount++
//	        return true
//	    })
//	})
func (fsm *State) HostIterate(fn func(h Host) bool) {
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

// HostIterateByShardType executes a callback for every host in the cluster supporting to the provided shard type,
// ordered by host id ascending. Return true to continue iterating, false to stop.
func (fsm *State) HostIterateByShardType(shardType string, fn func(h Host) bool) {
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

// HostIterateByTag executes a callback for every host in the cluster matching the specified tag,
// ordered by host id ascending. Return true to continue iterating, false to stop.
func (fsm *State) HostIterateByTag(tag string, fn func(h Host) bool) {
	iter, err := fsm.txn.Get(`host`, `Tags`, tag)
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

func (fsm *State) hostPut(h Host) {
	err := fsm.txn.Insert(`host`, h)
	if err != nil {
		panic(err)
	}
}

func (fsm *State) hostDelete(h Host) {
	err := fsm.txn.Delete(`host`, h)
	if err != nil && err != memdb.ErrNotFound {
		panic(err)
	}
}

func (fsm *State) hostTouch(id string, index uint64) {
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

func (fsm *State) hostByRaftAddress(raftAddress string) (h Host, ok bool) {
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

// Shard returns the shard with the specified id or ok false if not found.
func (fsm *State) Shard(id uint64) (s Shard, ok bool) {
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

// ShardFindByName returns the shard with the specified name or ok false if not found.
func (fsm *State) ShardFindByName(name string) (s Shard, ok bool) {
	res, err := fsm.txn.First(`shard`, `Name`, name)
	if err != nil {
		panic(err)
	}
	if res != nil {
		s = res.(Shard)
		ok = true
	}
	return
}

func (fsm *State) shardPut(s Shard) {
	err := fsm.txn.Insert(`shard`, s)
	if err != nil {
		panic(err)
	}
}

func (fsm *State) shardDelete(s Shard) {
	err := fsm.txn.Delete(`shard`, s)
	if err != nil && err != memdb.ErrNotFound {
		panic(err)
	}
}

func (fsm *State) shardTouch(id, index uint64) {
	if s, ok := fsm.Shard(id); ok {
		s.Updated = index
		fsm.shardPut(s)
	}
}

// ShardIterate executes a callback for every shard in the cluster ordered by shard id ascending.
// Return true to continue iterating, false to stop.
func (fsm *State) ShardIterate(fn func(s Shard) bool) {
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

// ShardIterateByTag executes a callback for every shard in the cluster matching the specified tag,
// ordered by shard id ascending. Return true to continue iterating, false to stop.
func (fsm *State) ShardIterateByTag(tag string, fn func(r Shard) bool) {
	iter, err := fsm.txn.Get(`shard`, `Tags`, tag)
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

// ShardIterateUpdatedAfter executes a callback for every shard in the cluster having an updated index
// greater than the supplied index
func (fsm *State) ShardIterateUpdatedAfter(index uint64, fn func(r Shard) bool) {
	iter, err := fsm.txn.LowerBound(`shard`, `Updated`, index+1)
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

// ShardMembers returns a map of shard members (replicaID: hostID)
func (fsm *State) ShardMembers(id uint64) map[uint64]string {
	members := map[uint64]string{}
	fsm.ReplicaIterateByShardID(id, func(r Replica) bool {
		if !r.IsNonVoting {
			members[r.ID] = r.HostID
		}
		return true
	})
	return members
}

// Replica returns the replica with the specified id or ok false if not found.
func (fsm *State) Replica(id uint64) (r Replica, ok bool) {
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

func (fsm *State) replicaPut(r Replica) {
	err := fsm.txn.Insert(`replica`, r)
	if err != nil {
		panic(err)
	}
}

func (fsm *State) replicaDelete(r Replica) {
	err := fsm.txn.Delete(`replica`, r)
	if err != nil && err != memdb.ErrNotFound {
		panic(err)
	}
}

func (fsm *State) replicaTouch(id, index uint64) {
	if r, ok := fsm.Replica(id); ok {
		r.Updated = index
		fsm.replicaPut(r)
	}
}

// ReplicaIterate executes a callback for every replica in the cluster ordered by replica id ascending.
// Return true to continue iterating, false to stop.
func (fsm *State) ReplicaIterate(fn func(r Replica) bool) {
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

// ReplicaIterateByShardID executes a callback for every replica in the cluster belonging to the provided shard id,
// ordered by replica id ascending. Return true to continue iterating, false to stop.
func (fsm *State) ReplicaIterateByShardID(shardID uint64, fn func(r Replica) bool) {
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

// ReplicaIterateByHostID executes a callback for every replica in the cluster belonging to the provided host id,
// ordered by replica id ascending. Return true to continue iterating, false to stop.
func (fsm *State) ReplicaIterateByHostID(hostID string, fn func(r Replica) bool) {
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

// ReplicaIterateByTag executes a callback for every host in the cluster matching the specified tag,
// ordered by host id ascending. Return true to continue iterating, false to stop.
func (fsm *State) ReplicaIterateByTag(tag string, fn func(r Replica) bool) {
	iter, err := fsm.txn.Get(`replica`, `Tags`, tag)
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
	HostID    string `json:"hostID"`
	Hosts     uint64 `json:"hosts"`
	ReplicaID uint64 `json:"replicaID"`
	Replicas  uint64 `json:"replicas"`
	ShardID   uint64 `json:"shardID"`
	Shards    uint64 `json:"shards"`
}

func (fsm *State) Save(w io.Writer) error {
	f := fsm.withTxn(false)
	header := fsmStateMetaHeader{
		Index:     f.metaGetVal(`index`),
		HostID:    f.metaGetDataString(`hostID`),
		ShardID:   f.metaGetVal(`shardID`),
		ReplicaID: f.metaGetVal(`replicaID`),
	}
	f.HostIterate(func(Host) bool {
		header.Hosts++
		return true
	})
	f.ShardIterate(func(Shard) bool {
		header.Shards++
		return true
	})
	f.ReplicaIterate(func(Replica) bool {
		header.Replicas++
		return true
	})
	b, _ := json.Marshal(header)
	w.Write(append(b, '\n'))
	f.HostIterate(func(h Host) bool {
		b, _ := json.Marshal(h)
		w.Write(append(b, '\n'))
		return true
	})
	f.ShardIterate(func(s Shard) bool {
		b, _ := json.Marshal(s)
		w.Write(append(b, '\n'))
		return true
	})
	f.ReplicaIterate(func(r Replica) bool {
		b, _ := json.Marshal(r)
		w.Write(append(b, '\n'))
		return true
	})
	return nil
}

func (fsm *State) recover(r io.Reader) (err error) {
	f := fsm.withTxn(true)
	defer f.commit()
	decoder := json.NewDecoder(r)
	var header fsmStateMetaHeader
	if err := decoder.Decode(&header); err != nil {
		return err
	}
	f.metaSet(`index`, header.Index, nil)
	f.metaSet(`hostID`, 0, []byte(header.HostID))
	f.metaSet(`shardID`, header.ShardID, nil)
	f.metaSet(`replicaID`, header.ReplicaID, nil)
	var i uint64
	for i = 0; i < header.Hosts; i++ {
		var h Host
		if err = decoder.Decode(&h); err != nil {
			return fmt.Errorf("parse host: %w", err)
		}
		f.hostPut(h)
	}
	for i = 0; i < header.Shards; i++ {
		var s Shard
		if err = decoder.Decode(&s); err != nil {
			return fmt.Errorf("parse shard: %w", err)
		}
		f.shardPut(s)
	}
	for i = 0; i < header.Replicas; i++ {
		var r Replica
		if err = decoder.Decode(&r); err != nil {
			return fmt.Errorf("parse replica: %w", err)
		}
		f.replicaPut(r)
	}
	return nil
}
