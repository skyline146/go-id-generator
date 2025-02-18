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
	rawClient *redis.Client
}

var Dragonfly = dragonfly{
	rawClient: redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	}),
}

func (dg *dragonfly) SetUniqueKey(ctx context.Context, key string, value any, exp time.Duration) (bool, error) {
	isKeyUnique, err := dg.rawClient.SetNX(ctx, key, value, exp).Result()
	if err != nil {
		return false, fmt.Errorf("error while setting to dragonfly db: %v", err)
	}

	return isKeyUnique, nil
}

func (dg *dragonfly) AcquireLock(ctx context.Context) error {
	for {
		isKeySet, err := dg.rawClient.SetNX(ctx, REDIS_LOCK_KEY, "", time.Second*10).Result()
		if err != nil {
			return fmt.Errorf("failed on setting lock key: %v", err)
		}

		if isKeySet {
			return nil
		}

		// channel := dragonfly.Subscribe(ctx, REDIS_NOTIFY_CHANNEL)
		// defer channel.Close()

		// notifyChannel := channel.Channel(redis.WithChannelSendTimeout(time.Second * 10))

		// for {
		// 	redisMsg := <-notifyChannel
		// 	if redisMsg.Payload == "unlock" {
		// 		break
		// 	}
		// }

		time.Sleep(time.Millisecond)
	}
}

func (dg *dragonfly) ReleaseLock(ctx context.Context) error {
	_, err := dg.rawClient.Del(ctx, REDIS_LOCK_KEY).Result()
	if err != nil {
		return fmt.Errorf("failed on deleting lock key: %v", err)
	}

	// _, err = dragonfly.Publish(ctx, REDIS_NOTIFY_CHANNEL, "unlock").Result()
	// if err != nil {
	// 	return fmt.Errorf("failed on publishing to notify channel: %v", err)
	// }

	return nil
}
