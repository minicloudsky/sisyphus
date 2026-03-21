package data

import (
	"context"
	"time"

	"sisyphus/server/internal/biz"
)

type queryHistoryRepo struct {
	data *Data
}

// NewQueryHistoryRepo 创建查询历史仓储
func NewQueryHistoryRepo(data *Data) biz.QueryHistoryRepo {
	return &queryHistoryRepo{data: data}
}

// QueryHistory 查询历史记录表
type QueryHistory struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       uint64    `gorm:"column:user_id;not null;index:idx_user"`
	InstanceID   uint64    `gorm:"column:instance_id;not null;index:idx_instance"`
	DatabaseName string    `gorm:"column:database_name;type:varchar(100);not null"`
	SQLText      string    `gorm:"column:sql_text;type:text;not null"`
	ExecuteTime  float64   `gorm:"column:execute_time;type:decimal(10,2);default:0;comment:执行耗时(秒)"`
	AffectedRows int       `gorm:"column:affected_rows;default:0"`
	ResultCount  int       `gorm:"column:result_count;default:0;comment:结果数量"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP;index:idx_created_at"`
}

func (QueryHistory) TableName() string {
	return "sisyphus_query_history"
}

// toBizQueryHistory 转换为业务对象
func toBizQueryHistory(h *QueryHistory) *biz.QueryHistory {
	return &biz.QueryHistory{
		ID:           h.ID,
		UserID:       h.UserID,
		InstanceID:   h.InstanceID,
		DatabaseName: h.DatabaseName,
		SQLText:      h.SQLText,
		ExecuteTime:  h.ExecuteTime,
		AffectedRows: h.AffectedRows,
		ResultCount:  h.ResultCount,
		CreatedAt:    h.CreatedAt,
	}
}

// Create 创建查询历史
func (r *queryHistoryRepo) Create(ctx context.Context, history *biz.QueryHistory) (*biz.QueryHistory, error) {
	h := &QueryHistory{
		UserID:       history.UserID,
		InstanceID:   history.InstanceID,
		DatabaseName: history.DatabaseName,
		SQLText:      history.SQLText,
		ExecuteTime:  history.ExecuteTime,
		AffectedRows: history.AffectedRows,
		ResultCount:  history.ResultCount,
		CreatedAt:    history.CreatedAt,
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now()
	}

	if err := r.data.db.WithContext(ctx).Create(h).Error; err != nil {
		return nil, err
	}
	return toBizQueryHistory(h), nil
}

// List 列出查询历史
func (r *queryHistoryRepo) List(ctx context.Context, userID uint64, page, pageSize int, instanceID *uint64) ([]*biz.QueryHistory, int64, error) {
	var histories []QueryHistory
	var total int64

	query := r.data.db.WithContext(ctx).Model(&QueryHistory{}).Where("user_id = ?", userID)

	if instanceID != nil {
		query = query.Where("instance_id = ?", *instanceID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.QueryHistory, len(histories))
	for i, h := range histories {
		result[i] = toBizQueryHistory(&h)
	}
	return result, total, nil
}

// GetByID 根据ID获取查询历史
func (r *queryHistoryRepo) GetByID(ctx context.Context, id uint64) (*biz.QueryHistory, error) {
	var h QueryHistory
	if err := r.data.db.WithContext(ctx).First(&h, id).Error; err != nil {
		return nil, err
	}
	return toBizQueryHistory(&h), nil
}

// Delete 删除查询历史
func (r *queryHistoryRepo) Delete(ctx context.Context, id uint64) error {
	return r.data.db.WithContext(ctx).Delete(&QueryHistory{}, id).Error
}
