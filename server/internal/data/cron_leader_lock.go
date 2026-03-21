package data

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"sisyphus/server/internal/biz"
)

type cronLeaderLock struct {
	rdb *redis.Client
}

var cronLeaderReleaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

// NewCronLeaderLock 基于 Redis 的定时任务 leader 锁（与 Asynq 共用同一 Redis 即可）。
func NewCronLeaderLock(d *Data) biz.CronLeaderLock {
	if d == nil {
		return nil
	}
	return &cronLeaderLock{rdb: d.Redis()}
}

func (c *cronLeaderLock) Acquire(ctx context.Context, key string, ttl time.Duration) (func(), bool, error) {
	if c == nil || c.rdb == nil {
		return nil, false, errors.New("redis client unavailable")
	}
	token := uuid.NewString()
	ok, err := c.rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	release := func() {
		_ = cronLeaderReleaseScript.Run(context.Background(), c.rdb, []string{key}, token).Err()
	}
	return release, true, nil
}
