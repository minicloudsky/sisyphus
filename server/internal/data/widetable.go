package data

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"sisyphus/server/internal/biz"
)

// WidetableConfig GORM模型
type WidetableConfig struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Name             string    `gorm:"column:name;type:varchar(128);not null"`
	Description      string    `gorm:"column:description;type:varchar(512)"`
	SourceInstanceID uint64    `gorm:"column:source_instance_id;not null;index:idx_source_instance"`
	MainDatabase     string    `gorm:"column:main_database;type:varchar(64);not null"`
	MainTable        string    `gorm:"column:main_table;type:varchar(64);not null"`
	JoinConfig       string    `gorm:"column:join_config;type:text"`
	FieldConfig      string    `gorm:"column:field_config;type:text"`
	SyncCondition    string    `gorm:"column:sync_condition;type:varchar(512)"`
	TargetInstanceID uint64    `gorm:"column:target_instance_id"`
	TargetDatabase   string    `gorm:"column:target_database;type:varchar(64)"`
	TargetTable      string    `gorm:"column:target_table;type:varchar(64)"`
	Status           int       `gorm:"column:status;type:tinyint;default:0"`
	CreatedBy        uint64    `gorm:"column:created_by;not null"`
	CreatedAt        time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
}

func (WidetableConfig) TableName() string {
	return "sisyphus_widetable_configs"
}

// WidetableSyncRecord GORM模型
type WidetableSyncRecord struct {
	ID            uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	ConfigID      uint64    `gorm:"column:config_id;not null;index:idx_config"`
	SyncType      int       `gorm:"column:sync_type;type:tinyint;not null"`
	Status        int       `gorm:"column:status;type:tinyint;default:0"`
	ProcessedRows int64     `gorm:"column:processed_rows;default:0"`
	ErrorMessage  string    `gorm:"column:error_message;type:text"`
	CostSeconds   float64   `gorm:"column:cost_seconds;type:decimal(10,2)"`
	StartedAt     time.Time `gorm:"column:started_at;type:datetime"`
	FinishedAt    time.Time `gorm:"column:finished_at;type:datetime"`
	CreatedAt     time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
}

func (WidetableSyncRecord) TableName() string {
	return "sisyphus_widetable_sync_records"
}

// widetableRepo 宽表仓储实现
type widetableRepo struct {
	data *Data
}

// NewWidetableRepo 创建宽表仓储
func NewWidetableRepo(data *Data) biz.WidetableRepo {
	return &widetableRepo{data: data}
}

func (r *widetableRepo) CreateConfig(ctx context.Context, config *biz.WidetableConfig) (*biz.WidetableConfig, error) {
	m := toDataWidetableConfig(config)
	if err := r.data.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}
	config.ID = m.ID
	return config, nil
}

func (r *widetableRepo) UpdateConfig(ctx context.Context, config *biz.WidetableConfig) error {
	updates := map[string]interface{}{
		"name":               config.Name,
		"description":        config.Description,
		"source_instance_id": config.SourceInstanceID,
		"main_database":      config.MainDatabase,
		"main_table":         config.MainTable,
		"join_config":        config.JoinConfig,
		"field_config":       config.FieldConfig,
		"sync_condition":     config.SyncCondition,
		"target_instance_id": config.TargetInstanceID,
		"target_database":    config.TargetDatabase,
		"target_table":       config.TargetTable,
		"updated_at":         time.Now(),
	}
	return r.data.db.WithContext(ctx).Model(&WidetableConfig{}).Where("id = ?", config.ID).Updates(updates).Error
}

func (r *widetableRepo) DeleteConfig(ctx context.Context, id uint64) error {
	return r.data.db.WithContext(ctx).Delete(&WidetableConfig{}, id).Error
}

func (r *widetableRepo) GetConfig(ctx context.Context, id uint64) (*biz.WidetableConfig, error) {
	var m WidetableConfig
	if err := r.data.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("配置不存在")
		}
		return nil, err
	}
	return toBizWidetableConfig(&m), nil
}

