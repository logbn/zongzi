package zongzi

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"google.golang.org/grpc"

	"github.com/logbn/zongzi/internal"
)

type hostClient struct {
	agent *Agent
	clock clock.Clock
	host  Host
}

func newhostClient(a *Agent, host Host) hostClient {
	return hostClient{
		agent: a,
		clock: clock.New(),
		host:  host,
	}
}

func (c *hostClient) Ping(ctx context.Context) (t time.Duration, err error) {
	if c.host.ID == c.agent.hostID() {
		return
	}
	start := c.clock.Now()
	_, err = c.agent.grpcClientPool.get(c.host.ApiAddress).Ping(ctx, &internal.PingRequest{})
	t = c.clock.Since(start)
	return
}

func (c *hostClient) ReadIndex(ctx context.Context, shardID uint64) (err error) {
	if c.host.ID == c.agent.hostID() {
		return
	}
	_, err = c.agent.grpcClientPool.get(c.host.ApiAddress).Index(ctx, &internal.IndexRequest{ShardId: shardID})
	return
}

func (c *hostClient) Apply(ctx context.Context, shardID uint64, cmd []byte) (value uint64, data []byte, err error) {
	var req = &internal.ApplyRequest{
		ShardId: shardID,
		Data:    cmd,
	}
	var res *internal.ApplyResponse
	if c.host.ID == c.agent.hostID() {
		res, err = c.agent.grpcServer.Apply(ctx, req)
	} else {
		res, err = c.agent.grpcClientPool.get(c.host.ApiAddress).Apply(ctx, req)
	}
	if err != nil {
		return
	}
	value = res.Value
	data = res.Data
	return
}

func (c *hostClient) Commit(ctx context.Context, shardID uint64, cmd []byte) (err error) {
	var req = &internal.CommitRequest{
		ShardId: shardID,
		Data:    cmd,
	}
	if c.host.ID == c.agent.hostID() {
		_, err = c.agent.grpcServer.Commit(ctx, req)
	} else {
		_, err = c.agent.grpcClientPool.get(c.host.ApiAddress).Commit(ctx, req)
	}
	return
}

func (c *hostClient) Read(ctx context.Context, shardID uint64, query []byte, stale bool) (value uint64, data []byte, err error) {
	var req = &internal.ReadRequest{
		ShardId: shardID,
		Stale:   stale,
		Data:    query,
	}
	var res *internal.ReadResponse
	if c.host.ID == c.agent.hostID() {
		res, err = c.agent.grpcServer.Read(ctx, req)
	} else {
		res, err = c.agent.grpcClientPool.get(c.host.ApiAddress).Read(ctx, req)
	}
	if err != nil {
		return
	}
	value = res.Value
	data = res.Data
	return
}

type watchServer struct {
	grpc.ServerStream

	ctx     context.Context
	results chan<- *Result
}

func newWatchServer(ctx context.Context, results chan<- *Result) *watchServer {
	return &watchServer{
		ctx:     ctx,
		results: results,
	}
}

func (s *watchServer) Context() context.Context {
	return s.ctx
}

func (s *watchServer) Send(res *internal.WatchResponse) error {
	s.results <- &Result{
		Value: res.Value,
		Data:  res.Data,
	}
	return nil
}

func (c *hostClient) Watch(ctx context.Context, shardID uint64, query []byte, results chan<- *Result, stale bool) (err error) {
	var req = &internal.WatchRequest{
		ShardId: shardID,
		Stale:   stale,
		Data:    query,
	}
	var client internal.Internal_WatchClient
	if c.host.ID == c.agent.hostID() {
		err = c.agent.grpcServer.Watch(req, newWatchServer(ctx, results))
	} else {
		client, err = c.agent.grpcClientPool.get(c.host.ApiAddress).Watch(ctx, req)
		for {
			res, err := client.Recv()
			if err != nil {
				break
			}
			results <- &Result{
				Value: res.Value,
				Data:  res.Data,
			}
		}
	}
	if err != nil {
		return
	}
	return
}

type streamServer struct {
	grpc.BidiStreamingServer[internal.StreamRequest, internal.StreamResponse]

	ctx     context.Context
	in      <-chan []byte
	init    bool
	out     chan<- *Result
	shardID uint64
	stale   bool
}

func newStreamServer(ctx context.Context, shardID uint64, in <-chan []byte, out chan<- *Result, stale bool) *streamServer {
	return &streamServer{
		ctx:     ctx,
		in:      in,
		out:     out,
		shardID: shardID,
		stale:   stale,
	}
}

func (s *streamServer) Context() context.Context {
	return s.ctx
}

func (s *streamServer) Recv() (req *internal.StreamRequest, err error) {
	if !s.init {
		req = &internal.StreamRequest{
			RequestUnion: &internal.StreamRequest_StreamConnect{
				StreamConnect: &internal.StreamConnect{
					ShardId: s.shardID,
					Stale:   s.stale,
				},
			},
		}
		s.init = true
	} else {
		select {
		case data := <-s.in:
			req = &internal.StreamRequest{
				RequestUnion: &internal.StreamRequest_StreamMessage{
					StreamMessage: &internal.StreamMessage{
						ShardId: s.shardID,
						Data:    data,
					},
				},
			}
		case <-s.ctx.Done():
			err = io.EOF
			return
		}
	}
	return
}

func (s *streamServer) Send(res *internal.StreamResponse) error {
	s.out <- &Result{
		Value: res.Value,
		Data:  res.Data,
	}
	return nil
}

func (c *hostClient) Stream(ctx context.Context, shardID uint64, in <-chan []byte, out chan<- *Result, stale bool) (err error) {
	var wg sync.WaitGroup
	var client internal.Internal_StreamClient
	var done = make(chan bool)
	if c.host.ID == c.agent.hostID() {
		err = c.agent.grpcServer.Stream(newStreamServer(ctx, shardID, in, out, stale))
	} else {
		client, err = c.agent.grpcClientPool.get(c.host.ApiAddress).Stream(ctx)
		wg.Go(func() {
			err = client.Send(&internal.StreamRequest{
				RequestUnion: &internal.StreamRequest_StreamConnect{
					StreamConnect: &internal.StreamConnect{
						ShardId: shardID,
						Stale:   stale,
					},
				},
			})
			if err != nil {
				return
			}
			for {
				select {
				case data := <-in:
					err := client.Send(&internal.StreamRequest{
						RequestUnion: &internal.StreamRequest_StreamMessage{
							StreamMessage: &internal.StreamMessage{
								ShardId: shardID,
								Data:    data,
							},
						},
					})
					if err != nil {
						panic(err)
					}
				case <-done:
					return
				}
			}
		})
		wg.Go(func() {
			for {
				res, err := client.Recv()
				if err != nil {
					if err == io.EOF {
						close(done)
					}
					return
				}
				out <- &Result{
					Value: res.Value,
					Data:  res.Data,
				}
			}
		})
		wg.Wait()
	}
	if err != nil {
		return
	}
	return
}
