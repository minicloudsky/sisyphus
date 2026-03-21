package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"

	"sisyphus/server/internal/biz"
	asynqPkg "sisyphus/server/internal/pkg/asynq"
)

// ArchiveTaskScheduler 归档任务调度器（用于定时任务）
type ArchiveTaskScheduler struct {
	archiveUsecase *biz.ArchiveUsecase
	asynqClient    *asynqPkg.Client
	log            *log.Helper
}

// NewArchiveTaskScheduler 创建归档任务调度器
func NewArchiveTaskScheduler(archiveUsecase *biz.ArchiveUsecase, asynqClient *asynqPkg.Client, logger log.Logger) *ArchiveTaskScheduler {
	return &ArchiveTaskScheduler{
		archiveUsecase: archiveUsecase,
		asynqClient:    asynqClient,
		log:            log.NewHelper(logger),
	}
}

// ScheduleArchiveTask 调度归档任务（供cron调用）
func (s *ArchiveTaskScheduler) ScheduleArchiveTask(ctx context.Context, configID uint64) {
	config, err := s.archiveUsecase.GetConfigByID(ctx, configID)
	if err != nil {
		s.log.Errorf("获取归档配置失败: config_id=%d, error=%v", configID, err)
		return
	}

	if !config.IsEnabled {
		return
	}

	// 生成workID
	workID := fmt.Sprintf("archive_%d_%d", configID, time.Now().Unix())

	// 构建归档命令
	command, err := s.archiveUsecase.BuildArchiveCommand(ctx, config, workID)
	if err != nil {
		s.log.Errorf("构建归档命令失败: config_id=%d, error=%v", configID, err)
		return
	}

	// 创建归档记录
	record := &biz.ArchiveRecord{
		ConfigID: configID,
		WorkID:   workID,
		Status:   0, // 待执行
	}
	record, err = s.archiveUsecase.CreateRecord(ctx, record)
	if err != nil {
		s.log.Errorf("创建归档记录失败: error=%v", err)
		return
	}

	// 使用Asynq入队任务
	payload := biz.ArchiveExecutePayload{
		ConfigID: configID,
		WorkID:   workID,
		Command:  command,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.log.Errorf("序列化任务负载失败: error=%v", err)
		return
	}

	opts := []asynq.Option{
		asynq.Queue("default"),
		asynq.TaskID(fmt.Sprintf("archive:%d:%s", configID, workID)),
	}

	// 如果配置了执行时间，设置延迟执行
	if config.CronExpression != "" {
		// 解析cron表达式，计算下次执行时间
		// 这里简化处理，立即执行
	}

	taskInfo, err := s.asynqClient.Enqueue(asynqPkg.TaskTypeArchiveExecute, payloadBytes, opts...)
	if err != nil {
		s.log.Errorf("入队归档任务失败: config_id=%d, error=%v", configID, err)
		// 更新记录状态为失败
		record.Status = 3
		record.ErrorMessage = fmt.Sprintf("入队任务失败: %v", err)
		s.archiveUsecase.UpdateRecord(ctx, record)
		return
	}

	s.log.Infof("归档任务已入队: config_id=%d, work_id=%s, task_id=%s", configID, workID, taskInfo.ID)
}
