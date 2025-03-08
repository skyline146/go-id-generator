package handlers

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"id-generator/internal/lib"
	"id-generator/internal/pb"

	"google.golang.org/grpc"
)

type GrpcServer struct {
	Port   int
	server *grpc.Server
}

type grpcServerInternal struct {
	pb.UnimplementedGeneratorServer
}

func (s *GrpcServer) Serve() error {
	if s.server != nil {
		return fmt.Errorf("grpc server is already running")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterGeneratorServer(grpcServer, &grpcServerInternal{})
	log.Printf("grpc server listening at %v", lis.Addr())
	s.server = grpcServer

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve grpc: %v", err)
	}

	return nil
}

func (s *GrpcServer) Stop(stopCh chan struct{}, done func()) {
	defer done()

	<-stopCh

	if s.server == nil {
		panic("can't stop non-exist grpc server")
	}

	s.server.GracefulStop()
}

func (s *grpcServerInternal) GetUniqueId(_ context.Context, req *pb.UniqueIdRequest) (*pb.UniqueIdReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	newId, err := lib.GetUniqueIdWithType(ctx, req.GetSysType().String())
	if err != nil {
		return nil, fmt.Errorf("error while generating new unique id: %v", err)
	}

	return &pb.UniqueIdReply{Id: newId}, nil
}
