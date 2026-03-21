package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	asynqPkg "sisyphus/server/internal/pkg/asynq"
	"sisyphus/server/internal/middleware"
)

// RedisService Redis分析服务
type RedisService struct {
	redisUsecase    *biz.RedisAnalysisUsecase
	instanceUsecase *biz.DatabaseInstanceUsecase
	asynqClient     *asynqPkg.Client
	log             *log.Helper
}

// NewRedisService 创建Redis分析服务
func NewRedisService(
	redisUsecase *biz.RedisAnalysisUsecase,
	instanceUsecase *biz.DatabaseInstanceUsecase,
	asynqClient *asynqPkg.Client,
	logger log.Logger,
) *RedisService {
	return &RedisService{
		redisUsecase:    redisUsecase,
		instanceUsecase: instanceUsecase,
		asynqClient:     asynqClient,
		log:             log.NewHelper(logger),
	}
}

// AnalyzeHotKeys 分析热key
func (s *RedisService) AnalyzeHotKeys(ctx context.Context, req *v1.AnalyzeHotKeysRequest) (*v1.AnalyzeHotKeysResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.AnalyzeHotKeysResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	// 获取实例信息
	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.AnalyzeHotKeysResponse{
			Code:    404,
			Message: "实例不存在",
		}, nil
	}
	_ = instance // 暂时保留，后续可能需要

	// 创建任务记录
	task := &biz.RedisHotKeyTask{
		InstanceID:  req.InstanceId,
		TaskType:    "hotkey",
		Status:      0, // 待执行
		SubmitterID: claims.UserID,
	}
	task, err = s.redisUsecase.CreateTask(ctx, task)
	if err != nil {
		return &v1.AnalyzeHotKeysResponse{
			Code:    500,
			Message: fmt.Sprintf("创建任务失败: %v", err),
		}, nil
	}

	// 使用Asynq入队任务
	payload := biz.RedisRDBParsePayload{
		TaskID:     task.ID,
		InstanceID: req.InstanceId,
		CreatedBy:  claims.UserID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return &v1.AnalyzeHotKeysResponse{
			Code:    500,
			Message: fmt.Sprintf("序列化任务负载失败: %v", err),
		}, nil
	}

	opts := []asynq.Option{
		asynq.Queue("default"),
		asynq.TaskID(fmt.Sprintf("redis_rdb:%d", task.ID)),
	}

	taskInfo, err := s.asynqClient.Enqueue(asynqPkg.TaskTypeRedisRDBParse, payloadBytes, opts...)
	if err != nil {
		return &v1.AnalyzeHotKeysResponse{
			Code:    500,
			Message: fmt.Sprintf("入队任务失败: %v", err),
		}, nil
	}

	return &v1.AnalyzeHotKeysResponse{
		Code:    200,
		Message: "分析任务已发起",
		Data: &v1.AnalyzeHotKeysData{
			TaskId:  task.ID,
			WorkId:  fmt.Sprintf("hotkey_%d_%d", req.InstanceId, time.Now().Unix()),
			Message: fmt.Sprintf("热key分析任务已发起，任务ID: %s", taskInfo.ID),
		},
	}, nil
}

// AnalyzeBigKeys 分析大key
func (s *RedisService) AnalyzeBigKeys(ctx context.Context, req *v1.AnalyzeBigKeysRequest) (*v1.AnalyzeBigKeysResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.AnalyzeBigKeysResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	// 获取实例信息（验证实例存在）
	_, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.AnalyzeBigKeysResponse{
			Code:    404,
			Message: "实例不存在",
		}, nil
	}

	// 创建任务记录
	task := &biz.RedisHotKeyTask{
		InstanceID:  req.InstanceId,
		TaskType:    "bigkey",
		Status:      0, // 待执行
		SubmitterID: claims.UserID,
	}
	task, err = s.redisUsecase.CreateTask(ctx, task)
	if err != nil {
		return &v1.AnalyzeBigKeysResponse{
			Code:    500,
			Message: fmt.Sprintf("创建任务失败: %v", err),
		}, nil
	}

	// 使用Asynq入队任务
	payload := biz.RedisRDBParsePayload{
		TaskID:     task.ID,
		InstanceID: req.InstanceId,
		CreatedBy:  claims.UserID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return &v1.AnalyzeBigKeysResponse{
			Code:    500,
			Message: fmt.Sprintf("序列化任务负载失败: %v", err),
		}, nil
	}

	opts := []asynq.Option{
		asynq.Queue("default"),
		asynq.TaskID(fmt.Sprintf("redis_rdb:%d", task.ID)),
	}

	taskInfo, err := s.asynqClient.Enqueue(asynqPkg.TaskTypeRedisRDBParse, payloadBytes, opts...)
	if err != nil {
		return &v1.AnalyzeBigKeysResponse{
			Code:    500,
			Message: fmt.Sprintf("入队任务失败: %v", err),
		}, nil
	}

	return &v1.AnalyzeBigKeysResponse{
		Code:    200,
		Message: "分析任务已发起",
		Data: &v1.AnalyzeBigKeysData{
			TaskId:  task.ID,
			WorkId:  fmt.Sprintf("bigkey_%d_%d", req.InstanceId, time.Now().Unix()),
			Message: fmt.Sprintf("大key分析任务已发起，任务ID: %s", taskInfo.ID),
		},
	}, nil
}

// ListAnalysisTasks 获取分析任务列表
func (s *RedisService) ListAnalysisTasks(ctx context.Context, req *v1.ListAnalysisTasksRequest) (*v1.ListAnalysisTasksResponse, error) {
	page := 1
	pageSize := 10
	if req.PageRequest != nil {
		page = int(req.PageRequest.Page)
		pageSize = int(req.PageRequest.PageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var instanceID uint64
	if req.InstanceId != nil {
		instanceID = *req.InstanceId
	}

	tasks, total, err := s.redisUsecase.ListTasks(ctx, instanceID, page, pageSize)
	if err != nil {
		return &v1.ListAnalysisTasksResponse{
			Code:    500,
			Message: fmt.Sprintf("查询失败: %v", err),
		}, nil
	}

	var protoTasks []*v1.RedisAnalysisTask
	for _, task := range tasks {
		protoTasks = append(protoTasks, toProtoRedisAnalysisTask(task))
	}

	return &v1.ListAnalysisTasksResponse{
		Code:    200,
		Message: "查询成功",
		List:    protoTasks,
		PageResponse: &v1.PageResponse{
			Total:    int64(total),
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
	}, nil
}

// GetAnalysisTask 获取分析任务详情
func (s *RedisService) GetAnalysisTask(ctx context.Context, req *v1.GetAnalysisTaskRequest) (*v1.GetAnalysisTaskResponse, error) {
	task, err := s.redisUsecase.GetTaskByID(ctx, req.Id)
	if err != nil {
		return &v1.GetAnalysisTaskResponse{
			Code:    404,
			Message: "任务不存在",
		}, nil
	}

	return &v1.GetAnalysisTaskResponse{
		Code:    200,
		Message: "查询成功",
		Data:    toProtoRedisAnalysisTask(task),
	}, nil
}
