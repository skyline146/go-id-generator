package lib

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/redis/go-redis/v9"
)

const REDIS_LOCK_KEY = "some-lock-key"

var dragonfly = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

func GetUniqueId(ctx context.Context, sysType string) (newId string, err error) {
	if err := acquireLock(ctx); err != nil {
		return "", err
	}
	defer func() {
		if err = releaseLock(ctx); err != nil {
			newId = ""
		}
	}()

	for {
		if newId, err = generateId(sysType); err != nil {
			return "", err
		}

		_, err := dragonfly.Get(ctx, newId).Result()

		if err == redis.Nil {
			break
		} else if err != nil {
			return "", fmt.Errorf("error on getting value from dragonfly db by key `%s`: %v", newId, err)
		}
	}

	err = dragonfly.SetEx(ctx, newId, "", time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("error while setting to dragonfly db: %v", err)
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

func acquireLock(ctx context.Context) error {
	for {
		isKeySet, err := dragonfly.SetNX(ctx, REDIS_LOCK_KEY, "", time.Second*10).Result()
		if err != nil {
			return fmt.Errorf("failed on setting lock key in dragonfly: %v", err)
		}

		if isKeySet {
			return nil
		}

		// TODO: maybe impement queue and pub/sub instead of active waiting
		// Pros: avoiding goroutine starvation
		// Cons: maybe less performance due to additional calls to dragonfly
		time.Sleep(time.Millisecond)
	}
}

func releaseLock(ctx context.Context) error {
	_, err := dragonfly.Del(ctx, REDIS_LOCK_KEY).Result()
	if err != nil {
		return fmt.Errorf("failed on deleting lock key from dragonfly: %v", err)
	}

	return nil
}
