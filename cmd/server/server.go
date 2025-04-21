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
	"id-generator/internal/pb"
	"id-generator/internal/servers"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	httpPort = flag.Int("http-port", 3000, "Port to run http server")
	grpcPort = flag.Int("grpc-port", 3001, "Port to run grpc server")
	env      = flag.String("env", ".env", "Env file to load values from")
)

type Server interface {
	Serve() error
	Stop(chan struct{}, func())
}

func main() {
	flag.Parse()
	storage := generator_storage.NewStorage(1000)
	initStorageWithMasterServer(storage)

	shutdown := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	servers := []Server{
		servers.NewGrpcServer(*grpcPort, storage),
		servers.NewHttpServer(*httpPort, storage),
	}

	for _, server := range servers {
		go func() {
			go server.Stop(shutdown, wg.Done)

			err := server.Serve()
			if err != nil {
				panic(err)
			}
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	close(shutdown)

	wg.Wait()
}

func initStorageWithMasterServer(storage *generator_storage.Storage) {
	err := godotenv.Load(*env)
	if err != nil {
		log.Print("failed to load env file")
	}

	MASTER_SERVER_GRPC_PORT := os.Getenv("MASTER_SERVER_GRPC_PORT")
	if MASTER_SERVER_GRPC_PORT == "" {
		log.Fatal("'MASTER_SERVER_GRPC_PORT' variable is undefined")
	}

	masterServerGrpcAddr := fmt.Sprintf("%s:%s", "localhost", MASTER_SERVER_GRPC_PORT)
	conn, err := grpc.NewClient(masterServerGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to master's grpc server (%s): %v\n", masterServerGrpcAddr, err)
	}

	storage.Init(pb.NewOrchestratorClient(conn))
}
