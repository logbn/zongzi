package zongzi

import (
	"context"
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

func (s *grpcServer) Ping(ctx context.Context,
	req *internal.PingRequest,
) (resp *internal.PingResponse, err error) {
	return &internal.PingResponse{}, nil
}

func (s *grpcServer) Probe(ctx context.Context,
	req *internal.ProbeRequest,
) (resp *internal.ProbeResponse, err error) {
	return &internal.ProbeResponse{
		GossipAdvertiseAddress: s.agent.hostConfig.Gossip.AdvertiseAddress,
	}, nil
}

func (s *grpcServer) Info(ctx context.Context,
	req *internal.InfoRequest,
) (resp *internal.InfoResponse, err error) {
	return &internal.InfoResponse{
		HostId:    s.agent.hostID(),
		ReplicaId: s.agent.replicaConfig.ReplicaID,
	}, nil
}

func (s *grpcServer) Members(ctx context.Context,
	req *internal.MembersRequest,
) (resp *internal.MembersResponse, err error) {
	return &internal.MembersResponse{
		Members: s.agent.members,
	}, nil
}

func (s *grpcServer) Join(ctx context.Context,
	req *internal.JoinRequest,
) (resp *internal.JoinResponse, err error) {
	resp = &internal.JoinResponse{}
	if s.agent.Status() != AgentStatus_Ready {
		err = ErrAgentNotReady
		return
	}
	resp.Value, err = s.agent.joinPrimeReplica(req.HostId, s.agent.replicaConfig.ShardID, req.IsNonVoting)
	return
}

func (s *grpcServer) Add(ctx context.Context,
	req *internal.AddRequest,
) (resp *internal.AddResponse, err error) {
	resp = &internal.AddResponse{}
	if s.agent.Status() != AgentStatus_Ready {
		err = ErrAgentNotReady
		return
	}
	resp.Value, err = s.agent.joinShardReplica(req.HostId, req.ShardId, req.ReplicaId, req.IsNonVoting)
	return
}

var emptyCommitResponse = &internal.CommitResponse{}

func (s *grpcServer) Commit(ctx context.Context,
	req *internal.CommitRequest,
) (resp *internal.CommitResponse, err error) {
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
				resp = emptyCommitResponse
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
		if err != nil || resp != nil {
			break
		}
	}
	return
}

func (s *grpcServer) Apply(ctx context.Context,
	req *internal.ApplyRequest,
) (resp *internal.ApplyResponse, err error) {
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
				resp = &internal.ApplyResponse{
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
		if err != nil || resp != nil {
			break
		}
	}
	return
}

var emptyIndexResponse = &internal.IndexResponse{}

func (s *grpcServer) Index(ctx context.Context,
	req *internal.IndexRequest,
) (resp *internal.IndexResponse, err error) {
	resp = emptyIndexResponse
	err = s.agent.index(ctx, req.ShardId)
	return
}

func (s *grpcServer) Read(ctx context.Context,
	req *internal.ReadRequest,
) (resp *internal.ReadResponse, err error) {
	resp = &internal.ReadResponse{}
	query := getLookupQuery()
	query.ctx = ctx
	query.data = req.Data
	defer query.Release()
	var r any
	if req.Stale {
		r, err = s.agent.host.StaleRead(req.ShardId, query)
	} else {
		ctx, cancel := context.WithTimeout(ctx, time.Hour<<20) // 1 million hours 🤙
		defer cancel()
		r, err = s.agent.host.SyncRead(ctx, req.ShardId, query)
	}
	if result, ok := r.(*Result); ok && result != nil {
		resp.Value = result.Value
		resp.Data = result.Data
	}
	return
}

func (s *grpcServer) Watch(req *internal.WatchRequest, srv internal.Internal_WatchServer) (err error) {
	done := make(chan bool)
	query := getWatchQuery()
	query.ctx = srv.Context()
	query.data = req.Data
	query.result = make(chan *Result)
	defer query.Release()
	wg := sync.WaitGroup{}
	wg.Go(func() {
		for {
			select {
			case result := <-query.result:
				err := srv.Send(&internal.WatchResponse{
					Value: result.Value,
					Data:  result.Data,
				})
				if err != nil {
					s.agent.log.Errorf(`Error sending watch response: %s`, err.Error())
				}
			case <-done:
				return
			}
		}
	})
	if req.Stale {
		_, err = s.agent.host.StaleRead(req.ShardId, query)
	} else {
		ctx, cancel := context.WithTimeout(srv.Context(), time.Hour<<20) // 1 million hours 🤙
		defer cancel()
		_, err = s.agent.host.SyncRead(ctx, req.ShardId, query)
	}
	close(done)
	wg.Wait()
	return
}

func (s *grpcServer) Stream(srv grpc.BidiStreamingServer[internal.StreamRequest, internal.StreamResponse]) (err error) {
	first, err := srv.Recv()
	if err != nil {
		return
	}
	var req *internal.StreamConnect
	switch ut := first.RequestUnion.(type) {
	case *internal.StreamRequest_StreamConnect:
		req = ut.StreamConnect
	default:
		err = ErrStreamConnectMissing
		return
	}
	done := make(chan bool)
	query := getStreamQuery()
	query.ctx = srv.Context()
	query.in = make(chan []byte)
	query.out = make(chan *Result)
	defer query.Release()
	wg := sync.WaitGroup{}
	wg.Go(func() {
		for {
			select {
			case res := <-query.out:
				err := srv.Send(&internal.StreamResponse{
					Value: res.Value,
					Data:  res.Data,
				})
				if err != nil {
					s.agent.log.Errorf(`Error sending stream response: %s`, err.Error())
				}
			case <-done:
				return
			}
		}
	})
	go func() {
		for {
			req, err := srv.Recv()
			if err != nil {
				return
			}
			switch ut := req.RequestUnion.(type) {
			case *internal.StreamRequest_StreamMessage:
				query.in <- ut.StreamMessage.Data
			default:
				err = ErrStreamConnectDuplicate
				return
			}
		}
	}()
	if req.Stale {
		_, err = s.agent.host.StaleRead(req.ShardId, query)
	} else {
		ctx, cancel := context.WithTimeout(srv.Context(), time.Hour<<20) // 1 million hours 🤙
		defer cancel()
		_, err = s.agent.host.SyncRead(ctx, req.ShardId, query)
	}
	close(done)
	wg.Wait()
	return
}
