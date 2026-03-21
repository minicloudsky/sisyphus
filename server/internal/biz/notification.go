package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// 通知渠道常量
const (
	NotificationChannelWeChatWork = "wechatwork"
	NotificationChannelFeishu     = "feishu"
	NotificationChannelDingTalk   = "dingtalk"
)

// NotificationConfig 通知渠道配置
type NotificationConfig struct {
	ID          uint64
	Channel     string
	Enabled     bool
	WebhookURL  string
	ExtraConfig string // JSON 字符串，存储渠道特定配置（如钉钉@手机号列表）
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NotificationConfigRepo 通知配置仓储接口
type NotificationConfigRepo interface {
	List(ctx context.Context) ([]*NotificationConfig, error)
	GetByChannel(ctx context.Context, channel string) (*NotificationConfig, error)
	Upsert(ctx context.Context, config *NotificationConfig) error
}

// NotificationService 通知服务
type NotificationService struct {
	repo NotificationConfigRepo
	log  *log.Helper
}

// NewNotificationService 创建通知服务
func NewNotificationService(repo NotificationConfigRepo, logger log.Logger) *NotificationService {
	return &NotificationService{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// GetConfigs 获取所有通知渠道配置
func (s *NotificationService) GetConfigs(ctx context.Context) ([]*NotificationConfig, error) {
	return s.repo.List(ctx)
}

// UpsertConfig 更新或新建通知渠道配置
func (s *NotificationService) UpsertConfig(ctx context.Context, config *NotificationConfig) error {
	return s.repo.Upsert(ctx, config)
}

// TestChannel 向指定渠道发送测试消息
func (s *NotificationService) TestChannel(ctx context.Context, channel string) error {
	testContent := "这是来自 Sisyphus DBA 平台的测试通知 ✅"
	switch channel {
	case NotificationChannelWeChatWork:
		return s.sendWeChatWorkMessage(ctx, testContent)
	case NotificationChannelFeishu:
		return s.sendFeishuMessage(ctx, testContent)
	case NotificationChannelDingTalk:
		return s.sendDingTalkMessage(ctx, testContent)
	default:
		return fmt.Errorf("不支持的通知渠道: %s", channel)
	}
}

// sendWeChatWorkMessage 发送企业微信机器人消息（Markdown 格式）
func (s *NotificationService) sendWeChatWorkMessage(ctx context.Context, content string) error {
	cfg, err := s.repo.GetByChannel(ctx, NotificationChannelWeChatWork)
	if err != nil || cfg == nil || !cfg.Enabled || cfg.WebhookURL == "" {
		if err != nil {
			s.log.Warnf("获取企业微信配置失败: %v", err)
		}
		return nil
	}
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}
	if err := s.postWebhook(ctx, cfg.WebhookURL, payload); err != nil {
		return fmt.Errorf("发送企业微信消息失败: %w", err)
	}
	s.log.Infof("企业微信消息发送成功")
	return nil
}

// sendFeishuMessage 发送飞书机器人消息
func (s *NotificationService) sendFeishuMessage(ctx context.Context, content string) error {
	cfg, err := s.repo.GetByChannel(ctx, NotificationChannelFeishu)
	if err != nil || cfg == nil || !cfg.Enabled || cfg.WebhookURL == "" {
		if err != nil {
			s.log.Warnf("获取飞书配置失败: %v", err)
		}
		return nil
	}
	// 飞书自定义机器人 webhook 消息格式
	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}
	if err := s.postWebhook(ctx, cfg.WebhookURL, payload); err != nil {
		return fmt.Errorf("发送飞书消息失败: %w", err)
	}
	s.log.Infof("飞书消息发送成功")
	return nil
}

// sendDingTalkMessage 发送钉钉机器人消息
func (s *NotificationService) sendDingTalkMessage(ctx context.Context, content string) error {
	cfg, err := s.repo.GetByChannel(ctx, NotificationChannelDingTalk)
	if err != nil || cfg == nil || !cfg.Enabled || cfg.WebhookURL == "" {
		if err != nil {
			s.log.Warnf("获取钉钉配置失败: %v", err)
		}
		return nil
	}

	// 从 extra_config 读取 @手机号列表
	var extra struct {
		Phones []string `json:"phones"`
	}
	if cfg.ExtraConfig != "" {
		_ = json.Unmarshal([]byte(cfg.ExtraConfig), &extra)
	}

	msg := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}
	if len(extra.Phones) > 0 {
		msg["at"] = map[string]interface{}{
			"atMobiles": extra.Phones,
			"isAtAll":   false,
		}
	}

	if err := s.postWebhook(ctx, cfg.WebhookURL, msg); err != nil {
		return fmt.Errorf("发送钉钉消息失败: %w", err)
	}
	s.log.Infof("钉钉消息发送成功")
	return nil
}

// SendSQLOrderNotification 发送 SQL 工单状态变更通知到所有已启用渠道
func (s *NotificationService) SendSQLOrderNotification(ctx context.Context, order *SQLOrder, status string, actionUser string) error {
	content := fmt.Sprintf(
		"**Sisyphus数据库变更%s通知**\n\n工单编号: %s\n数据库: %s\n状态: %s\n操作人: %s\n备注: %s",
		status, order.WorkID, order.DatabaseName, status, actionUser, order.Remark,
	)

	if err := s.sendWeChatWorkMessage(ctx, content); err != nil {
		s.log.Warnf("发送企业微信通知失败: %v", err)
	}
	if err := s.sendFeishuMessage(ctx, content); err != nil {
		s.log.Warnf("发送飞书通知失败: %v", err)
	}
	if err := s.sendDingTalkMessage(ctx, content); err != nil {
		s.log.Warnf("发送钉钉通知失败: %v", err)
	}
	return nil
}

// postWebhook 向指定 webhook URL 发送 JSON POST 请求
func (s *NotificationService) postWebhook(ctx context.Context, webhookURL string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook响应异常: status=%d", resp.StatusCode)
	}
	return nil
}
