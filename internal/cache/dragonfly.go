package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	REDIS_LOCK_KEY       = "some-lock-key"
	REDIS_NOTIFY_CHANNEL = "some-notify-channel"
)

type dragonfly struct {
	RawClient *redis.Client
}

var Dragonfly = dragonfly{
	RawClient: redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	}),
}

func (dg *dragonfly) SetUniqueKey(ctx context.Context, key string, value any, exp time.Duration) (bool, error) {
	isKeyUnique, err := dg.RawClient.SetNX(ctx, key, value, exp).Result()
	if err != nil {
		return false, fmt.Errorf("error while setting to dragonfly db: %v", err)
	}

	return isKeyUnique, nil
}

func (dg *dragonfly) AcquireLock(ctx context.Context) error {
	for {
		isKeySet, err := dg.RawClient.SetNX(ctx, REDIS_LOCK_KEY, "", time.Second*10).Result()
		if err != nil {
			return fmt.Errorf("failed on setting lock key: %v", err)
		}

		if isKeySet {
			return nil
		}

		time.Sleep(time.Millisecond)
	}
}

func (dg *dragonfly) ReleaseLock(ctx context.Context) error {
	_, err := dg.RawClient.Del(ctx, REDIS_LOCK_KEY).Result()
	if err != nil {
		return fmt.Errorf("failed on deleting lock key: %v", err)
	}

	return nil
}
