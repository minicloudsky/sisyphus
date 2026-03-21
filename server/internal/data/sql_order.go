package data

import (
	"context"
	"time"

	"sisyphus/server/internal/biz"
)

type sqlOrderRepo struct {
	data *Data
}

// NewSQLOrderRepo 创建SQL工单仓储
func NewSQLOrderRepo(data *Data) biz.SQLOrderRepo {
	return &sqlOrderRepo{data: data}
}

func (r *sqlOrderRepo) Create(ctx context.Context, order *biz.SQLOrder) (*biz.SQLOrder, error) {
	o := &SQLOrder{
		WorkID:        order.WorkID,
		InstanceID:    order.InstanceID,
		DatabaseName:  order.DatabaseName,
		OrderType:     int8(order.OrderType),
		Status:        int8(order.Status),
		SQLContent:    order.SQLContent,
		Remark:        order.Remark,
		SubmitterID:   order.SubmitterID,
		DelayMinutes:  order.DelayMinutes,
		NeedBackup:    order.NeedBackup,
		ExecuteMethod: int8(order.ExecuteMethod),
		RiskLevel:     int8(order.RiskLevel),
	}
	if err := r.data.db.WithContext(ctx).Create(o).Error; err != nil {
		return nil, err
	}
	return toBizSQLOrder(o), nil
}

func (r *sqlOrderRepo) Update(ctx context.Context, order *biz.SQLOrder) error {
	updates := map[string]interface{}{
		"status":        order.Status,
		"auditor_id":    order.AuditorID,
		"executor_id":   order.ExecutorID,
		"reject_reason": order.RejectReason,
		"updated_at":    time.Now(),
	}
	if order.ExecuteTime != nil {
		updates["execute_time"] = *order.ExecuteTime
	}
	return r.data.db.WithContext(ctx).Model(&SQLOrder{}).Where("id = ?", order.ID).Updates(updates).Error
}

func (r *sqlOrderRepo) GetByID(ctx context.Context, id uint64) (*biz.SQLOrder, error) {
	var order SQLOrder
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&order).Error; err != nil {
		return nil, err
	}
	return toBizSQLOrder(&order), nil
}

func (r *sqlOrderRepo) GetByWorkID(ctx context.Context, workID string) (*biz.SQLOrder, error) {
	var order SQLOrder
	if err := r.data.db.WithContext(ctx).Where("work_id = ?", workID).First(&order).Error; err != nil {
		return nil, err
	}
	return toBizSQLOrder(&order), nil
}

