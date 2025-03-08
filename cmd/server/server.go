package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	generator_storage "id-generator/internal/generator-storage"
	"id-generator/internal/handlers"
	"id-generator/internal/pb"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	httpPort = flag.Int("http-port", 3000, "Port to run http server")
	grpcPort = flag.Int("grpc-port", 3001, "Port to run grpc server")
)

func main() {
	initWithMaster()
	flag.Parse()

	shutdown := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		httpServer := &handlers.HttpServer{Port: *httpPort}

		go httpServer.Stop(shutdown, wg.Done)

		err := httpServer.Serve()
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		grpcServer := &handlers.GrpcServer{Port: *grpcPort}

		go grpcServer.Stop(shutdown, wg.Done)

		err := grpcServer.Serve()
		if err != nil {
			panic(err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	close(shutdown)

	wg.Wait()
}

func initWithMaster() {
	err := godotenv.Load()
	if err != nil {
		log.Print("failed to load env file")
	}

	MASTER_SERVER_GRPC_PORT := os.Getenv("MASTER_SERVER_GRPC_PORT")
	if MASTER_SERVER_GRPC_PORT == "" {
		log.Fatal("'KAFKA_ADDR' variable is undefined")
	}

	masterServerGrpcAddr := fmt.Sprintf("%s:%s", "localhost", MASTER_SERVER_GRPC_PORT)
	conn, err := grpc.NewClient(masterServerGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to master's grpc server (%s): %v\n", masterServerGrpcAddr, err)
	}

	generator_storage.Storage.MasterGrpcClient = pb.NewOrchestratorClient(conn)
	generator_storage.Storage.Fill()
}
