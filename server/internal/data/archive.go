package data

import (
	"context"
	"time"

	"sisyphus/server/internal/biz"
)

type archiveRepo struct {
	data *Data
}

// NewArchiveRepo 创建归档仓储
func NewArchiveRepo(data *Data) biz.ArchiveRepo {
	return &archiveRepo{data: data}
}

func (r *archiveRepo) CreateConfig(ctx context.Context, config *biz.ArchiveConfig) (*biz.ArchiveConfig, error) {
	c := &ArchiveConfig{
		InstanceID:      config.InstanceID,
		SourceDatabase:  config.SourceDatabase,
		SourceTable:     config.SourceTable,
		DestInstanceID:  config.DestInstanceID,
		DestDatabase:    config.DestDatabase,
		DestTable:       config.DestTable,
		ArchiveField:    config.ArchiveField,
		RetentionDays:   config.RetentionDays,
		ArchiveCondition: config.ArchiveCondition,
		IsEnabled:       config.IsEnabled,
		CronExpression:  config.CronExpression,
		Owner:           config.Owner,
	}
	if err := r.data.db.WithContext(ctx).Create(c).Error; err != nil {
		return nil, err
	}
	return toBizArchiveConfig(c), nil
}

func (r *archiveRepo) UpdateConfig(ctx context.Context, config *biz.ArchiveConfig) error {
	updates := map[string]interface{}{
		"source_database":   config.SourceDatabase,
		"source_table":      config.SourceTable,
		"dest_instance_id":  config.DestInstanceID,
		"dest_database":     config.DestDatabase,
		"dest_table":        config.DestTable,
		"archive_field":     config.ArchiveField,
		"retention_days":    config.RetentionDays,
		"archive_condition": config.ArchiveCondition,
		"is_enabled":        config.IsEnabled,
		"cron_expression":   config.CronExpression,
		"owner":             config.Owner,
		"updated_at":        time.Now(),
	}
	return r.data.db.WithContext(ctx).Model(&ArchiveConfig{}).Where("id = ?", config.ID).Updates(updates).Error
}

func (r *archiveRepo) DeleteConfig(ctx context.Context, id uint64) error {
	return r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&ArchiveConfig{}).Error
}

func (r *archiveRepo) GetConfigByID(ctx context.Context, id uint64) (*biz.ArchiveConfig, error) {
	var config ArchiveConfig
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&config).Error; err != nil {
		return nil, err
	}
	return toBizArchiveConfig(&config), nil
}

func (r *archiveRepo) ListConfigs(ctx context.Context, page, pageSize int) ([]*biz.ArchiveConfig, int64, error) {
	var configs []ArchiveConfig
	var total int64

	if err := r.data.db.WithContext(ctx).Model(&ArchiveConfig{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.data.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.ArchiveConfig, len(configs))
	for i, c := range configs {
		result[i] = toBizArchiveConfig(&c)
	}
	return result, total, nil
}

func (r *archiveRepo) GetEnabledConfigs(ctx context.Context) ([]*biz.ArchiveConfig, error) {
	var configs []ArchiveConfig
	if err := r.data.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&configs).Error; err != nil {
		return nil, err
	}

	result := make([]*biz.ArchiveConfig, len(configs))
	for i, c := range configs {
		result[i] = toBizArchiveConfig(&c)
	}
	return result, nil
}

func (r *archiveRepo) CreateRecord(ctx context.Context, record *biz.ArchiveRecord) (*biz.ArchiveRecord, error) {
	rec := &ArchiveRecord{
		ConfigID:     record.ConfigID,
		WorkID:       record.WorkID,
		Status:       int8(record.Status),
		ArchiveCount: record.ArchiveCount,
		ExecuteTime:  record.ExecuteTime,
		FinishedTime: record.FinishedTime,
		ErrorMessage: record.ErrorMessage,
	}
	if err := r.data.db.WithContext(ctx).Create(rec).Error; err != nil {
		return nil, err
	}
	return toBizArchiveRecord(rec), nil
}

