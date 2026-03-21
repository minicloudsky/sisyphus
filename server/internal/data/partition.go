package data

import (
	"context"
	"time"

	"sisyphus/server/internal/biz"
)

// partitionRepo 分区仓储实现
type partitionRepo struct {
	data *Data
}

// NewPartitionRepo 创建分区仓储
func NewPartitionRepo(data *Data) biz.PartitionRepo {
	return &partitionRepo{data: data}
}

// PartitionTableInfo 分区表信息数据库模型
type PartitionTableInfo struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	InstanceID       uint64    `gorm:"column:instance_id;not null;index"`
	InstanceName     string    `gorm:"column:instance_name;type:varchar(100);not null"`
	ConnectionString string    `gorm:"column:connection_string;type:varchar(512);not null"`
	DatabaseName     string    `gorm:"column:database_name;type:varchar(100);not null"`
	TableNameField   string    `gorm:"column:table_name;type:varchar(100);not null"`
	PartType         string    `gorm:"column:part_type;type:varchar(32)"`
	PartKey          string    `gorm:"column:part_key;type:varchar(255)"`
	PartCycle        string    `gorm:"column:part_cycle;type:varchar(10)"`
	PartNum          int       `gorm:"column:part_num;default:0"`
	PartMaxValue     string    `gorm:"column:part_max_value;type:varchar(128)"`
	PartMaxValueDay  string    `gorm:"column:part_max_value_day;type:varchar(16);default:'未到期'"`
	HaveMaxValue     string    `gorm:"column:have_max_value;type:varchar(8);default:'否'"`
	TableSize        float64   `gorm:"column:table_size;type:decimal(12,2);default:0"`
	TableRows        int64     `gorm:"column:table_rows;default:0"`
	MaxPartRows      int64     `gorm:"column:max_part_rows;default:0"`
	IsCrontab        string    `gorm:"column:is_crontab;type:varchar(8);default:'否'"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (PartitionTableInfo) TableName() string {
	return "sisyphus_partition_table_info"
}

// PartitionOperationRecord 分区操作记录数据库模型
type PartitionOperationRecord struct {
	ID               uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	WorkID           string     `gorm:"column:work_id;type:varchar(50);not null;index"`
	InstanceID       uint64     `gorm:"column:instance_id;not null;index"`
	InstanceName     string     `gorm:"column:instance_name;type:varchar(100);not null"`
	ConnectionString string     `gorm:"column:connection_string;type:varchar(512);not null"`
	TableSchema      string     `gorm:"column:table_schema;type:varchar(100);not null"`
	TableNameField   string     `gorm:"column:table_name;type:varchar(100);not null"`
	OperateType      int8       `gorm:"column:operate_type;type:tinyint;not null;index"`
	ExeStatus        int8       `gorm:"column:exe_status;type:tinyint;default:1;index"`
	CheckStatus      int8       `gorm:"column:check_status;type:tinyint;default:0"`
	ExePct           string     `gorm:"column:exe_pct;type:varchar(10)"`
	ExeResult        string     `gorm:"column:exe_result;type:text"`
	ExeCmd           string     `gorm:"column:exe_cmd;type:varchar(1000)"`
	WhereCondition   string     `gorm:"column:where_condition;type:varchar(500)"`
	AddUser          string     `gorm:"column:add_user;type:varchar(50)"`
	UpdateUser       string     `gorm:"column:update_user;type:varchar(50)"`
	ExeTime          *time.Time `gorm:"column:exe_time;default:null"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime;index"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (PartitionOperationRecord) TableName() string {
	return "sisyphus_partition_operation_records"
}

