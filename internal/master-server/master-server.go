package master_server

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"id-generator/internal/cache"
)

type MasterServer struct {
	scriptSha            string
	redisCounterKey      string
	redisTimestampKey    string
	maxAllowedMultiplier int
}

func NewMasterServer(redisCounterKey, redisTimestampKey, maxAllowedMultiplierStr, freeDigitsForIdsStr string) (*MasterServer, error) {
	if redisCounterKey == "" || redisTimestampKey == "" {
		return nil, fmt.Errorf("redis keys REDIS_COUNTER_KEY or REDIS_TIMESTAMP_KEY must not be empty")
	}

	maxAllowedMultiplier, err := strconv.Atoi(maxAllowedMultiplierStr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to int MAX_ALLOWED_MULTIPLIER")
	}

	freeDigitsForIds, err := strconv.Atoi(freeDigitsForIdsStr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to int FREE_DIGITS_FOR_IDS")
	}

	maxNumberOfIds := math.Pow10(freeDigitsForIds)
	if maxNumberOfIds < float64(maxAllowedMultiplier) {
		return nil, fmt.Errorf("10^(FREE_DIGITS_FOR_IDS) must not be less than MAX_ALLOWED_MULTIPLIER")
	}

	return &MasterServer{"", redisCounterKey, redisTimestampKey, maxAllowedMultiplier}, nil
}

func (ms *MasterServer) LoadRedisScript() {
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

func (ms *MasterServer) GetMultiplierAndTimestamp(ctx context.Context) (multiplier int32, timestamp int64, err error) {
	result, err := cache.Dragonfly.RawClient.EvalSha(
		ctx,
		ms.scriptSha, []string{ms.redisCounterKey, ms.redisTimestampKey}, ms.maxAllowedMultiplier,
	).Int64Slice()
	if err != nil {
		return 0, 0, fmt.Errorf("there was an error while getting multiplier or timestamp: %v", err)
	}

	return int32(result[0]), result[1], nil
}
