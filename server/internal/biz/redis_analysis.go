package biz

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

// RedisAnalysisUsecase Redis分析用例
type RedisAnalysisUsecase struct {
	repo RedisAnalysisRepo
	log  *log.Helper
}

// NewRedisAnalysisUsecase 创建Redis分析用例
func NewRedisAnalysisUsecase(repo RedisAnalysisRepo, logger log.Logger) *RedisAnalysisUsecase {
	return &RedisAnalysisUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// RedisHotKeyTask Redis热key分析任务
type RedisHotKeyTask struct {
	ID          uint64
	InstanceID  uint64
	TaskType    string // hotkey, bigkey
	Status      int    // 0-待执行, 1-执行中, 2-完成, 3-失败
	Result      map[string]interface{}
	SubmitterID uint64
	StartedAt   *time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RedisAnalysisRepo Redis分析仓储接口
type RedisAnalysisRepo interface {
	CreateTask(ctx context.Context, task *RedisHotKeyTask) (*RedisHotKeyTask, error)
	UpdateTask(ctx context.Context, task *RedisHotKeyTask) error
	GetTaskByID(ctx context.Context, id uint64) (*RedisHotKeyTask, error)
	ListTasks(ctx context.Context, instanceID uint64, page, pageSize int) ([]*RedisHotKeyTask, int64, error)
}

// AnalyzeHotKeys 分析热key
// 支持时间窗口统计和访问频率阈值
func (uc *RedisAnalysisUsecase) AnalyzeHotKeys(ctx context.Context, instance *InstanceInfo, timeWindowSeconds int, threshold int64) ([]*HotKeyInfo, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", instance.Host, instance.Port),
		Password: instance.Password,
		DB:       0,
	})
	defer rdb.Close()

	// 默认时间窗口为60秒，阈值100
	if timeWindowSeconds <= 0 {
		timeWindowSeconds = 60
	}
	if threshold <= 0 {
		threshold = 100
	}

	keyAccessCount := make(map[string]int64)
	keyAccessTime := make(map[string]time.Time)
	
	// 使用SCAN采样方式统计（避免MONITOR影响性能）
	iter := rdb.Scan(ctx, 0, "*", 100).Iterator()
	sampleCount := 0
	maxSamples := 1000
	
	for iter.Next(ctx) && sampleCount < maxSamples {
		key := iter.Val()
		// 使用OBJECT IDLETIME估算访问频率（空闲时间越短，访问越频繁）
		idleTime, err := rdb.ObjectIdleTime(ctx, key).Result()
		if err == nil && idleTime > 0 {
			// 空闲时间转换为访问频率估算（秒级）
			// 如果空闲时间很短，说明访问频繁
			accessFreq := int64(time.Hour.Seconds() / float64(idleTime.Seconds()))
			if accessFreq > threshold {
				keyAccessCount[key] = accessFreq
				keyAccessTime[key] = time.Now().Add(-idleTime)
			}
		}
		sampleCount++
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}
	
	// 筛选热key（超过阈值）
	var hotKeys []*HotKeyInfo
	for key, count := range keyAccessCount {
		if count >= threshold {
			size, _ := rdb.MemoryUsage(ctx, key).Result()
			keyType, _ := rdb.Type(ctx, key).Result()
			lastAccess := keyAccessTime[key]
			if lastAccess.IsZero() {
				lastAccess = time.Now()
			}
			
			hotKeys = append(hotKeys, &HotKeyInfo{
				Key:         key,
				Type:        keyType,
				Size:        size,
				AccessCount: count,
				LastAccess:  lastAccess,
			})
		}
	}
	
	// 按访问频率排序
	sort.Slice(hotKeys, func(i, j int) bool {
		return hotKeys[i].AccessCount > hotKeys[j].AccessCount
	})
	
	// 返回Top 100
	if len(hotKeys) > 100 {
		return hotKeys[:100], nil
	}
	return hotKeys, nil
}

// AnalyzeBigKeys 分析Bigkey
func (uc *RedisAnalysisUsecase) AnalyzeBigKeys(ctx context.Context, instance *InstanceInfo, threshold int64) ([]*BigKeyInfo, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", instance.Host, instance.Port),
		Password: instance.Password,
		DB:       0,
	})
	defer rdb.Close()

	var bigKeys []*BigKeyInfo
	
	// 扫描所有key
	iter := rdb.Scan(ctx, 0, "*", 0).Iterator()
	
	for iter.Next(ctx) {
		key := iter.Val()
		size, err := rdb.MemoryUsage(ctx, key).Result()
		if err != nil {
			continue
		}
		
		if size >= threshold {
			keyType, _ := rdb.Type(ctx, key).Result()
			ttl, _ := rdb.TTL(ctx, key).Result()
			
			bigKeys = append(bigKeys, &BigKeyInfo{
				Key:  key,
				Type: keyType,
				Size: size,
				TTL:  ttl,
			})
		}
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}
	
	return bigKeys, nil
}

// HotKeyInfo 热key信息
type HotKeyInfo struct {
	Key         string
	Type        string
	Size        int64
	AccessCount int64
	LastAccess  time.Time
}

// BigKeyInfo Bigkey信息
type BigKeyInfo struct {
	Key  string
	Type string
	Size int64
	TTL  time.Duration
}

// CreateTask 创建分析任务
func (uc *RedisAnalysisUsecase) CreateTask(ctx context.Context, task *RedisHotKeyTask) (*RedisHotKeyTask, error) {
	return uc.repo.CreateTask(ctx, task)
}

// UpdateTask 更新分析任务
func (uc *RedisAnalysisUsecase) UpdateTask(ctx context.Context, task *RedisHotKeyTask) error {
	return uc.repo.UpdateTask(ctx, task)
}

// GetTaskByID 获取分析任务
func (uc *RedisAnalysisUsecase) GetTaskByID(ctx context.Context, id uint64) (*RedisHotKeyTask, error) {
	return uc.repo.GetTaskByID(ctx, id)
}

// ListTasks 列出分析任务
func (uc *RedisAnalysisUsecase) ListTasks(ctx context.Context, instanceID uint64, page, pageSize int) ([]*RedisHotKeyTask, int64, error) {
	return uc.repo.ListTasks(ctx, instanceID, page, pageSize)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
