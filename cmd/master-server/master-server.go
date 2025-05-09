package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	master_server "id-generator/internal/master-server"
	"id-generator/internal/pb"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

type grpcServerInternal struct {
	pb.UnimplementedOrchestratorServer
}

var (
	masterServerCache = master_server.NewMasterServer()
	env               = flag.String("env", ".env", "Env file to load values from")
)

func main() {
	flag.Parse()

	err := godotenv.Load(*env)
	if err != nil {
		log.Print("failed to load env file")
	}

	MASTER_SERVER_GRPC_PORT := os.Getenv("MASTER_SERVER_GRPC_PORT")
	if MASTER_SERVER_GRPC_PORT == "" {
		log.Fatal("'KAFKA_ADDR' variable is undefined")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", MASTER_SERVER_GRPC_PORT))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOrchestratorServer(grpcServer, &grpcServerInternal{})
	log.Printf("grpc server listening at %v", lis.Addr())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve grpc: %v", err)
		}
	}()

	masterServerCache.LoadRedisScript()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	grpcServer.GracefulStop()

}

func (s *grpcServerInternal) GetMultiplierAndTimestamp(_ context.Context, _ *pb.MultiplierAndTimestampRequest) (*pb.MultiplierAndTimestampReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	multiplier, timestamp, err := masterServerCache.GetMultiplierAndTimestamp(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.MultiplierAndTimestampReply{
			Timestamp:  timestamp,
			Multiplier: multiplier,
		},
		nil
}
