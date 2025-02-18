package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"id-generator/internal/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var numOfRequestsFlag = flag.Int("requests", 100, "Number of test requests per 1 server")

const (
	host = "localhost"
)

var (
	grpcAddr1 = fmt.Sprintf("%s:%d", host, 3001)
	grpcAddr2 = fmt.Sprintf("%s:%d", host, 3003)
	httpAddr1 = fmt.Sprintf("http://%s:%d", host, 3000)
	httpAddr2 = fmt.Sprintf("http://%s:%d", host, 3002)
)

func main() {
	execStart := time.Now()

	flag.Parse()
	numOfRequests := *numOfRequestsFlag

	grpcClient1, conn1 := initGrpcClient(grpcAddr1)
	grpcClient2, conn2 := initGrpcClient(grpcAddr2)

	var wg sync.WaitGroup
	wg.Add(4)

	var ids sync.Map

	storeOrIncrement := func(key string) {
		if key == "" {
			return
		}

		v, loaded := ids.LoadOrStore(key, 1)
		if loaded {
			ids.Store(key, v.(int)+1)
		}
	}

	go func() {
		defer wg.Done()
		for i := 1; i <= numOfRequests; i++ {
			storeOrIncrement(mockHttpRequest(httpAddr1))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 1; i <= numOfRequests; i++ {
			storeOrIncrement(mockHttpRequest(httpAddr2))
		}
	}()

	go func() {
		defer wg.Done()
		defer conn1.Close()

		for i := 1; i <= numOfRequests; i++ {
			storeOrIncrement(mockGrpcRequest(grpcClient1, grpcAddr1))
		}
	}()

	go func() {
		defer wg.Done()
		defer conn2.Close()

		for i := 1; i <= numOfRequests; i++ {
			storeOrIncrement(mockGrpcRequest(grpcClient2, grpcAddr2))
		}
	}()

	wg.Wait()

	fmt.Printf("Total time of execution %d requests: %.3fs\n", numOfRequests*4, time.Since(execStart).Seconds())

	i := 0
	ids.Range(func(key, value any) bool {
		i++
		if value.(int) > 1 {
			fmt.Println("Found duplicate id: ", key)
		}

		return true
	})

	fmt.Println("Total received ids: ", i)
	if numOfRequests*4 == i {
		fmt.Println("No duplicate ids were found")
	}
}

func mockHttpRequest(httpAddr string) string {
	httpIpAndPort := httpAddr[7:]

	response, err := http.Get(fmt.Sprintf("%s/get-unique-id?sys_type=Clients", httpAddr))
	if err != nil {
		log.Printf("failed when getting id from http (%s): %v\n", httpIpAndPort, err)
		return ""
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("error reading http response body (%s): %v\n", httpIpAndPort, err)
		return ""
	}

	fmt.Printf("unique id from http (%s): %s\n", httpIpAndPort, string(body))

	return string(body)
}

func mockGrpcRequest(grpcClient pb.GeneratorClient, grpcAddr string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	response, err := grpcClient.GetUniqueId(ctx, &pb.UniqueIdRequest{SysType: pb.SysType_Vendor})
	if err != nil {
		log.Printf("failed when getting id from gprc (%s): %v\n", grpcAddr, err)
		cancel()
		return ""
	}

	fmt.Printf("unique id from grpc (%s): %s\n", grpcAddr, response.GetId())
	cancel()

	return response.GetId()
}

func initGrpcClient(grpcAddr string) (pb.GeneratorClient, *grpc.ClientConn) {
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to grpc server (%s): %v\n", grpcAddr, err)
	}

	grpcClient := pb.NewGeneratorClient(conn)

	return grpcClient, conn
}
