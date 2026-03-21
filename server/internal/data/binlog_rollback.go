package data

import (
	"context"
	"encoding/json"
	"time"

	"sisyphus/server/internal/biz"
)

type binlogRollbackRepo struct {
	data *Data
}

// NewBinlogRollbackRepo 创建Binlog回滚仓储
func NewBinlogRollbackRepo(data *Data) biz.BinlogRollbackRepo {
	return &binlogRollbackRepo{data: data}
}

func (r *binlogRollbackRepo) CreateOrder(ctx context.Context, order *biz.BinlogRollbackOrder) (*biz.BinlogRollbackOrder, error) {
	tableListJSON, _ := json.Marshal(order.TableList)
	o := &BinlogRollbackOrder{
		WorkID:          order.WorkID,
		InstanceID:      order.InstanceID,
		DatabaseName:    order.DatabaseName,
		StartTime:       order.StartTime,
		StopTime:        order.StopTime,
		BinlogFile:      order.BinlogFile,
		TableList:       string(tableListJSON),
		Status:          int8(order.Status),
		RollbackSQLCount: order.RollbackSQLCount,
		SubmitterID:     order.SubmitterID,
	}
	if err := r.data.db.WithContext(ctx).Create(o).Error; err != nil {
		return nil, err
	}
	return toBizBinlogRollbackOrder(o), nil
}

func (r *binlogRollbackRepo) UpdateOrder(ctx context.Context, order *biz.BinlogRollbackOrder) error {
	updates := map[string]interface{}{
		"status":            order.Status,
		"rollback_sql_count": order.RollbackSQLCount,
		"updated_at":        time.Now(),
	}
	if order.BinlogFile != "" {
		updates["binlog_file"] = order.BinlogFile
	}
	if len(order.TableList) > 0 {
		tableListJSON, _ := json.Marshal(order.TableList)
		updates["table_list"] = string(tableListJSON)
	}
	return r.data.db.WithContext(ctx).Model(&BinlogRollbackOrder{}).Where("id = ?", order.ID).Updates(updates).Error
}

func (r *binlogRollbackRepo) GetOrderByID(ctx context.Context, id uint64) (*biz.BinlogRollbackOrder, error) {
	var order BinlogRollbackOrder
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&order).Error; err != nil {
		return nil, err
	}
	return toBizBinlogRollbackOrder(&order), nil
}

func (r *binlogRollbackRepo) GetOrderByWorkID(ctx context.Context, workID string) (*biz.BinlogRollbackOrder, error) {
	var order BinlogRollbackOrder
	if err := r.data.db.WithContext(ctx).Where("work_id = ?", workID).First(&order).Error; err != nil {
		return nil, err
	}
	return toBizBinlogRollbackOrder(&order), nil
}

func (r *binlogRollbackRepo) ListOrders(ctx context.Context, page, pageSize int) ([]*biz.BinlogRollbackOrder, int64, error) {
	var orders []BinlogRollbackOrder
	var total int64

	if err := r.data.db.WithContext(ctx).Model(&BinlogRollbackOrder{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.data.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.BinlogRollbackOrder, len(orders))
	for i, o := range orders {
		result[i] = toBizBinlogRollbackOrder(&o)
	}
	return result, total, nil
}

func (r *binlogRollbackRepo) CreateRollbackSQL(ctx context.Context, sql *biz.BinlogRollbackSQL) error {
	s := &BinlogRollbackSQL{
		OrderID:       sql.OrderID,
		TableNameField: sql.TableName,
		RollbackSQL:   sql.RollbackSQL,
		OriginalSQL:   sql.OriginalSQL,
	}
	return r.data.db.WithContext(ctx).Create(s).Error
}

func (r *binlogRollbackRepo) GetRollbackSQLsByOrderID(ctx context.Context, orderID uint64) ([]*biz.BinlogRollbackSQL, error) {
	var sqls []BinlogRollbackSQL
	if err := r.data.db.WithContext(ctx).Where("order_id = ?", orderID).Find(&sqls).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.BinlogRollbackSQL, len(sqls))
	for i, s := range sqls {
		result[i] = toBizBinlogRollbackSQL(&s)
	}
	return result, nil
}

// BinlogRollbackOrder 数据库模型
type BinlogRollbackOrder struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	WorkID           string    `gorm:"column:work_id;type:varchar(50);uniqueIndex;not null"`
	InstanceID       uint64    `gorm:"column:instance_id;not null;index"`
	DatabaseName     string    `gorm:"column:database_name;type:varchar(100);not null"`
	StartTime        time.Time `gorm:"column:start_time;not null"`
	StopTime         time.Time `gorm:"column:stop_time;not null"`
	BinlogFile       string    `gorm:"column:binlog_file;type:varchar(255);default:''"`
	TableList        string    `gorm:"column:table_list;type:text"`
	Status           int8      `gorm:"column:status;type:tinyint;default:0"`
	RollbackSQLCount int       `gorm:"column:rollback_sql_count;default:0"`
	SubmitterID      uint64    `gorm:"column:submitter_id;not null;index"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (BinlogRollbackOrder) TableName() string {
	return "sisyphus_binlog_rollback_orders"
}

type BinlogRollbackSQL struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	OrderID        uint64    `gorm:"column:order_id;not null;index"`
	TableNameField string    `gorm:"column:table_name;type:varchar(100);not null"`
	RollbackSQL    string    `gorm:"column:rollback_sql;type:text;not null"`
	OriginalSQL    string    `gorm:"column:original_sql;type:text"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (BinlogRollbackSQL) TableName() string {
	return "sisyphus_binlog_rollback_sqls"
}

func toBizBinlogRollbackOrder(o *BinlogRollbackOrder) *biz.BinlogRollbackOrder {
	var tableList []string
	json.Unmarshal([]byte(o.TableList), &tableList)
	
	return &biz.BinlogRollbackOrder{
		ID:              o.ID,
		WorkID:          o.WorkID,
		InstanceID:      o.InstanceID,
		DatabaseName:    o.DatabaseName,
		StartTime:       o.StartTime,
		StopTime:        o.StopTime,
		BinlogFile:      o.BinlogFile,
		TableList:       tableList,
		Status:          int(o.Status),
		RollbackSQLCount: o.RollbackSQLCount,
		SubmitterID:     o.SubmitterID,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
}

func toBizBinlogRollbackSQL(s *BinlogRollbackSQL) *biz.BinlogRollbackSQL {
	return &biz.BinlogRollbackSQL{
		ID:          s.ID,
		OrderID:     s.OrderID,
		TableName:   s.TableNameField,
		RollbackSQL: s.RollbackSQL,
		OriginalSQL: s.OriginalSQL,
		CreatedAt:   s.CreatedAt,
	}
}
