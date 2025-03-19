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
	ids              []id
	mu               *sync.Mutex
	defaultCapacity  int32
	MasterGrpcClient pb.OrchestratorClient
}

var Storage = storage{make([]id, 0), &sync.Mutex{}, 1000, nil}

func (s *storage) Fill() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	newData, err := s.MasterGrpcClient.GetMultiplierAndTimestamp(ctx, &pb.MultiplierAndTimestampRequest{})
	if err != nil {
		log.Fatalf("could not get data from master server: %v", err)
	}

	s.ids = append(s.ids, s.generateIdsByCapacity(newData.Multiplier, newData.Timestamp)...)
}

func (s *storage) GetRawId() id {
	s.mu.Lock()
	defer s.mu.Unlock()

	rawId := s.ids[0]
	s.ids = s.ids[1:]

	idsLeftPercentage := float64(len(s.ids)) / float64(s.defaultCapacity)
	if idsLeftPercentage < LEFT_IDS_PERCENTAGE_TO_FILL {
		s.Fill()
	}

	return rawId
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
