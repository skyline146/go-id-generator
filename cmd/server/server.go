package main

import (
	"flag"
	"log"

	"id-generator/internal/handlers"
)

var (
	httpPort = flag.Int("http-port", 3000, "Port to run http server")
	grpcPort = flag.Int("grpc-port", 3001, "Port to run grpc server")
)

func main() {
	flag.Parse()

	go func() {
		httpServer := &handlers.HttpServer{Port: *httpPort}

		err := httpServer.Serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	grpcServer := &handlers.GrpcServer{Port: *grpcPort}

	err := grpcServer.Serve()
	if err != nil {
		log.Fatal(err)
	}
}
