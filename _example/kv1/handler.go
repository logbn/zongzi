package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"time"

	"github.com/logbn/zongzi"
)

type handler struct {
	clients []zongzi.ShardClient
}

func hash(b []byte) uint32 {
	hash := fnv.New32a()
	hash.Write(b)
	return hash.Sum32()
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer w.Write([]byte("\n"))
	var err error
	var shard = int(hash([]byte(r.URL.Path)) % uint32(len(h.clients)))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	if r.Method == "GET" {
		query := kvQuery{
			Op:  queryOpRead,
			Key: r.URL.Path,
		}
		code, data, err := h.clients[shard].Read(ctx, query.MustMarshalBinary(), r.FormValue("stale") == "true")
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		if code == ResultCodeNotFound {
			w.WriteHeader(404)
			w.Write([]byte(`Not found`))
			return
		}
		w.WriteHeader(200)
		w.Write(data)
	} else if r.Method == "PUT" {
		var ver int
		if len(r.FormValue("ver")) > 0 {
			ver, err = strconv.Atoi(r.FormValue("ver"))
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte("Version must be uint64"))
				return
			}
		}
		var cmd = kvCmd{
			Op:  cmdOpSet,
			Key: r.URL.Path,
			Record: kvRecord{
				Ver: uint64(ver),
				Val: r.FormValue("val"),
			},
		}
		var code uint64
		var data []byte
		if r.FormValue("stale") == "true" {
			err = h.clients[shard].Commit(ctx, cmd.MustMarshalBinary())
		} else {
			code, data, err = h.clients[shard].Apply(ctx, cmd.MustMarshalBinary())
			if code == ResultCodeFailure {
				w.WriteHeader(400)
				if err != nil {
					w.Write([]byte(err.Error()))
				}
				return
			}
			if code == ResultCodeVersionMismatch {
				var record kvRecord
				json.Unmarshal(data, &record)
				w.WriteHeader(409)
				w.Write([]byte(fmt.Sprintf("Version mismatch (%d != %d)", cmd.Record.Ver, record.Ver)))
				return
			}
		}
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(200)
		w.Write(data)
	} else {
		w.WriteHeader(405)
		w.Write([]byte("Method not supported"))
	}
}
