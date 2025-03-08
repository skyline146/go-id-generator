package generator_storage

import (
	"context"
	"log"
	"sync"
	"time"

	"id-generator/internal/pb"
)

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

func (gs *storage) Fill() {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	newData, err := gs.MasterGrpcClient.GetMultiplierAndTimestamp(ctx, &pb.MultiplierAndTimestampRequest{})
	if err != nil {
		log.Fatalf("could not get data from master server: %v", err)
	}

	newIds := make([]id, gs.defaultCapacity)
	min, max := int(newData.Multiplier*gs.defaultCapacity-gs.defaultCapacity), int(newData.Multiplier*gs.defaultCapacity)

	idx := 0
	for i := min; i < max; i++ {
		newIds[idx] = id{newData.Timestamp, int32(i)}
		idx++
	}

	gs.ids = append(gs.ids, newIds...)
}

func (gs *storage) GetRawId() id {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	id := gs.ids[0]
	gs.ids = gs.ids[1:]

	idsLeftPercentage := float64(len(gs.ids)) / float64(gs.defaultCapacity)
	if idsLeftPercentage < 0.3 {
		go gs.Fill()
	}

	return id
}
