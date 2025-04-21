package master_server

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"id-generator/internal/cache"
)

const (
	REDIS_COUNTER_KEY      = "counter-key"
	REDIS_TIMESTAMP_KEY    = "timestamp-key"
	MAX_ALLOWED_MULTIPLIER = 10000
)

type masterServer struct {
	scriptSha string
}

func NewMasterServer() *masterServer {
	return &masterServer{}
}

func (ms *masterServer) LoadRedisScript() {
	bytes, err := os.ReadFile("redis-script.lua")
	if err != nil {
		log.Fatalf("failed to load script.lua file: %v", err)
	}

	hash := sha1.New()
	hash.Write(bytes)
	scriptSha := hex.EncodeToString(hash.Sum(nil))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	results, err := cache.Dragonfly.RawClient.ScriptExists(ctx, scriptSha).Result()
	if err != nil {
		fmt.Printf("failed to check if script exists by sha representation: %v\n", err)
	}

	ms.scriptSha = scriptSha

	if results[0] {
		return
	}

	_, err = cache.Dragonfly.RawClient.ScriptLoad(ctx, string(bytes)).Result()
	if err != nil {
		log.Fatalf("failed to load script: %v", err)
	}

}

func (ms *masterServer) GetMultiplierAndTimestamp(ctx context.Context) (multiplier int32, timestamp int64, err error) {
	result, err := cache.Dragonfly.RawClient.EvalSha(ctx, ms.scriptSha, []string{REDIS_COUNTER_KEY, REDIS_TIMESTAMP_KEY}, MAX_ALLOWED_MULTIPLIER).Int64Slice()
	if err != nil {
		return 0, 0, fmt.Errorf("there was an error while getting multiplier or timestamp: %v", err)
	}

	return int32(result[0]), result[1], nil
}
