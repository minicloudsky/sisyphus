package biz

import (
	"context"
	"time"
)

// QueryHistory 查询历史记录
type QueryHistory struct {
	ID           uint64
	UserID       uint64
	InstanceID   uint64
	DatabaseName string
	SQLText      string
	ExecuteTime  float64 // 执行耗时(秒)
	AffectedRows int
	ResultCount  int
	CreatedAt    time.Time
}

// QueryHistoryRepo 查询历史仓储接口
type QueryHistoryRepo interface {
	Create(ctx context.Context, history *QueryHistory) (*QueryHistory, error)
	List(ctx context.Context, userID uint64, page, pageSize int, instanceID *uint64) ([]*QueryHistory, int64, error)
	GetByID(ctx context.Context, id uint64) (*QueryHistory, error)
	Delete(ctx context.Context, id uint64) error
}

// QueryHistoryUsecase 查询历史用例
type QueryHistoryUsecase struct {
	repo QueryHistoryRepo
}

// NewQueryHistoryUsecase 创建查询历史用例
func NewQueryHistoryUsecase(repo QueryHistoryRepo) *QueryHistoryUsecase {
	return &QueryHistoryUsecase{repo: repo}
}

// CreateHistory 创建查询历史
func (uc *QueryHistoryUsecase) CreateHistory(ctx context.Context, history *QueryHistory) (*QueryHistory, error) {
	return uc.repo.Create(ctx, history)
}

// ListHistory 列出查询历史
func (uc *QueryHistoryUsecase) ListHistory(ctx context.Context, userID uint64, page, pageSize int, instanceID *uint64) ([]*QueryHistory, int64, error) {
	return uc.repo.List(ctx, userID, page, pageSize, instanceID)
}

// GetHistory 获取查询历史
func (uc *QueryHistoryUsecase) GetHistory(ctx context.Context, id uint64) (*QueryHistory, error) {
	return uc.repo.GetByID(ctx, id)
}

// DeleteHistory 删除查询历史
func (uc *QueryHistoryUsecase) DeleteHistory(ctx context.Context, id uint64) error {
	return uc.repo.Delete(ctx, id)
}
