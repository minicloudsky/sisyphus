package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// CronLeaderLock 多副本部署时通过 Redis 分布式锁保证同一时刻只有一个实例执行定时调度逻辑。
type CronLeaderLock interface {
	// Acquire 尝试获取锁。ok 为 false 表示其他实例已持有锁，应跳过本次调度。
	// release 在成功获取锁时必须调用（通常 defer），用于安全释放锁。
	Acquire(ctx context.Context, key string, ttl time.Duration) (release func(), ok bool, err error)
}

// runCronAsLeader 在持锁时执行 fn；无锁或 Redis 不可用时降级为本地执行（与单副本行为一致）。
func runCronAsLeader(h *log.Helper, lock CronLeaderLock, ctx context.Context, key string, ttl time.Duration, fn func()) {
	if lock == nil {
		fn()
		return
	}
	release, ok, err := lock.Acquire(ctx, key, ttl)
	if err != nil {
		h.Warnf("cron leader lock 不可用，降级为本地执行（多副本时可能重复）: key=%s err=%v", key, err)
		fn()
		return
	}
	if !ok {
		return
	}
	defer release()
	fn()
}
