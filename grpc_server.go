package zongzi

import (
	"google.golang.org/grpc"

	"github.com/logbn/zongzi/internal"
)

type grpcServer struct {
	internal.UnimplementedZongziServer

	agent Agent
	grpc  *grpc.Server
	listenAddr string
	secrets []string
}

func newGrpcServer(a Agent, listenAddr string, secrets []string) *grpcServer {
	return &grpcServer{
		agent: a,
		listenAddr: listenAddr,
		secrets: secrets,
	}
}

func (s *grpcServer) Probe(ctx context.Context, req *internal.ProbeRequest) (res *internal.ProbeResponse, err error) {
	return &internal.ProbeResponse {
		GossipAdvertiseAddress: s.agent.GetHostConfig().Gossip.AdvertiseAddress,
	}, nil
}

func (s *grpcServer) Info(ctx context.Context, req *internal.InfoRequest) (res *internal.InfoResponse, err error) {
	return &internal.InfoResponse {
		HostID: a.GetHostID(),
		ReplicaID: a.GetPrimeConfig().ReplicaID,
	}, nil
}

func (s *grpcServer) Members(ctx context.Context, req *internal.MembersRequest) (res *internal.MembersResponse, err error) {
	return &internal.MembersResponse {
		Members: a.GetMembers(),
	}, nil
}

func (s *grpcServer) Join(ctx context.Context, req *internal.JoinRequest) (res *internal.JoinResponse, err error) {
	h := a.GetHost()
	if req.Voting {
		err = h.SyncRequestAddReplica(raftCtx(), a.GetPrimeConfig().ShardID, req.ReplicaID, req.HostID, req.Index)
	} else {
		err = h.SyncRequestAddNonVoting(raftCtx(), a.GetPrimeConfig().ShardID, req.ReplicaID, req.HostID, req.Index)
	}
	res = &internal.JoinResponse{}
	if err == nil {
		res.Value = 1
	} else {
		res.Error = err.Error()
	}

	return
}

func (s *grpcServer) Propose(ctx context.Context, req *internal.ProposeRequest) (res *internal.ProposeResponse, err error) {

}

func (s *grpcServer) Query(ctx context.Context, req *internal.QueryRequest) (res *internal.QueryResponse, err error) {

}

func (s *grpcServer) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	var opts []grpc.ServerOption
	// https://github.com/grpc/grpc-go/tree/master/examples/features/authentication
	// opts = append(opts, grpc.UnaryInterceptor(ensureValidToken))
	s.grpc := grpc.NewServer(opts...)
	internal.RegisterCoordinationServiceServer(s.grpc, s)
	var err error
	var done = make(chan bool)
	go func() {
		err = s.grpc.Serve(lis)
		close(done)
	}()
	select {
	case <-ctx.Done():
		s.grpc.GracefulStop()
	case <-done:
	}
	return err
}

func (s *grpcServer) Stop() {
	s.grpc.GracefulStop()
}
