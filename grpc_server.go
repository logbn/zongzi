package zongzi

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/logbn/zongzi/internal"
)

type grpcServer struct {
	internal.UnimplementedZongziServer

	agent      *Agent
	server     *grpc.Server
	listenAddr string
	secrets    []string
}

func newGrpcServer(listenAddr string, secrets []string) *grpcServer {
	return &grpcServer{
		listenAddr: listenAddr,
		secrets:    secrets,
	}
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
		HostId:    s.agent.HostID(),
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
	res.Value, err = s.agent.joinPrimeReplica(req.HostId, s.agent.replicaConfig.ShardID, req.IsNonVoting)
	return
}

func (s *grpcServer) ShardJoin(ctx context.Context, req *internal.ShardJoinRequest) (res *internal.ShardJoinResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Join: %#v`, req)
	res = &internal.ShardJoinResponse{}
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
				return
			} else if r.Aborted() {
				err = ErrAborted
				return
			} else if r.Dropped() {
				err = ErrShardNotReady
				return
			} else if r.Rejected() {
				err = ErrRejected
				return
			} else if r.Terminated() {
				err = ErrShardClosed
				return
			} else if r.Timeout() {
				err = ErrTimeout
				return
			}
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				err = ErrCanceled
				return
			} else if ctx.Err() == context.DeadlineExceeded {
				err = ErrTimeout
				return
			}
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
				return
			} else if r.Aborted() {
				err = ErrAborted
				return
			} else if r.Dropped() {
				err = ErrShardNotReady
				return
			} else if r.Rejected() {
				err = ErrRejected
				return
			} else if r.Terminated() {
				err = ErrShardClosed
				return
			} else if r.Timeout() {
				err = ErrTimeout
				return
			}
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				err = ErrCanceled
				return
			} else if ctx.Err() == context.DeadlineExceeded {
				err = ErrTimeout
				return
			}
		}
	}
	return
}

func (s *grpcServer) Query(ctx context.Context, req *internal.QueryRequest) (res *internal.QueryResponse, err error) {
	// s.agent.log.Debugf(`gRPC Req Query: %#v`, req)
	res = &internal.QueryResponse{}
	query := getLookupQuery()
	query.ctx = ctx
	query.data = req.Data
	defer query.Release()
	var r any
	if req.Stale {
		r, err = s.agent.host.StaleRead(req.ShardId, query)
	} else {
		ctx, _ := context.WithTimeout(ctx, raftTimeout)
		r, err = s.agent.host.SyncRead(ctx, req.ShardId, query)
	}
	if result, ok := r.(*Result); ok && result != nil {
		res.Value = result.Value
		res.Data = result.Data
		releaseResult(result)
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
	var opts []grpc.ServerOption
	// https://github.com/grpc/grpc-go/tree/master/examples/features/authentication
	// opts = append(opts, grpc.UnaryInterceptor(ensureValidToken))
	s.server = grpc.NewServer(opts...)
	internal.RegisterZongziServer(s.server, s)
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