// UpsertPartitionTableInfo 插入或更新分区表信息
func (r *partitionRepo) UpsertPartitionTableInfo(ctx context.Context, info *biz.PartitionTableInfo) error {
	pt := &PartitionTableInfo{
		InstanceID:       info.InstanceID,
		InstanceName:     info.InstanceName,
		ConnectionString: info.ConnectionString,
		DatabaseName:     info.DatabaseName,
		TableNameField:   info.TableName,
		PartType:         info.PartType,
		PartKey:          info.PartKey,
		PartCycle:        info.PartCycle,
		PartNum:          info.PartNum,
		PartMaxValue:     info.PartMaxValue,
		PartMaxValueDay:  info.PartMaxValueDay,
		HaveMaxValue:     info.HaveMaxValue,
		TableSize:        info.TableSize,
		TableRows:        info.TableRows,
		MaxPartRows:      info.MaxPartRows,
		IsCrontab:        info.IsCrontab,
	}

	return r.data.db.WithContext(ctx).
		Where("instance_id = ? AND database_name = ? AND table_name = ?", 
			info.InstanceID, info.DatabaseName, info.TableName).
		Assign(pt).
		FirstOrCreate(pt).Error
}

// GetPartitionTableInfo 获取分区表信息
func (r *partitionRepo) GetPartitionTableInfo(ctx context.Context, instanceID uint64, databaseName, tableName string) (*biz.PartitionTableInfo, error) {
	var pt PartitionTableInfo
	if err := r.data.db.WithContext(ctx).
		Where("instance_id = ? AND database_name = ? AND table_name = ?", instanceID, databaseName, tableName).
		First(&pt).Error; err != nil {
		return nil, err
	}

	return toBizPartitionTableInfo(&pt), nil
}

// ListExpiredPartitionTables 列出已到期的分区表
func (r *partitionRepo) ListExpiredPartitionTables(ctx context.Context) ([]*biz.PartitionTableInfo, error) {
	var pts []PartitionTableInfo
	if err := r.data.db.WithContext(ctx).
		Where("part_maxvalue_day = ?", "已到期").
		Find(&pts).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.PartitionTableInfo, len(pts))
	for i, pt := range pts {
		result[i] = toBizPartitionTableInfo(&pt)
	}
	return result, nil
}

// ListPartitionTables 列出分区表
func (r *partitionRepo) ListPartitionTables(ctx context.Context, instanceID uint64) ([]*biz.PartitionTableInfo, error) {
	var pts []PartitionTableInfo
	query := r.data.db.WithContext(ctx)
	if instanceID > 0 {
		query = query.Where("instance_id = ?", instanceID)
	}
	if err := query.Find(&pts).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.PartitionTableInfo, len(pts))
	for i, pt := range pts {
		result[i] = toBizPartitionTableInfo(&pt)
	}
	return result, nil
}

// CreateOperationRecord 创建操作记录
func (r *partitionRepo) CreateOperationRecord(ctx context.Context, record *biz.PartitionOperationRecord) error {
	rec := &PartitionOperationRecord{
		WorkID:          record.WorkID,
		InstanceID:      record.InstanceID,
		InstanceName:    record.InstanceName,
		ConnectionString: record.ConnectionString,
		TableSchema:     record.TableSchema,
		TableNameField:  record.TableName,
		OperateType:     int8(record.OperateType),
		ExeStatus:       int8(record.ExeStatus),
		CheckStatus:     int8(record.CheckStatus),
		ExePct:          record.ExePct,
		ExeResult:       record.ExeResult,
		ExeCmd:          record.ExeCmd,
		WhereCondition:  record.WhereCondition,
		AddUser:         record.AddUser,
		UpdateUser:      record.UpdateUser,
		ExeTime:         record.ExeTime,
	}
	return r.data.db.WithContext(ctx).Create(rec).Error
}

// UpdateOperationRecord 更新操作记录
func (r *partitionRepo) UpdateOperationRecord(ctx context.Context, record *biz.PartitionOperationRecord) error {
	updates := map[string]interface{}{
		"exe_status":   int8(record.ExeStatus),
		"check_status": int8(record.CheckStatus),
		"exe_pct":     record.ExePct,
		"exe_result":  record.ExeResult,
		"exe_cmd":     record.ExeCmd,
		"update_user": record.UpdateUser,
		"updated_at":  time.Now(),
	}
	if record.ExeTime != nil {
		updates["exe_time"] = record.ExeTime
	}
	return r.data.db.WithContext(ctx).
		Model(&PartitionOperationRecord{}).
		Where("work_id = ?", record.WorkID).
		Updates(updates).Error
}

