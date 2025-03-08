package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"id-generator/internal/pb"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

const CURRENT_MULTIPLIER_KEY = "current-multiplier-key"

type grpcServerInternal struct {
	pb.UnimplementedOrchestratorServer
}

type masterServer struct {
	multiplier int32
	timestamp  int64
	mu         *sync.Mutex
}

var masterServerCache = masterServer{1, time.Now().Unix(), &sync.Mutex{}}

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

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve grpc: %v", err)
	}
}

func (s *grpcServerInternal) GetMultiplierAndTimestamp(_ context.Context, _ *pb.MultiplierAndTimestampRequest) (*pb.MultiplierAndTimestampReply, error) {
	masterServerCache.mu.Lock()
	defer masterServerCache.mu.Unlock()

	defer func() {
		masterServerCache.multiplier++
	}()

	now := time.Now().Unix()

	if masterServerCache.multiplier > 10000 {
		waitUntilTimestampChanges(masterServerCache.timestamp)
	}

	if masterServerCache.timestamp < now {
		masterServerCache.timestamp = now
		masterServerCache.multiplier = 1
	}

	return &pb.MultiplierAndTimestampReply{
			Timestamp:  masterServerCache.timestamp,
			Multiplier: masterServerCache.multiplier,
		},
		nil
}

func waitUntilTimestampChanges(currentTimestamp int64) {
	for currentTimestamp != time.Now().Unix() {
	}
}
