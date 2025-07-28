package generator_storage

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"id-generator/internal/cache"
	"id-generator/internal/lib"

	_ "embed"
)

type id struct {
	Timestamp int64
	Tail      int32
}

type Storage struct {
	scriptSha            string
	redisCounterKey      string
	redisTimestampKey    string
	maxAllowedMultiplier int
	idsCh                chan id
	// masterGrpcClient     pb.OrchestratorClient
	isFilling       chan struct{}
	percentWhenFill float64
}

//go:embed redis-script.lua
var redisScript string

func NewStorage(
	redisCounterKey, redisTimestampKey, maxAllowedMultiplierStr, freeDigitsForIdsStr string,
	percentWhenFill float64,
) (*Storage, error) {
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

	storage := &Storage{
		"",
		redisCounterKey,
		redisTimestampKey,
		maxAllowedMultiplier,
		make(chan id, int(maxNumberOfIds/float64(maxAllowedMultiplier))),
		make(chan struct{}, 1),
		percentWhenFill,
	}

	storage.loadRedisScript()
	storage.fill()

	return storage, nil
}

// func (s *Storage) Init(masterGrpcClient pb.OrchestratorClient) {
// 	s.masterGrpcClient = masterGrpcClient
// 	s.fill()
// }

func (s *Storage) GetRawId() id {
	go s.fill()

	return <-s.idsCh
}

func (s *Storage) GetUniqueIdWithType(sysType string) (newId string, err error) {
	sysTypeId, err := lib.GetSysTypeValue(sysType)
	if err != nil {
		return "", err
	}

	rawId := s.GetRawId()

	return fmt.Sprintf("%010d%01d%07d", rawId.Timestamp, sysTypeId, rawId.Tail), nil
}

func (s *Storage) fill() {
	if s.isFillNeeded() {
		select {
		case s.isFilling <- struct{}{}:
		default:
			return
		}
	} else {
		return
	}

	defer func() {
		<-s.isFilling
	}()

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	multiplier, timestamp, err := s.getMultiplierAndTimestamp()
	// fmt.Println(multiplier)
	// fmt.Println(timestamp)
	if err != nil {
		log.Fatalf("could not get data from master server: %v", err)
	}

	newIds := s.generateIdsByChanCapacity(multiplier, timestamp)

	for _, id := range newIds {
		s.idsCh <- id
	}
}

func (s *Storage) isFillNeeded() bool {
	idsLeftPercentage := float64(len(s.idsCh)) / float64(cap(s.idsCh))

	return idsLeftPercentage < s.percentWhenFill
}

func (s *Storage) generateIdsByChanCapacity(multiplier int32, timestamp int64) []id {
	chanCap := cap(s.idsCh)
	newIds := make([]id, chanCap)
	min := int(multiplier)*chanCap - chanCap

	for i := range newIds {
		newIds[i] = id{timestamp, int32(min + i)}
	}

	return newIds
}

func (s *Storage) getMultiplierAndTimestamp() (multiplier int32, timestamp int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result, err := cache.Dragonfly.RawClient.EvalSha(
		ctx,
		s.scriptSha, []string{s.redisCounterKey, s.redisTimestampKey}, s.maxAllowedMultiplier,
	).Int64Slice()
	if err != nil {
		return 0, 0, fmt.Errorf("there was an error while getting multiplier or timestamp: %v", err)
	}

	return int32(result[0]), result[1], nil
}

func (s *Storage) loadRedisScript() {
	hash := sha1.New()
	hash.Write([]byte(redisScript))
	scriptSha := hex.EncodeToString(hash.Sum(nil))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	results, err := cache.Dragonfly.RawClient.ScriptExists(ctx, scriptSha).Result()
	if err != nil {
		fmt.Printf("failed to check if script exists by sha representation: %v\n", err)
	}

	s.scriptSha = scriptSha

	if results[0] {
		return
	}

	_, err = cache.Dragonfly.RawClient.ScriptLoad(ctx, redisScript).Result()
	if err != nil {
		log.Fatalf("failed to load script: %v", err)
	}
}
