package master_server

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"id-generator/internal/cache"
	"id-generator/internal/lib"
)

const (
	REDIS_COUNTER_KEY      = "counter-key"
	REDIS_TIMESTAMP_KEY    = "timestamp-key"
	MAX_ALLOWED_MULTIPLIER = 10000
)

type masterServer struct{}

func NewMaster() *masterServer {
	_, err := cache.Dragonfly.RawClient.SetNX(context.Background(), REDIS_TIMESTAMP_KEY, time.Now().UTC().Unix(), 0).Result()
	if err != nil {
		log.Fatalf("error while setting init timestamp to dragonfly: %v", err)
	}

	return &masterServer{}
}

func (ms *masterServer) getMultiplier(ctx context.Context) int32 {
	multiplier, err := cache.Dragonfly.RawClient.Incr(ctx, REDIS_COUNTER_KEY).Result()
	if err != nil {
		log.Printf("error while incrementing the counter in dragonfly: %v", err)
		return 0
	}

	return int32(multiplier)
}

func (ms *masterServer) getTimestamp(ctx context.Context) int64 {
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

func (ms *masterServer) reset(timestamp int64) {
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

func (ms *masterServer) GetMultiplierAndTimestamp(ctx context.Context) (multiplier int32, timestamp int64, err error) {
	timestampCh := make(chan int64, 1)
	go func() {
		timestampCh <- ms.getTimestamp(ctx)
	}()
	multiplier = ms.getMultiplier(ctx)
	timestamp = <-timestampCh

	if multiplier == 0 || timestamp == 0 {
		return 0, 0, fmt.Errorf("there was an error while getting multiplier or timestamp")
	}

	if multiplier > MAX_ALLOWED_MULTIPLIER {
		timestamp = lib.WaitUntilTimestampChanges(timestamp)
		multiplier = 1

		ms.reset(timestamp)
	}

	return
}
