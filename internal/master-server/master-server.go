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
	REDIS_SCRIPT_HASH_KEY  = "script-hash-key"
	REDIS_COUNTER_KEY      = "counter-key"
	REDIS_TIMESTAMP_KEY    = "timestamp-key"
	MAX_ALLOWED_MULTIPLIER = 10000
)

type masterServer struct {
	scriptSha string
}

func NewMaster() *masterServer {
	_, err := cache.Dragonfly.RawClient.SetNX(context.Background(), REDIS_TIMESTAMP_KEY, time.Now().UTC().Unix(), 0).Result()
	if err != nil {
		log.Fatalf("error while setting init timestamp to dragonfly: %v", err)
	}

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

	if results[0] {
		ms.scriptSha = scriptSha
		return
	}

	scriptSha, err = cache.Dragonfly.RawClient.ScriptLoad(ctx, string(bytes)).Result()
	if err != nil {
		log.Fatalf("failed to load script: %v", err)
	}

	ms.scriptSha = scriptSha
}

func (ms *masterServer) GetMultiplierAndTimestamp(ctx context.Context) (multiplier int32, timestamp int64, err error) {
	result, err := cache.Dragonfly.RawClient.EvalSha(ctx, ms.scriptSha, []string{REDIS_COUNTER_KEY, REDIS_TIMESTAMP_KEY}, MAX_ALLOWED_MULTIPLIER).Int64Slice()
	if err != nil {
		return 0, 0, fmt.Errorf("there was an error while getting multiplier or timestamp: %v", err)
	}

	return int32(result[0]), result[1], nil
}

// func (ms *masterServer) getMultiplier(ctx context.Context) int32 {
// 	multiplier, err := cache.Dragonfly.RawClient.Incr(ctx, REDIS_COUNTER_KEY).Result()
// 	if err != nil {
// 		log.Printf("error while incrementing the counter in dragonfly: %v", err)
// 		return 0
// 	}

// 	return int32(multiplier)
// }

// func (ms *masterServer) getTimestamp(ctx context.Context) int64 {
// 	timestampStr, err := cache.Dragonfly.RawClient.Get(ctx, REDIS_TIMESTAMP_KEY).Result()
// 	if err != nil {
// 		log.Printf("error while getting timestamp from dragonfly: %v", err)
// 		return 0
// 	}

// 	timestampInt, err := strconv.Atoi(timestampStr)
// 	if err != nil {
// 		log.Printf("timestamp must be of integer type")
// 		return 0
// 	}

// 	return int64(timestampInt)
// }

// func (ms *masterServer) reset(timestamp int64) {
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
// 	defer cancel()

// 	_, err := cache.Dragonfly.RawClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
// 		_, err := pipe.Set(ctx, REDIS_TIMESTAMP_KEY, timestamp, 0).Result()
// 		if err != nil {
// 			log.Printf("error while setting timestamp to dragonfly: %v\n", err)
// 		}
// 		_, err = pipe.Set(ctx, REDIS_COUNTER_KEY, 1, 0).Result()
// 		if err != nil {
// 			log.Printf("error while setting timestamp to dragonfly: %v\n", err)
// 		}

// 		return nil
// 	})
// 	if err != nil {
// 		log.Printf("error in pipeline while resetting multiplier and timestamp: %v\n", err)
// 	}
// }
