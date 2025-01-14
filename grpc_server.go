package zongzi

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/logbn/zongzi/internal"
)

type grpcServer struct {
	internal.UnimplementedInternalServer

	agent      *Agent
	server     *grpc.Server
	listenAddr string
	serverOpts []grpc.ServerOption
}

func newGrpcServer(listenAddr string, opts ...grpc.ServerOption) *grpcServer {
	return &grpcServer{
		listenAddr: listenAddr,
		serverOpts: opts,
	}
}

func (s *grpcServer) Ping(ctx context.Context, req *internal.PingRequest) (res *internal.PingResponse, err error) {
	return &internal.PingResponse{}, nil
}

func (s *grpcServer) Probe(ctx context.Context, req *internal.ProbeRequest) (res *internal.ProbeResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Probe: %#v`, req)
	return &internal.ProbeResponse{
		GossipAdvertiseAddress: s.agent.hostConfig.Gossip.AdvertiseAddress,
	}, nil
}

func (s *grpcServer) Info(ctx context.Context, req *internal.InfoRequest) (res *internal.InfoResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Info: %#v`, req)
	return &internal.InfoResponse{
		HostId:    s.agent.hostID(),
		ReplicaId: s.agent.replicaConfig.ReplicaID,
	}, nil
}

func (s *grpcServer) Members(ctx context.Context, req *internal.MembersRequest) (res *internal.MembersResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Members: %#v`, req)
	return &internal.MembersResponse{
		Members: s.agent.members,
	}, nil
}

func (s *grpcServer) Join(ctx context.Context, req *internal.JoinRequest) (res *internal.JoinResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Join: %#v`, req)
	res = &internal.JoinResponse{}
	if s.agent.Status() != AgentStatus_Ready {
		err = ErrAgentNotReady
		return
	}
	res.Value, err = s.agent.joinPrimeReplica(req.HostId, s.agent.replicaConfig.ShardID, req.IsNonVoting)
	return
}

func (s *grpcServer) Add(ctx context.Context, req *internal.AddRequest) (res *internal.AddResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Join: %#v`, req)
	res = &internal.AddResponse{}
	if s.agent.Status() != AgentStatus_Ready {
		err = ErrAgentNotReady
		return
	}
	res.Value, err = s.agent.joinShardReplica(req.HostId, req.ShardId, req.ReplicaId, req.IsNonVoting)
	return
}

var emptyCommitResponse = &internal.CommitResponse{}

func (s *grpcServer) Commit(ctx context.Context, req *internal.CommitRequest) (res *internal.CommitResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Propose: %#v`, req)
	if !s.agent.hostConfig.NotifyCommit {
		s.agent.log.Warningf(`%v`, ErrNotifyCommitDisabled)
	}
	if s.agent.Status() != AgentStatus_Ready {
		err = ErrAgentNotReady
		return
	}
	rs, err := s.agent.host.Propose(s.agent.host.GetNoOPSession(req.ShardId), req.Data, raftTimeout)
	if err != nil {
		return
	}
	defer rs.Release()
	for {
		select {
		case r := <-rs.ResultC():
			if r.Committed() {
				res = emptyCommitResponse
			} else if r.Aborted() {
				err = ErrAborted
			} else if r.Dropped() {
				err = ErrShardNotReady
			} else if r.Rejected() {
				err = ErrRejected
			} else if r.Terminated() {
				err = ErrShardClosed
			} else if r.Timeout() {
				err = ErrTimeout
			}
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				err = ErrCanceled
			} else if ctx.Err() == context.DeadlineExceeded {
				err = ErrTimeout
			}
		}
		if err != nil || res != nil {
			break
		}
	}
	return
}

func (s *grpcServer) Apply(ctx context.Context, req *internal.ApplyRequest) (res *internal.ApplyResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Propose: %#v`, req)
	if s.agent.Status() != AgentStatus_Ready {
		err = ErrAgentNotReady
		return
	}
	rs, err := s.agent.host.Propose(s.agent.host.GetNoOPSession(req.ShardId), req.Data, raftTimeout)
	if err != nil {
		return
	}
	defer rs.Release()
	for {
		select {
		case r := <-rs.ResultC():
			if r.Completed() {
				res = &internal.ApplyResponse{
					Value: r.GetResult().Value,
					Data:  r.GetResult().Data,
				}
			} else if r.Aborted() {
				err = ErrAborted
			} else if r.Dropped() {
				err = ErrShardNotReady
			} else if r.Rejected() {
				err = ErrRejected
			} else if r.Terminated() {
				err = ErrShardClosed
			} else if r.Timeout() {
				err = ErrTimeout
			}
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				err = ErrCanceled
			} else if ctx.Err() == context.DeadlineExceeded {
				err = ErrTimeout
			}
		}
		if err != nil || res != nil {
			break
		}
	}
	return
}

