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
	Port int
}

type grpcServerInternal struct {
	pb.UnimplementedGeneratorServer
}

func (s *GrpcServer) Serve() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterGeneratorServer(grpcServer, &grpcServerInternal{})
	log.Printf("grpc server listening at %v", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve grpc: %v", err)
	}

	return nil
}

func (s *grpcServerInternal) GetUniqueId(_ context.Context, req *pb.UniqueIdRequest) (*pb.UniqueIdReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	newId, err := lib.GetUniqueId(ctx, req.GetSysType().String())
	if err != nil {
		return nil, fmt.Errorf("error while generating new unique id: %v", err)
	}

	return &pb.UniqueIdReply{Id: newId}, nil
}