func (r *widetableRepo) ListConfigs(ctx context.Context, page, pageSize int, keyword string) ([]*biz.WidetableConfig, int64, error) {
	db := r.data.db.WithContext(ctx).Model(&WidetableConfig{})
	if keyword != "" {
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []WidetableConfig
	offset := (page - 1) * pageSize
	if err := db.Order("id DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.WidetableConfig, len(records))
	for i, r := range records {
		result[i] = toBizWidetableConfig(&r)
	}
	return result, total, nil
}

func (r *widetableRepo) CreateSyncRecord(ctx context.Context, record *biz.WidetableSyncRecord) (*biz.WidetableSyncRecord, error) {
	m := toDataSyncRecord(record)
	if err := r.data.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}
	record.ID = m.ID
	return record, nil
}

func (r *widetableRepo) UpdateSyncRecord(ctx context.Context, record *biz.WidetableSyncRecord) error {
	updates := map[string]interface{}{
		"status":         record.Status,
		"processed_rows": record.ProcessedRows,
		"error_message":  record.ErrorMessage,
		"cost_seconds":   record.CostSeconds,
		"finished_at":    record.FinishedAt,
		"updated_at":     time.Now(),
	}
	return r.data.db.WithContext(ctx).Model(&WidetableSyncRecord{}).Where("id = ?", record.ID).Updates(updates).Error
}

func (r *widetableRepo) ListSyncRecords(ctx context.Context, configID uint64, page, pageSize int) ([]*biz.WidetableSyncRecord, int64, error) {
	db := r.data.db.WithContext(ctx).Model(&WidetableSyncRecord{}).Where("config_id = ?", configID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []WidetableSyncRecord
	offset := (page - 1) * pageSize
	if err := db.Order("id DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.WidetableSyncRecord, len(records))
	for i, rec := range records {
		result[i] = toBizSyncRecord(&rec)
	}
	return result, total, nil
}

// ---- Converters ----

func toDataWidetableConfig(c *biz.WidetableConfig) *WidetableConfig {
	return &WidetableConfig{
		ID:               c.ID,
		Name:             c.Name,
		Description:      c.Description,
		SourceInstanceID: c.SourceInstanceID,
		MainDatabase:     c.MainDatabase,
		MainTable:        c.MainTable,
		JoinConfig:       c.JoinConfig,
		FieldConfig:      c.FieldConfig,
		SyncCondition:    c.SyncCondition,
		TargetInstanceID: c.TargetInstanceID,
		TargetDatabase:   c.TargetDatabase,
		TargetTable:      c.TargetTable,
		Status:           c.Status,
		CreatedBy:        c.CreatedBy,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func toBizWidetableConfig(m *WidetableConfig) *biz.WidetableConfig {
	return &biz.WidetableConfig{
		ID:               m.ID,
		Name:             m.Name,
		Description:      m.Description,
		SourceInstanceID: m.SourceInstanceID,
		MainDatabase:     m.MainDatabase,
		MainTable:        m.MainTable,
		JoinConfig:       m.JoinConfig,
		FieldConfig:      m.FieldConfig,
		SyncCondition:    m.SyncCondition,
		TargetInstanceID: m.TargetInstanceID,
		TargetDatabase:   m.TargetDatabase,
		TargetTable:      m.TargetTable,
		Status:           m.Status,
		CreatedBy:        m.CreatedBy,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func toDataSyncRecord(r *biz.WidetableSyncRecord) *WidetableSyncRecord {
	return &WidetableSyncRecord{
		ID:            r.ID,
		ConfigID:      r.ConfigID,
		SyncType:      r.SyncType,
		Status:        r.Status,
		ProcessedRows: r.ProcessedRows,
		ErrorMessage:  r.ErrorMessage,
		CostSeconds:   r.CostSeconds,
		StartedAt:     r.StartedAt,
		FinishedAt:    r.FinishedAt,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func toBizSyncRecord(m *WidetableSyncRecord) *biz.WidetableSyncRecord {
	return &biz.WidetableSyncRecord{
		ID:            m.ID,
		ConfigID:      m.ConfigID,
		SyncType:      m.SyncType,
		Status:        m.Status,
		ProcessedRows: m.ProcessedRows,
		ErrorMessage:  m.ErrorMessage,
		CostSeconds:   m.CostSeconds,
		StartedAt:     m.StartedAt,
		FinishedAt:    m.FinishedAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