func (r *sqlOrderRepo) List(ctx context.Context, page, pageSize int, status *int, submitterID *uint64) ([]*biz.SQLOrder, int64, error) {
	var orders []SQLOrder
	var total int64

	query := r.data.db.WithContext(ctx).Model(&SQLOrder{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if submitterID != nil {
		query = query.Where("submitter_id = ?", *submitterID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.SQLOrder, len(orders))
	for i, o := range orders {
		result[i] = toBizSQLOrder(&o)
	}
	return result, total, nil
}

func (r *sqlOrderRepo) CreateItem(ctx context.Context, item *biz.SQLOrderItem) error {
	i := &SQLOrderItem{
		OrderID:      item.OrderID,
		Sequence:     item.Sequence,
		SQLText:      item.SQLText,
		Status:       int8(item.Status),
		ExecuteTime:  item.ExecuteTime,
		AffectedRows: item.AffectedRows,
		ErrorMessage: item.ErrorMessage,
	}
	return r.data.db.WithContext(ctx).Create(i).Error
}

func (r *sqlOrderRepo) UpdateItem(ctx context.Context, item *biz.SQLOrderItem) error {
	updates := map[string]interface{}{
		"status":        item.Status,
		"execute_time":  item.ExecuteTime,
		"affected_rows": item.AffectedRows,
		"error_message": item.ErrorMessage,
	}
	// 数据库表中没有 execute_type 和 result_text 字段，不更新
	return r.data.db.WithContext(ctx).Model(&SQLOrderItem{}).Where("id = ?", item.ID).Updates(updates).Error
}

func (r *sqlOrderRepo) GetItemsByOrderID(ctx context.Context, orderID uint64) ([]*biz.SQLOrderItem, error) {
	var items []SQLOrderItem
	if err := r.data.db.WithContext(ctx).Where("order_id = ?", orderID).Order("sequence ASC").Find(&items).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.SQLOrderItem, len(items))
	for i, item := range items {
		result[i] = toBizSQLOrderItem(&item)
	}
	return result, nil
}

func (r *sqlOrderRepo) DeleteItem(ctx context.Context, itemID uint64) error {
	return r.data.db.WithContext(ctx).Delete(&SQLOrderItem{}, itemID).Error
}

func (r *sqlOrderRepo) CreateReviewRecord(ctx context.Context, record *biz.SQLReviewRecord) error {
	rec := &SQLReviewRecord{
		OrderID:      record.OrderID,
		Sequence:     record.Sequence,
		// Stage 字段可能不存在于数据库表中，使用 Omit 排除
		ErrorLevel:   int8(record.ErrorLevel),
		StageStatus:  record.StageStatus,
		ErrorMessage: record.ErrorMessage,
		SQLText:      record.SQLText,
		AffectedRows: record.AffectedRows,
		BackupDBName: record.BackupDBName,
		ExecuteTime:  record.ExecuteTime,
		SQLHash:      record.SQLHash,
	}
	return r.data.db.WithContext(ctx).Create(rec).Error
}

func (r *sqlOrderRepo) GetReviewRecordsByOrderID(ctx context.Context, orderID uint64) ([]*biz.SQLReviewRecord, error) {
	var records []SQLReviewRecord
	if err := r.data.db.WithContext(ctx).Where("order_id = ?", orderID).Find(&records).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.SQLReviewRecord, len(records))
	for i, rec := range records {
		result[i] = toBizSQLReviewRecord(&rec)
	}
	return result, nil
}

// SQLOrder 数据库模型
type SQLOrder struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	WorkID        string     `gorm:"column:work_id;type:varchar(50);uniqueIndex;not null"`
	InstanceID    uint64     `gorm:"column:instance_id;not null;index"`
	DatabaseName  string     `gorm:"column:database_name;type:varchar(100);not null"`
	OrderType     int8       `gorm:"column:order_type;type:tinyint;not null"`
	Status        int8       `gorm:"column:status;type:tinyint;default:0;index"`
	SQLContent    string     `gorm:"column:sql_content;type:text;not null"`
	Remark        string     `gorm:"column:remark;type:text"`
	SubmitterID   uint64     `gorm:"column:submitter_id;not null;index"`
	AuditorID     uint64     `gorm:"column:auditor_id;default:0"`
	ExecutorID    uint64     `gorm:"column:executor_id;default:0"`
	RejectReason  string     `gorm:"column:reject_reason;type:text"`
	DelayMinutes  int        `gorm:"column:delay_minutes;default:0"`
	// IMPORTANT: keep `false` as a real value.
	// Previously the `default:true` tag caused GORM to treat `false` as "unset" and
	// fall back to the DB default (need_backup=1), breaking GoInception backup mode.
	NeedBackup    bool       `gorm:"column:need_backup;type:tinyint;not null"`
	ExecuteMethod int8       `gorm:"column:execute_method;type:tinyint;default:0"`
	RiskLevel     int8       `gorm:"column:risk_level;type:tinyint;default:0"`
	ExecuteTime   *time.Time `gorm:"column:execute_time;default:null"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;index"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (SQLOrder) TableName() string {
	return "sisyphus_sql_orders"
}

type SQLOrderItem struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	OrderID      uint64    `gorm:"column:order_id;not null;index"`
	Sequence     int       `gorm:"column:sequence;not null"`
	SQLText      string    `gorm:"column:sql_text;type:text;not null"`
	Status       int8      `gorm:"column:status;type:tinyint;default:0"`
	ExecuteType  string    `gorm:"column:execute_type;type:varchar(50);default:''"`
	ExecuteTime  float64   `gorm:"column:execute_time;type:decimal(10,2);default:0"`
	AffectedRows int       `gorm:"column:affected_rows;default:0"`
	ResultText   string    `gorm:"column:result_text;type:text"`
	ErrorMessage string    `gorm:"column:error_message;type:text"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (SQLOrderItem) TableName() string {
	return "sisyphus_sql_order_items"
}

type SQLReviewRecord struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	OrderID      uint64    `gorm:"column:order_id;not null;index"`
	Sequence     string    `gorm:"column:sequence;type:varchar(50)"`
	Stage        string    `gorm:"column:stage;type:varchar(20)"`
	ErrorLevel   int8      `gorm:"column:error_level;type:tinyint"`
	StageStatus  string    `gorm:"column:stage_status;type:varchar(50)"`
	ErrorMessage string    `gorm:"column:error_message;type:text"`
	SQLText      string    `gorm:"column:sql_text;type:text"`
	AffectedRows int       `gorm:"column:affected_rows;default:0"`
	BackupDBName string    `gorm:"column:backup_dbname;type:varchar(100)"`
	ExecuteTime  string    `gorm:"column:execute_time;type:varchar(50)"`
	SQLHash      string    `gorm:"column:sql_hash;type:varchar(64)"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (SQLReviewRecord) TableName() string {
	return "sisyphus_sql_review_records"
}

func toBizSQLOrder(o *SQLOrder) *biz.SQLOrder {
	return &biz.SQLOrder{
		ID:            o.ID,
		WorkID:        o.WorkID,
		InstanceID:    o.InstanceID,
		DatabaseName:  o.DatabaseName,
		OrderType:     int(o.OrderType),
		Status:        int(o.Status),
		SQLContent:    o.SQLContent,
		Remark:        o.Remark,
		SubmitterID:   o.SubmitterID,
		AuditorID:     o.AuditorID,
		ExecutorID:    o.ExecutorID,
		RejectReason:  o.RejectReason,
		DelayMinutes:  o.DelayMinutes,
		NeedBackup:    o.NeedBackup,
		ExecuteMethod: int(o.ExecuteMethod),
		RiskLevel:     int(o.RiskLevel),
		ExecuteTime:   o.ExecuteTime,
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
	}
}

func toBizSQLOrderItem(item *SQLOrderItem) *biz.SQLOrderItem {
	return &biz.SQLOrderItem{
		ID:           item.ID,
		OrderID:      item.OrderID,
		Sequence:     item.Sequence,
		SQLText:      item.SQLText,
		Status:       int(item.Status),
		ExecuteType:  item.ExecuteType,
		ExecuteTime:  item.ExecuteTime,
		AffectedRows: item.AffectedRows,
		ResultText:   item.ResultText,
		ErrorMessage: item.ErrorMessage,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func toBizSQLReviewRecord(rec *SQLReviewRecord) *biz.SQLReviewRecord {
	return &biz.SQLReviewRecord{
		ID:           rec.ID,
		OrderID:      rec.OrderID,
		ItemID:       0, // 数据库表中没有item_id字段，默认为0
		Sequence:     rec.Sequence,
		Stage:        rec.Stage,
		ErrorLevel:   int(rec.ErrorLevel),
		StageStatus:  rec.StageStatus,
		ErrorMessage: rec.ErrorMessage,
		SQLText:      rec.SQLText,
		AffectedRows: rec.AffectedRows,
		BackupDBName: rec.BackupDBName,
		ExecuteTime:  rec.ExecuteTime,
		SQLHash:      rec.SQLHash,
		CreatedAt:    rec.CreatedAt,
	}
}
