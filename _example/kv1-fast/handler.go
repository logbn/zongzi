package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/valyala/fasthttp"

	"github.com/logbn/zongzi"
)

func hash(b []byte) uint32 {
	hash := fnv.New32a()
	hash.Write(b)
	return hash.Sum32()
}

func handler(clients []zongzi.ShardClient) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		defer ctx.Write([]byte("\n"))
		var err error
		var shard = int(hash(ctx.URI().Path()) % uint32(len(clients)))
		if ctx.IsGet() {
			query := kvQuery{
				Op:  queryOpRead,
				Key: string(ctx.URI().Path()),
			}
			code, data, err := clients[shard].Read(ctx, query.MustMarshalBinary(), string(ctx.FormValue("stale")) == "true")
			if err != nil {
				ctx.SetStatusCode(500)
				ctx.Write([]byte(err.Error()))
				return
			}
			if code == ResultCodeNotFound {
				ctx.SetStatusCode(404)
				ctx.Write([]byte(`Not found`))
				return
			}
			ctx.SetStatusCode(200)
			ctx.Write(data)
		} else if ctx.IsPut() {
			var ver int
			if len(string(ctx.FormValue("ver"))) > 0 {
				ver, err = strconv.Atoi(string(ctx.FormValue("ver")))
				if err != nil {
					ctx.SetStatusCode(400)
					ctx.Write([]byte("Version must be uint64"))
					return
				}
			}
			var cmd = kvCmd{
				Op:  cmdOpSet,
				Key: string(ctx.URI().Path()),
				Record: kvRecord{
					Ver: uint64(ver),
					Val: string(ctx.FormValue("val")),
				},
			}
			var code uint64
			var data []byte
			if string(ctx.FormValue("stale")) == "true" {
				err = clients[shard].Commit(ctx, cmd.MustMarshalBinary())
			} else {
				code, data, err = clients[shard].Apply(ctx, cmd.MustMarshalBinary())
				if code == ResultCodeFailure {
					ctx.SetStatusCode(400)
					ctx.Write(data)
					return
				}
				if code == ResultCodeVersionMismatch {
					var record kvRecord
					json.Unmarshal(data, &record)
					ctx.SetStatusCode(409)
					ctx.Write([]byte(fmt.Sprintf("Version mismatch (%d != %d)", cmd.Record.Ver, record.Ver)))
					return
				}
			}
			if err != nil {
				ctx.SetStatusCode(500)
				ctx.Write([]byte(err.Error()))
				return
			}
			ctx.SetStatusCode(200)
			ctx.Write(data)
		} else {
			ctx.SetStatusCode(405)
			ctx.Write([]byte("Method not supported"))
		}
	}
}