func (s *grpcServer) Index(ctx context.Context, req *internal.IndexRequest) (res *internal.IndexResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Query: %#v`, req)
	res = &internal.IndexResponse{}
	err = s.agent.index(ctx, req.ShardId)
	return
}

func (s *grpcServer) Read(ctx context.Context, req *internal.ReadRequest) (res *internal.ReadResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Query: %#v`, req)
	res = &internal.ReadResponse{}
	query := getLookupQuery()
	query.ctx = ctx
	query.data = req.Data
	defer query.Release()
	var r any
	if req.Stale {
		r, err = s.agent.host.StaleRead(req.ShardId, query)
	} else {
		ctx, cancel := context.WithTimeout(ctx, raftTimeout)
		defer cancel()
		r, err = s.agent.host.SyncRead(ctx, req.ShardId, query)
	}
	if result, ok := r.(*Result); ok && result != nil {
		res.Value = result.Value
		res.Data = result.Data
		ReleaseResult(result)
	}
	return
}

func (s *grpcServer) Stream(req *internal.StreamRequest, srv internal.Internal_StreamServer) (err error) {
	// s.agent.log.Debugf(`gRPC Create Stream: %#v`, req)
	query := getStreamQuery()
	query.ctx = srv.Context()
	query.data = req.Data
	query.result = make(chan *Result)
	defer query.Release()
	var done = make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case result := <-query.result:
				err := srv.Send(&internal.StreamResponse{
					Value: result.Value,
					Data:  result.Data,
				})
				if err != nil {
					s.agent.log.Errorf(`Error sending stream response: %s`, err.Error())
				}
				ReleaseResult(result)
			case <-done:
				return
			}
		}
	}()
	if req.Stale {
		_, err = s.agent.host.StaleRead(req.ShardId, query)
	} else {
		_, err = s.agent.host.SyncRead(srv.Context(), req.ShardId, query)
	}
	close(done)
	wg.Wait()
	return
}

func (s *grpcServer) Watch(srv internal.Internal_WatchServer) (err error) {
	// s.agent.log.Debugf(`gRPC Create Watch`)
	log.Println("grpcServer.Watch(main): 1")
	query := getWatchQuery()
	query.ctx = srv.Context()
	query.data = make(chan []byte)
	query.result = make(chan *Result)
	defer query.Release()
	log.Println("grpcServer.Watch(main): 2")
	var shardID uint64
	var done = make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("grpcServer.Watch(result): 1")
		for {
			log.Println("grpcServer.Watch(result): 2")
			select {
			case result := <-query.result:
				log.Println("grpcServer.Watch(result): 3a")
				err := srv.Send(&internal.WatchResponse{
					Value: result.Value,
					Data:  result.Data,
				})
				if err != nil {
					s.agent.log.Errorf(`Error forwarding watch response: %s`, err.Error())
				}
				ReleaseResult(result)
			case <-done:
				log.Println("grpcServer.Watch(result): 3b")
				return
			}
			log.Println("grpcServer.Watch(result): 4")
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("grpcServer.Watch(recv): 1")
		for {
			log.Println("grpcServer.Watch(recv): 2")
			req, err := srv.Recv()
			if err != nil {
				if err != io.EOF {
					s.agent.log.Errorf(`Error forwarding watch request: %s`, err.Error())
				}
				break
			}
			if shardID == 0 {
				shardID = req.ShardId
				wg.Add(1)
				go func() {
					defer wg.Done()
					log.Println("grpcServer.Watch(read): 1")
					ctx, cancel := context.WithTimeout(srv.Context(), time.Hour)
					defer cancel()
					if _, err = s.agent.host.SyncRead(ctx, req.ShardId, query); err != nil {
						log.Println("grpcServer.Watch(read): 2a - " + err.Error())
					}
					close(done)
					log.Println("grpcServer.Watch(read): 3")
				}()
			}
			if shardID != req.ShardId {
				log.Println("INVALID SHARD ID")
			}
			log.Println("grpcServer.Watch(recv): 3")
			query.data <- req.Data
			log.Println("grpcServer.Watch(recv): 4")
		}
	}()
	select {
	case <-srv.Context().Done():
	case <-done:
	}
	return
}

func (s *grpcServer) Start(a *Agent) error {
	// a.log.Errorf("Starting gRPC server on %s", s.listenAddr)
	s.agent = a
	lis, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	// https://github.com/grpc/grpc-go/tree/master/examples/features/authentication
	// opts = append(opts, grpc.UnaryInterceptor(ensureValidToken))
	s.server = grpc.NewServer(s.serverOpts...)
	internal.RegisterInternalServer(s.server, s)
	var done = make(chan bool)
	go func() {
		err = s.server.Serve(lis)
		close(done)
	}()
	select {
	case <-a.ctx.Done():
	case <-done:
	}
	return err
}

func (s *grpcServer) Stop() {
	if s.server != nil {
		var ch = make(chan bool)
		go func() {
			s.server.GracefulStop()
			close(ch)
		}()
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			s.server.Stop()
		}
	}
}
