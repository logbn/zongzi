package zongzi

import (
	"context"
	"net"

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
	return &internal.ProbeResponse{
		GossipAdvertiseAddress: s.agent.hostConfig.Gossip.AdvertiseAddress,
	}, nil
}

func (s *grpcServer) Info(ctx context.Context, req *internal.InfoRequest) (res *internal.InfoResponse, err error) {
	return &internal.InfoResponse{
		HostId:    s.agent.GetHostID(),
		ReplicaId: s.agent.replicaConfig.ReplicaID,
	}, nil
}

func (s *grpcServer) Members(ctx context.Context, req *internal.MembersRequest) (res *internal.MembersResponse, err error) {
	return &internal.MembersResponse{
		Members: s.agent.members,
	}, nil
}

func (s *grpcServer) Join(ctx context.Context, req *internal.JoinRequest) (res *internal.JoinResponse, err error) {
	res = &internal.JoinResponse{}
	res.Value, err = s.agent.addReplica(req.HostId, req.IsNonVoting)
	return
}

func (s *grpcServer) Propose(ctx context.Context, req *internal.Request) (res *internal.Response, err error) {
	if s.agent.GetStatus() != AgentStatus_Ready {
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
			if r.Aborted() {
				err = ErrAborted
				return
			} else if r.Committed() {
				if !req.Linear {
					return
				}
			} else if r.Completed() {
				if req.Linear {
					res = &internal.Response{
						Value: r.GetResult().Value,
						Data:  r.GetResult().Data,
					}
				}
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

func (s *grpcServer) Query(ctx context.Context, req *internal.Request) (res *internal.Response, err error) {
	var r any
	if req.Linear {
		r, err = s.agent.host.SyncRead(raftCtx(), req.ShardId, req.Data)
	} else {
		r, err = s.agent.host.StaleRead(req.ShardId, req.Data)
	}
	if r != nil {
		res = r.(*internal.Response)
	}
	return
}

func (s *grpcServer) Start(a *Agent) error {
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
		s.server.GracefulStop()
	case <-done:
	}
	return err
}

func (s *grpcServer) Stop() {
	s.server.GracefulStop()
}
