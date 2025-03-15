package master_server

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"id-generator/internal/cache"
)

const (
	REDIS_COUNTER_KEY   = "counter-key"
	REDIS_TIMESTAMP_KEY = "timestamp-key"
)

type masterServer struct{}

func NewMaster() *masterServer {
	_, err := cache.Dragonfly.RawClient.SetNX(context.Background(), REDIS_TIMESTAMP_KEY, time.Now().UTC().Unix(), 0).Result()
	if err != nil {
		log.Fatalf("error while setting init timestamp to dragonfly: %v", err)
	}

	return &masterServer{}
}

func (c *masterServer) GetMultiplier(ctx context.Context) int32 {
	multiplier, err := cache.Dragonfly.RawClient.Incr(ctx, REDIS_COUNTER_KEY).Result()
	if err != nil {
		log.Printf("error while incrementing the counter in dragonfly: %v", err)
		return 0
	}

	return int32(multiplier)
}

func (c *masterServer) GetTimestamp(ctx context.Context) int64 {
	timestampStr, err := cache.Dragonfly.RawClient.Get(ctx, REDIS_TIMESTAMP_KEY).Result()
	if err != nil {
		log.Printf("error while getting timestamp from dragonfly: %v", err)
		return 0
	}

	timestampInt, err := strconv.Atoi(timestampStr)
	if err != nil {
		log.Printf("timestamp must be of integer type")
		return 0
	}

	return int64(timestampInt)
}

func (c *masterServer) Reset(timestamp int64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		_, err := cache.Dragonfly.RawClient.Set(ctx, REDIS_TIMESTAMP_KEY, timestamp, 0).Result()
		if err != nil {
			log.Printf("error while setting timestamp to dragonfly: %v", err)
		}

		wg.Done()
	}()

	go func() {
		_, err := cache.Dragonfly.RawClient.Set(ctx, REDIS_COUNTER_KEY, 1, 0).Result()
		if err != nil {
			log.Printf("error while setting timestamp to dragonfly: %v", err)
		}

		wg.Done()
	}()

	wg.Wait()
}
