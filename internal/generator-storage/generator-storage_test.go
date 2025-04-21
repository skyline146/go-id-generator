package generator_storage

import (
	"fmt"
	"id-generator/internal/pb"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	testStorage_master1 = NewStorage(1000)
	testStorage_master2 = NewStorage(1000)
)

func getMasterGrpcClientFromEnv(envFile string) pb.OrchestratorClient {
	err := godotenv.Load(envFile)
	if err != nil {
		log.Printf("failed to load env file: %s\n", envFile)
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

	return pb.NewOrchestratorClient(conn)
}

func setup() {
	testStorage_master1.Init(getMasterGrpcClientFromEnv("../../.env.master1"))
	testStorage_master2.Init(getMasterGrpcClientFromEnv("../../.env.master2"))
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func TestIdsOnUniqueness(t *testing.T) {
	var (
		wg   sync.WaitGroup
		sMap sync.Map
	)
	wg.Add(2)

	start := time.Now()

	go func() {
		for range 1000000 {
			id, err := testStorage_master1.GetUniqueIdWithType("Vendor")
			if err != nil {
				fmt.Println(err)
				continue
			}

			val, loaded := sMap.LoadOrStore(id, 1)
			if loaded {
				sMap.Store(id, val.(int)+1)
			}
		}
		wg.Done()
	}()

	go func() {
		for range 1000000 {
			id, err := testStorage_master2.GetUniqueIdWithType("Vendor")
			if err != nil {
				fmt.Println(err)
				continue
			}

			val, loaded := sMap.LoadOrStore(id, 1)
			if loaded {
				sMap.Store(id, val.(int)+1)
			}
		}
		wg.Done()
	}()

	wg.Wait()

	t.Logf("execution time of 2kk requests: %.3fs\n", time.Since(start).Seconds())

	notUniqueIds := []string{}

	sMap.Range(func(key, value any) bool {
		if value.(int) > 1 {
			notUniqueIds = append(notUniqueIds, key.(string))
		}

		return true
	})

	if len(notUniqueIds) != 0 {
		t.Errorf("there are not uniques ids: %q\n", notUniqueIds)
	}
}