// GetOperationRecordByWorkID 根据工单ID获取操作记录
func (r *partitionRepo) GetOperationRecordByWorkID(ctx context.Context, workID string) (*biz.PartitionOperationRecord, error) {
	var rec PartitionOperationRecord
	if err := r.data.db.WithContext(ctx).
		Where("work_id = ?", workID).
		Order("id DESC").
		First(&rec).Error; err != nil {
		return nil, err
	}

	return toBizPartitionOperationRecord(&rec), nil
}

// ListOperationRecords 列出操作记录
func (r *partitionRepo) ListOperationRecords(ctx context.Context, instanceID uint64, operateType int, page, pageSize int) ([]*biz.PartitionOperationRecord, int64, error) {
	var records []PartitionOperationRecord
	var total int64

	query := r.data.db.WithContext(ctx).Model(&PartitionOperationRecord{})
	if instanceID > 0 {
		query = query.Where("instance_id = ?", instanceID)
	}
	if operateType > 0 {
		query = query.Where("operate_type = ?", operateType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.PartitionOperationRecord, len(records))
	for i, rec := range records {
		result[i] = toBizPartitionOperationRecord(&rec)
	}
	return result, total, nil
}

// GetPartitionTableInfoByID 根据ID获取分区表信息
func (r *partitionRepo) GetPartitionTableInfoByID(ctx context.Context, id uint64) (*biz.PartitionTableInfo, error) {
	var pt PartitionTableInfo
	if err := r.data.db.WithContext(ctx).
		Where("id = ?", id).
		First(&pt).Error; err != nil {
		return nil, err
	}
	return toBizPartitionTableInfo(&pt), nil
}

// CreatePartitionTableInfo 创建分区表信息
func (r *partitionRepo) CreatePartitionTableInfo(ctx context.Context, info *biz.PartitionTableInfo) error {
	pt := &PartitionTableInfo{
		InstanceID:       info.InstanceID,
		InstanceName:     info.InstanceName,
		ConnectionString: info.ConnectionString,
		DatabaseName:     info.DatabaseName,
		TableNameField:   info.TableName,
		PartType:         info.PartType,
		PartKey:          info.PartKey,
		PartCycle:        info.PartCycle,
		PartNum:          info.PartNum,
		PartMaxValue:     info.PartMaxValue,
		PartMaxValueDay:  info.PartMaxValueDay,
		HaveMaxValue:     info.HaveMaxValue,
		TableSize:        info.TableSize,
		TableRows:        info.TableRows,
		MaxPartRows:      info.MaxPartRows,
		IsCrontab:        info.IsCrontab,
	}
	if err := r.data.db.WithContext(ctx).Create(pt).Error; err != nil {
		return err
	}
	info.ID = pt.ID
	return nil
}

// ListPartitionTablesPaged 分页列出分区表
func (r *partitionRepo) ListPartitionTablesPaged(ctx context.Context, instanceID uint64, page, pageSize int) ([]*biz.PartitionTableInfo, int64, error) {
	var pts []PartitionTableInfo
	var total int64

	query := r.data.db.WithContext(ctx).Model(&PartitionTableInfo{})
	if instanceID > 0 {
		query = query.Where("instance_id = ?", instanceID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&pts).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.PartitionTableInfo, len(pts))
	for i, pt := range pts {
		result[i] = toBizPartitionTableInfo(&pt)
	}
	return result, total, nil
}

// UpdatePartitionTableInfo 更新分区表信息
func (r *partitionRepo) UpdatePartitionTableInfo(ctx context.Context, info *biz.PartitionTableInfo) error {
	updates := map[string]interface{}{
		"instance_id":       info.InstanceID,
		"instance_name":     info.InstanceName,
		"connection_string": info.ConnectionString,
		"database_name":     info.DatabaseName,
		"table_name":        info.TableName,
		"part_type":         info.PartType,
		"part_key":          info.PartKey,
		"part_cycle":        info.PartCycle,
		"part_num":          info.PartNum,
		"part_maxvalue":     info.PartMaxValue,
		"part_maxvalue_day": info.PartMaxValueDay,
		"have_maxvalue":     info.HaveMaxValue,
		"table_size":        info.TableSize,
		"table_rows":        info.TableRows,
		"max_part_rows":     info.MaxPartRows,
		"is_crontab":        info.IsCrontab,
		"updated_at":        time.Now(),
	}
	return r.data.db.WithContext(ctx).
		Model(&PartitionTableInfo{}).
		Where("id = ?", info.ID).
		Updates(updates).Error
}

// DeletePartitionTableInfo 删除分区表信息
func (r *partitionRepo) DeletePartitionTableInfo(ctx context.Context, id uint64) error {
	return r.data.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&PartitionTableInfo{}).Error
}

