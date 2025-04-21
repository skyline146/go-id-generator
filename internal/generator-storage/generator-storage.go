package generator_storage

import (
	"context"
	"fmt"
	"log"
	"time"

	"id-generator/internal/lib"
	"id-generator/internal/pb"
)

const LEFT_IDS_PERCENTAGE_TO_FILL = 0.3

type id struct {
	Timestamp int64
	Tail      int32
}

type Storage struct {
	idsCh            chan id
	masterGrpcClient pb.OrchestratorClient
}

func NewStorage(capacity int32) *Storage {
	return &Storage{
		idsCh: make(chan id, capacity),
	}
}

func (s *Storage) Init(grpcClient pb.OrchestratorClient) {
	s.masterGrpcClient = grpcClient
	s.fill()
	go s.observeIdsChanLen()
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	newData, err := s.masterGrpcClient.GetMultiplierAndTimestamp(ctx, &pb.MultiplierAndTimestampRequest{})
	if err != nil {
		log.Fatalf("could not get data from master server: %v", err)
	}

	newIds := s.generateIdsByChanCapacity(newData)

	for _, id := range newIds {
		s.idsCh <- id
	}
}

func (s *Storage) GetRawId() id {
	return <-s.idsCh
}

func (s *Storage) isFillNeeded() bool {
	idsLeftPercentage := float64(len(s.idsCh)) / float64(cap(s.idsCh))

	return idsLeftPercentage < LEFT_IDS_PERCENTAGE_TO_FILL
}

func (s *Storage) observeIdsChanLen() {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if s.isFillNeeded() {
			s.fill()
		}
	}
}

func (s *Storage) generateIdsByChanCapacity(newData *pb.MultiplierAndTimestampReply) []id {
	chanCap := cap(s.idsCh)
	newIds := make([]id, chanCap)
	min := int(newData.Multiplier)*chanCap - chanCap

	for i := range newIds {
		newIds[i] = id{newData.Timestamp, int32(min + i)}
	}

	return newIds
}
