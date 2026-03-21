package data

import (
	"context"
	"time"

	"sisyphus/server/internal/biz"
)

type SQLRollback struct {
	ID             uint64  `gorm:"column:id;primaryKey;autoIncrement"`
	OrderID        uint64  `gorm:"column:order_id;not null;index:idx_order_id"`
	OrderItemID    *uint64 `gorm:"column:order_item_id;index:idx_order_item_id"`
	BackupDBName   string  `gorm:"column:backup_db_name;type:varchar(100);not null"`
	OpidTime       string  `gorm:"column:opid_time;type:varchar(50);not null;index:idx_opid_time"`
	TableNameField string  `gorm:"column:table_name;type:varchar(100);not null"`
	RollbackSQL    string  `gorm:"column:rollback_sql;type:text;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (SQLRollback) TableName() string {
	return "sisyphus_sql_order_rollback"
}

type rollbackRepo struct {
	data *Data
}

// NewRollbackRepo 创建回滚记录仓储
func NewRollbackRepo(data *Data) biz.RollbackRepo {
	return &rollbackRepo{data: data}
}

func (r *rollbackRepo) Create(ctx context.Context, rollback *biz.SQLRollback) error {
	rb := &SQLRollback{
		OrderID:        rollback.OrderID,
		OrderItemID:    rollback.OrderItemID,
		BackupDBName:   rollback.BackupDBName,
		OpidTime:       rollback.OpidTime,
		TableNameField: rollback.TableName,
		RollbackSQL:    rollback.RollbackSQL,
	}
	if err := r.data.db.WithContext(ctx).Create(rb).Error; err != nil {
		return err
	}
	rollback.ID = rb.ID
	rollback.CreatedAt = rb.CreatedAt
	return nil
}

func (r *rollbackRepo) GetByOrderID(ctx context.Context, orderID uint64) ([]*biz.SQLRollback, error) {
	var rollbacks []SQLRollback
	if err := r.data.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at ASC").
		Find(&rollbacks).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.SQLRollback, len(rollbacks))
	for i, rb := range rollbacks {
		result[i] = toBizRollback(&rb)
	}
	return result, nil
}

func (r *rollbackRepo) GetByOrderItemID(ctx context.Context, orderItemID uint64) ([]*biz.SQLRollback, error) {
	var rollbacks []SQLRollback
	if err := r.data.db.WithContext(ctx).
		Where("order_item_id = ?", orderItemID).
		Order("created_at ASC").
		Find(&rollbacks).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.SQLRollback, len(rollbacks))
	for i, rb := range rollbacks {
		result[i] = toBizRollback(&rb)
	}
	return result, nil
}

func toBizRollback(rb *SQLRollback) *biz.SQLRollback {
	return &biz.SQLRollback{
		ID:           rb.ID,
		OrderID:      rb.OrderID,
		OrderItemID:  rb.OrderItemID,
		BackupDBName: rb.BackupDBName,
		OpidTime:     rb.OpidTime,
		TableName:    rb.TableNameField,
		RollbackSQL:  rb.RollbackSQL,
		CreatedAt:    rb.CreatedAt,
	}
}