func (r *archiveRepo) UpdateRecord(ctx context.Context, record *biz.ArchiveRecord) error {
	updates := map[string]interface{}{
		"status":        record.Status,
		"archive_count": record.ArchiveCount,
		"error_message": record.ErrorMessage,
		"updated_at":    time.Now(),
	}
	if record.ExecuteTime != nil {
		updates["execute_time"] = *record.ExecuteTime
	}
	if record.FinishedTime != nil {
		updates["finished_time"] = *record.FinishedTime
	}
	return r.data.db.WithContext(ctx).Model(&ArchiveRecord{}).Where("id = ?", record.ID).Updates(updates).Error
}

func (r *archiveRepo) GetRecordByID(ctx context.Context, id uint64) (*biz.ArchiveRecord, error) {
	var record ArchiveRecord
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return nil, err
	}
	return toBizArchiveRecord(&record), nil
}

func (r *archiveRepo) ListRecords(ctx context.Context, configID uint64, page, pageSize int) ([]*biz.ArchiveRecord, int64, error) {
	var records []ArchiveRecord
	var total int64

	query := r.data.db.WithContext(ctx).Model(&ArchiveRecord{}).Where("config_id = ?", configID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.ArchiveRecord, len(records))
	for i, rec := range records {
		result[i] = toBizArchiveRecord(&rec)
	}
	return result, total, nil
}

// ArchiveConfig 数据库模型
type ArchiveConfig struct {
	ID               uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	InstanceID       uint64     `gorm:"column:instance_id;not null;index"`
	SourceDatabase   string     `gorm:"column:source_database;type:varchar(100);not null"`
	SourceTable      string     `gorm:"column:source_table;type:varchar(100);not null"`
	DestInstanceID   *uint64    `gorm:"column:dest_instance_id;default:null;index"`
	DestDatabase     string     `gorm:"column:dest_database;type:varchar(100)"`
	DestTable        string     `gorm:"column:dest_table;type:varchar(100)"`
	ArchiveField     string     `gorm:"column:archive_field;type:varchar(100);not null"`
	RetentionDays    int        `gorm:"column:retention_days;not null"`
	ArchiveCondition string     `gorm:"column:archive_condition;type:varchar(500);default:''"`
	IsEnabled        bool       `gorm:"column:is_enabled;default:true;index"`
	CronExpression   string     `gorm:"column:cron_expression;type:varchar(100);not null"`
	Owner            string     `gorm:"column:owner;type:varchar(100);default:''"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (ArchiveConfig) TableName() string {
	return "sisyphus_archive_configs"
}

type ArchiveRecord struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ConfigID     uint64     `gorm:"column:config_id;not null;index"`
	WorkID       string     `gorm:"column:work_id;type:varchar(50);not null;index"`
	Status       int8       `gorm:"column:status;type:tinyint;default:0;index"`
	ArchiveCount int        `gorm:"column:archive_count;default:0"`
	ExecuteTime  *time.Time `gorm:"column:execute_time;default:null"`
	FinishedTime *time.Time `gorm:"column:finished_time;default:null"`
	ErrorMessage string     `gorm:"column:error_message;type:text"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (ArchiveRecord) TableName() string {
	return "sisyphus_archive_records"
}

func toBizArchiveConfig(c *ArchiveConfig) *biz.ArchiveConfig {
	return &biz.ArchiveConfig{
		ID:              c.ID,
		InstanceID:      c.InstanceID,
		SourceDatabase:  c.SourceDatabase,
		SourceTable:     c.SourceTable,
		DestInstanceID:  c.DestInstanceID,
		DestDatabase:    c.DestDatabase,
		DestTable:       c.DestTable,
		ArchiveField:    c.ArchiveField,
		RetentionDays:   c.RetentionDays,
		ArchiveCondition: c.ArchiveCondition,
		IsEnabled:       c.IsEnabled,
		CronExpression:  c.CronExpression,
		Owner:           c.Owner,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

func toBizArchiveRecord(r *ArchiveRecord) *biz.ArchiveRecord {
	return &biz.ArchiveRecord{
		ID:           r.ID,
		ConfigID:     r.ConfigID,
		WorkID:       r.WorkID,
		Status:       int(r.Status),
		ArchiveCount: r.ArchiveCount,
		ExecuteTime:  r.ExecuteTime,
		FinishedTime: r.FinishedTime,
		ErrorMessage: r.ErrorMessage,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}
