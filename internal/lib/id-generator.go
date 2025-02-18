package lib

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"id-generator/internal/cache"
)

func GetUniqueId(ctx context.Context, sysType string) (newId string, err error) {
	for {
		if newId, err = generateId(sysType); err != nil {
			return "", err
		}

		isNewIdUnique, err := cache.Dragonfly.SetUniqueKey(ctx, newId, "", time.Second*5)
		if err != nil {
			return "", fmt.Errorf("error while setting to dragonfly db: %v", err)
		}

		if isNewIdUnique {
			break
		}
	}

	return newId, nil
}

func generateId(sysType string) (string, error) {
	sysTypeId, err := GetSysTypeValue(sysType)
	if err != nil {
		return "", err
	}

	randTail := rand.Int32N(int32(math.Pow10(7)))

	return fmt.Sprintf("%010d%01d%07d", time.Now().Unix(), sysTypeId, randTail), nil
}
