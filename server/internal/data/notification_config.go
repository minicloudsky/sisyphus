package data

import (
	"context"
	"time"

	"sisyphus/server/internal/biz"
)

// NotificationConfig GORM 模型
type NotificationConfig struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	Channel     string    `gorm:"size:50;uniqueIndex;not null"`
	Enabled     bool      `gorm:"not null;default:false"`
	WebhookURL  string    `gorm:"column:webhook_url;size:500;not null;default:''"`
	ExtraConfig string    `gorm:"column:extra_config;type:json"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"not null;autoUpdateTime"`
}

func (NotificationConfig) TableName() string {
	return "sisyphus_notification_configs"
}

type notificationConfigRepo struct {
	data *Data
}

// NewNotificationConfigRepo 创建通知配置仓储
func NewNotificationConfigRepo(data *Data) biz.NotificationConfigRepo {
	return &notificationConfigRepo{data: data}
}

func (r *notificationConfigRepo) List(ctx context.Context) ([]*biz.NotificationConfig, error) {
	var models []NotificationConfig
	if err := r.data.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*biz.NotificationConfig, 0, len(models))
	for i := range models {
		result = append(result, toNotificationConfigBiz(&models[i]))
	}
	return result, nil
}

func (r *notificationConfigRepo) GetByChannel(ctx context.Context, channel string) (*biz.NotificationConfig, error) {
	var model NotificationConfig
	if err := r.data.db.WithContext(ctx).Where("channel = ?", channel).First(&model).Error; err != nil {
		return nil, err
	}
	return toNotificationConfigBiz(&model), nil
}

func (r *notificationConfigRepo) Upsert(ctx context.Context, config *biz.NotificationConfig) error {
	updates := map[string]interface{}{
		"enabled":      config.Enabled,
		"webhook_url":  config.WebhookURL,
		"extra_config": config.ExtraConfig,
		"updated_at":   time.Now(),
	}
	result := r.data.db.WithContext(ctx).
		Model(&NotificationConfig{}).
		Where("channel = ?", config.Channel).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 不存在，创建
		model := NotificationConfig{
			Channel:     config.Channel,
			Enabled:     config.Enabled,
			WebhookURL:  config.WebhookURL,
			ExtraConfig: config.ExtraConfig,
		}
		return r.data.db.WithContext(ctx).Create(&model).Error
	}
	return nil
}

func toNotificationConfigBiz(m *NotificationConfig) *biz.NotificationConfig {
	return &biz.NotificationConfig{
		ID:          m.ID,
		Channel:     m.Channel,
		Enabled:     m.Enabled,
		WebhookURL:  m.WebhookURL,
		ExtraConfig: m.ExtraConfig,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
