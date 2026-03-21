package data

import (
	"context"
	"encoding/json"
	"time"

	"sisyphus/server/internal/biz"
)

type redisAnalysisRepo struct {
	data *Data
}

// NewRedisAnalysisRepo 创建Redis分析仓储
func NewRedisAnalysisRepo(data *Data) biz.RedisAnalysisRepo {
	return &redisAnalysisRepo{data: data}
}

func (r *redisAnalysisRepo) CreateTask(ctx context.Context, task *biz.RedisHotKeyTask) (*biz.RedisHotKeyTask, error) {
	resultJSON, _ := json.Marshal(task.Result)
	t := &RedisHotKeyTask{
		InstanceID:  task.InstanceID,
		TaskType:    task.TaskType,
		Status:      int8(task.Status),
		Result:       string(resultJSON),
		SubmitterID: task.SubmitterID,
		StartedAt:   task.StartedAt,
		FinishedAt:  task.FinishedAt,
	}
	if err := r.data.db.WithContext(ctx).Create(t).Error; err != nil {
		return nil, err
	}
	return toBizRedisHotKeyTask(t), nil
}

func (r *redisAnalysisRepo) UpdateTask(ctx context.Context, task *biz.RedisHotKeyTask) error {
	updates := map[string]interface{}{
		"status":     task.Status,
		"updated_at": time.Now(),
	}
	if task.Result != nil {
		resultJSON, _ := json.Marshal(task.Result)
		updates["result"] = string(resultJSON)
	}
	if task.StartedAt != nil {
		updates["started_at"] = *task.StartedAt
	}
	if task.FinishedAt != nil {
		updates["finished_at"] = *task.FinishedAt
	}
	return r.data.db.WithContext(ctx).Model(&RedisHotKeyTask{}).Where("id = ?", task.ID).Updates(updates).Error
}

func (r *redisAnalysisRepo) GetTaskByID(ctx context.Context, id uint64) (*biz.RedisHotKeyTask, error) {
	var task RedisHotKeyTask
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return toBizRedisHotKeyTask(&task), nil
}

func (r *redisAnalysisRepo) ListTasks(ctx context.Context, instanceID uint64, page, pageSize int) ([]*biz.RedisHotKeyTask, int64, error) {
	var tasks []RedisHotKeyTask
	var total int64

	query := r.data.db.WithContext(ctx).Model(&RedisHotKeyTask{}).Where("instance_id = ?", instanceID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.RedisHotKeyTask, len(tasks))
	for i, t := range tasks {
		result[i] = toBizRedisHotKeyTask(&t)
	}
	return result, total, nil
}

// RedisHotKeyTask 数据库模型
type RedisHotKeyTask struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	InstanceID  uint64     `gorm:"column:instance_id;not null;index"`
	TaskType    string     `gorm:"column:task_type;type:varchar(50);not null"`
	Status      int8       `gorm:"column:status;type:tinyint;default:0;index"`
	Result      string     `gorm:"column:result;type:json"`
	SubmitterID uint64     `gorm:"column:submitter_id;not null"`
	StartedAt   *time.Time `gorm:"column:started_at;default:null"`
	FinishedAt  *time.Time `gorm:"column:finished_at;default:null"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (RedisHotKeyTask) TableName() string {
	return "sisyphus_redis_hotkey_tasks"
}

func toBizRedisHotKeyTask(t *RedisHotKeyTask) *biz.RedisHotKeyTask {
	var result map[string]interface{}
	json.Unmarshal([]byte(t.Result), &result)
	
	return &biz.RedisHotKeyTask{
		ID:          t.ID,
		InstanceID:  t.InstanceID,
		TaskType:    t.TaskType,
		Status:      int(t.Status),
		Result:      result,
		SubmitterID: t.SubmitterID,
		StartedAt:   t.StartedAt,
		FinishedAt:  t.FinishedAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