// ListOperationRecordsByTableInfoID 根据分区表配置ID列出操作记录
func (r *partitionRepo) ListOperationRecordsByTableInfoID(ctx context.Context, tableInfoID uint64, page, pageSize int) ([]*biz.PartitionOperationRecord, int64, error) {
	var records []PartitionOperationRecord
	var total int64

	// 先取得分区表信息以获取 instance_id/table_schema/table_name，用于匹配操作记录
	var pt PartitionTableInfo
	if err := r.data.db.WithContext(ctx).Where("id = ?", tableInfoID).First(&pt).Error; err != nil {
		return nil, 0, err
	}

	query := r.data.db.WithContext(ctx).Model(&PartitionOperationRecord{}).
		Where("instance_id = ? AND table_schema = ? AND table_name = ?",
			pt.InstanceID, pt.DatabaseName, pt.TableNameField)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.PartitionOperationRecord, len(records))
	for i, rec := range records {
		result[i] = toBizPartitionOperationRecord(&rec)
	}
	return result, total, nil
}

// GetPartitionArchiveConfigs 获取分区归档配置
func (r *partitionRepo) GetPartitionArchiveConfigs(ctx context.Context) ([]*biz.PartitionArchiveConfig, error) {
	// 这里需要查询archive_configs表，并关联partition_table_info判断是否为分区表
	// 简化实现，返回空列表
	return []*biz.PartitionArchiveConfig{}, nil
}

// toBizPartitionTableInfo 转换为业务实体
func toBizPartitionTableInfo(pt *PartitionTableInfo) *biz.PartitionTableInfo {
	return &biz.PartitionTableInfo{
		ID:              pt.ID,
		InstanceID:      pt.InstanceID,
		InstanceName:    pt.InstanceName,
		ConnectionString: pt.ConnectionString,
		DatabaseName:    pt.DatabaseName,
		TableName:       pt.TableNameField,
		PartType:        pt.PartType,
		PartKey:         pt.PartKey,
		PartCycle:       pt.PartCycle,
		PartNum:         pt.PartNum,
		PartMaxValue:    pt.PartMaxValue,
		PartMaxValueDay: pt.PartMaxValueDay,
		HaveMaxValue:    pt.HaveMaxValue,
		TableSize:       pt.TableSize,
		TableRows:       pt.TableRows,
		MaxPartRows:     pt.MaxPartRows,
		IsCrontab:       pt.IsCrontab,
		CreatedAt:       pt.CreatedAt,
		UpdatedAt:       pt.UpdatedAt,
	}
}

// toBizPartitionOperationRecord 转换为业务实体
func toBizPartitionOperationRecord(rec *PartitionOperationRecord) *biz.PartitionOperationRecord {
	return &biz.PartitionOperationRecord{
		ID:              rec.ID,
		WorkID:          rec.WorkID,
		InstanceID:      rec.InstanceID,
		InstanceName:    rec.InstanceName,
		ConnectionString: rec.ConnectionString,
		TableSchema:     rec.TableSchema,
		TableName:       rec.TableNameField,
		OperateType:     int(rec.OperateType),
		ExeStatus:       int(rec.ExeStatus),
		CheckStatus:     int(rec.CheckStatus),
		ExePct:          rec.ExePct,
		ExeResult:       rec.ExeResult,
		ExeCmd:          rec.ExeCmd,
		WhereCondition:  rec.WhereCondition,
		AddUser:         rec.AddUser,
		UpdateUser:      rec.UpdateUser,
		ExeTime:         rec.ExeTime,
		CreatedAt:       rec.CreatedAt,
		UpdatedAt:       rec.UpdatedAt,
	}
}
