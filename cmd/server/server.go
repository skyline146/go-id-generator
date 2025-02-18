package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"id-generator/internal/handlers"
)

var (
	httpPort = flag.Int("http-port", 3000, "Port to run http server")
	grpcPort = flag.Int("grpc-port", 3001, "Port to run grpc server")
)

func main() {
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
