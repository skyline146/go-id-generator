package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	generator_storage "id-generator/internal/generator-storage"
	"id-generator/internal/servers"

	"github.com/joho/godotenv"
)

var (
	httpPort        = flag.Int("http-port", 3000, "Port to run http server")
	grpcPort        = flag.Int("grpc-port", 3001, "Port to run grpc server")
	env             = flag.String("env", ".env", "Env(s) file to load variables from. E.g. .env or .env1,.env2")
	percentWhenFill = flag.Float64("when-fill", 0.3, "Percentage when channel of generated ids make a new request for multiplier. E.g. 0.3 = 30%")
)

type Server interface {
	Serve() error
	Stop(chan struct{}, func())
}

func main() {
	flag.Parse()

	err := godotenv.Load(strings.Split(*env, ",")...)
	if err != nil {
		log.Print("failed to load env file(s)")
	}

	storage, err := generator_storage.NewStorage(
		os.Getenv("REDIS_COUNTER_KEY"),
		os.Getenv("REDIS_TIMESTAMP_KEY"),
		os.Getenv("MAX_ALLOWED_MULTIPLIER"),
		os.Getenv("FREE_DIGITS_FOR_IDS"),
		*percentWhenFill,
	)
	if err != nil {
		log.Fatalf("error in initializing storage server: %v", err)
	}

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

// func initStorageWithMasterServer(storage *generator_storage.Storage) {
// 	MASTER_SERVER_GRPC_PORT := os.Getenv("MASTER_SERVER_GRPC_PORT")
// 	if MASTER_SERVER_GRPC_PORT == "" {
// 		log.Fatal("'MASTER_SERVER_GRPC_PORT' variable is undefined")
// 	}

// 	masterServerGrpcAddr := fmt.Sprintf("%s:%s", "localhost", MASTER_SERVER_GRPC_PORT)
// 	conn, err := grpc.NewClient(masterServerGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		log.Fatalf("failed to connect to master's grpc server (%s): %v\n", masterServerGrpcAddr, err)
// 	}

// 	storage.Init(pb.NewOrchestratorClient(conn))
// }
