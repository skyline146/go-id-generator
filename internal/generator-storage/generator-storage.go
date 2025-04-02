package generator_storage

import (
	"context"
	"log"
	"sync"
	"time"

	"id-generator/internal/pb"
)

const LEFT_IDS_PERCENTAGE_TO_FILL = 0.3

type id struct {
	Timestamp int64
	Tail      int32
}

type storage struct {
	idsCh            chan id
	mu               *sync.Mutex
	isFilling        bool
	defaultCapacity  int32
	MasterGrpcClient pb.OrchestratorClient
}

var Storage = storage{make(chan id, 1000), &sync.Mutex{}, false, 1000, nil}

func (s *storage) Fill() {
	defer func() {
		s.mu.Lock()
		s.isFilling = false
		s.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	newData, err := s.MasterGrpcClient.GetMultiplierAndTimestamp(ctx, &pb.MultiplierAndTimestampRequest{})
	if err != nil {
		log.Fatalf("could not get data from master server: %v", err)
	}

	newIds := s.generateIdsByCapacity(newData.Multiplier, newData.Timestamp)

	for _, id := range newIds {
		s.idsCh <- id
	}
}

func (s *storage) GetRawId() id {
	go s.checkIsFillNeeded()

	rawId := <-s.idsCh

	return rawId
}

func (s *storage) checkIsFillNeeded() {
	idsLeftPercentage := float64(len(s.idsCh)) / float64(s.defaultCapacity)

	if idsLeftPercentage < LEFT_IDS_PERCENTAGE_TO_FILL {
		s.mu.Lock()
		if !s.isFilling {
			s.isFilling = true
			go s.Fill()
		}
		s.mu.Unlock()
	}
}

func (s *storage) generateIdsByCapacity(multiplier int32, timestamp int64) []id {
	newIds := make([]id, s.defaultCapacity)
	min, max := int(multiplier*s.defaultCapacity-s.defaultCapacity), int(multiplier*s.defaultCapacity)

	idx := 0
	for i := min; i < max; i++ {
		newIds[idx] = id{timestamp, int32(i)}
		idx++
	}

	return newIds
}
