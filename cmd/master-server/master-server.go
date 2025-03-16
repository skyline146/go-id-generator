package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"id-generator/internal/cache"
	"id-generator/internal/lib"
	master_server "id-generator/internal/master-server"
	"id-generator/internal/pb"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

const MAX_ALLOWED_MULTIPLIER = 10000

type grpcServerInternal struct {
	pb.UnimplementedOrchestratorServer
}

var masterServerCache = master_server.NewMaster()

func main() {
	err := godotenv.Load()
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
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve grpc: %v", err)
	}
}

func (s *grpcServerInternal) GetMultiplierAndTimestamp(_ context.Context, _ *pb.MultiplierAndTimestampRequest) (*pb.MultiplierAndTimestampReply, error) {
	masterServerCache.Lock()
	defer masterServerCache.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cache.Dragonfly.AcquireLock(ctx)
	defer cache.Dragonfly.ReleaseLock(ctx)

	multiplier, timestamp := masterServerCache.GetMultiplierAndTimestamp(ctx)

	if multiplier == 0 || timestamp == 0 {
		return nil, fmt.Errorf("there was an error while getting multiplier or timestamp")
	}

	if multiplier > MAX_ALLOWED_MULTIPLIER {
		timestamp = lib.WaitUntilTimestampChanges(timestamp)
		multiplier = 1

		masterServerCache.Reset(timestamp)
	}

	return &pb.MultiplierAndTimestampReply{
			Timestamp:  timestamp,
			Multiplier: multiplier,
		},
		nil
}
