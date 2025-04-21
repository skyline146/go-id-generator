package servers

import (
	"context"
	"fmt"
	"log"
	"net"

	generator_storage "id-generator/internal/generator-storage"
	"id-generator/internal/pb"

	"google.golang.org/grpc"
)

type grpcServer struct {
	Port    int
	Storage *generator_storage.Storage
	server  *grpc.Server
}

type grpcController struct {
	pb.UnimplementedGeneratorServer
	storage *generator_storage.Storage
}

func NewGrpcServer(port int, storage *generator_storage.Storage) *grpcServer {
	return &grpcServer{
		Port:    port,
		Storage: storage,
	}
}

func (s *grpcServer) Serve() error {
	if s.server != nil {
		return fmt.Errorf("grpc server is already running")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterGeneratorServer(grpcServer, &grpcController{
		storage: s.Storage,
	})
	log.Printf("grpc server listening at %v", lis.Addr())
	s.server = grpcServer

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve grpc: %v", err)
	}

	return nil
}

func (s *grpcServer) Stop(stopCh chan struct{}, done func()) {
	defer done()

	<-stopCh

	if s.server == nil {
		panic("can't stop non-exist grpc server")
	}

	s.server.GracefulStop()
}

func (s *grpcController) GetUniqueId(_ context.Context, req *pb.UniqueIdRequest) (*pb.UniqueIdReply, error) {
	newId, err := s.storage.GetUniqueIdWithType(req.GetSysType().String())
	if err != nil {
		return nil, fmt.Errorf("error while generating new unique id: %v", err)
	}

	return &pb.UniqueIdReply{Id: newId}, nil
}
