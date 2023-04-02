package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type handler struct {
	ctrl *controller
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer w.Write([]byte("\n"))
	var err error
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	if r.Method == "GET" {
		query := kvQuery{
			Op:  queryOpRead,
			Key: r.URL.Path,
		}
		code, data, err := h.ctrl.getClient(r.FormValue("local") != "true", false).Query(ctx, h.ctrl.shard.ID, query.MustMarshalBinary(), r.FormValue("stale") == "true")
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
		code, data, err := h.ctrl.getClient(false, true).Apply(ctx, h.ctrl.shard.ID, cmd.MustMarshalBinary())
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		if code == ResultCodeFailure {
			w.WriteHeader(400)
			w.Write(data)
			return
		}
		if code == ResultCodeVersionMismatch {
			var record kvRecord
			json.Unmarshal(data, &record)
			w.WriteHeader(409)
			w.Write([]byte(fmt.Sprintf("Version mismatch (%d != %d)", cmd.Record.Ver, record.Ver)))
			return
		}
		w.WriteHeader(200)
		w.Write(data)
	} else {
		w.WriteHeader(405)
		w.Write([]byte("Method not supported"))
	}
}
