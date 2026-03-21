package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	"sisyphus/server/internal/middleware"
)

// GhOstService gh-ost管理服务
type GhOstService struct {
	v1.UnimplementedGhOstServiceServer
	ghostManager *biz.GhOstManager
	log          *log.Helper
}

// NewGhOstService 创建gh-ost管理服务
func NewGhOstService(ghostManager *biz.GhOstManager, logger log.Logger) *GhOstService {
	return &GhOstService{
		ghostManager: ghostManager,
		log:          log.NewHelper(logger),
	}
}

// GetProgress 获取gh-ost执行进度
func (s *GhOstService) GetProgress(ctx context.Context, req *v1.GetProgressRequest) (*v1.GetProgressResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetProgressResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if req.WorkId == "" {
		return &v1.GetProgressResponse{
			Code:    400,
			Message: "work_id参数必填",
		}, nil
	}

	progress, err := s.ghostManager.GetProgress(ctx, req.WorkId)
	if err != nil {
		return &v1.GetProgressResponse{
			Code:    500,
			Message: fmt.Sprintf("获取进度失败: %v", err),
		}, nil
	}

	// 从日志文件名提取表名
	tableName := ""
	if progress.LogName != "" {
		fileName := filepath.Base(progress.LogName)
		if idx := strings.LastIndex(fileName, "_"); idx > 0 {
			tableName = fileName[:idx]
		}
	}

	// 解析预计剩余时间（从 Consuming 字符串中提取）
	estimatedSeconds := 0.0
	// TODO: 解析 Consuming 字符串为秒数

	protoProgress := toProtoGhOstProgress(progress, req.WorkId, tableName)
	protoProgress.TotalRows = 0  // 需要从日志中解析
	protoProgress.CopiedRows = 0 // 需要从日志中解析
	protoProgress.EstimatedSecondsRemaining = estimatedSeconds

	return &v1.GetProgressResponse{
		Code:    200,
		Message: "查询成功",
		Data:    protoProgress,
	}, nil
}

// Throttle 暂停gh-ost执行
func (s *GhOstService) Throttle(ctx context.Context, req *v1.ThrottleRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if req.TableName == "" || req.WorkId == "" {
		return &v1.CommonResponse{
			Code:    400,
			Message: "table_name和work_id参数必填",
		}, nil
	}

	err := s.ghostManager.Throttle(ctx, req.TableName, req.WorkId)
	if err != nil {
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("暂停失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "DDL执行任务暂停成功!",
	}, nil
}

// NoThrottle 恢复gh-ost执行
func (s *GhOstService) NoThrottle(ctx context.Context, req *v1.NoThrottleRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if req.TableName == "" || req.WorkId == "" {
		return &v1.CommonResponse{
			Code:    400,
			Message: "table_name和work_id参数必填",
		}, nil
	}

	err := s.ghostManager.NoThrottle(ctx, req.TableName, req.WorkId)
	if err != nil {
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("恢复失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "DDL执行任务已恢复执行!",
	}, nil
}

// GetParams 获取gh-ost执行参数
func (s *GhOstService) GetParams(ctx context.Context, req *v1.GetParamsRequest) (*v1.GetParamsResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetParamsResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if req.WorkId == "" {
		return &v1.GetParamsResponse{
			Code:    400,
			Message: "work_id参数必填",
		}, nil
	}

	// 从 work_id 查找日志文件
	progress, err := s.ghostManager.GetProgress(ctx, req.WorkId)
	if err != nil {
		return &v1.GetParamsResponse{
			Code:    500,
			Message: fmt.Sprintf("获取进度失败，无法找到日志文件: %v", err),
		}, nil
	}

	if progress.LogName == "" {
		return &v1.GetParamsResponse{
			Code:    404,
			Message: "未找到日志文件",
		}, nil
	}

	params, err := s.ghostManager.GetParams(ctx, progress.LogName)
	if err != nil {
		return &v1.GetParamsResponse{
			Code:    500,
			Message: fmt.Sprintf("获取参数失败: %v", err),
		}, nil
	}

	return &v1.GetParamsResponse{
		Code:    200,
		Message: "查询成功",
		Data:    toProtoGhOstParams(params, req.WorkId),
	}, nil
}

// ModifyParams 修改gh-ost执行参数
func (s *GhOstService) ModifyParams(ctx context.Context, req *v1.ModifyParamsRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if req.WorkId == "" {
		return &v1.CommonResponse{
			Code:    400,
			Message: "work_id参数必填",
		}, nil
	}

	// 从 work_id 获取表名和日志文件
	progress, err := s.ghostManager.GetProgress(ctx, req.WorkId)
	if err != nil {
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("获取进度失败: %v", err),
		}, nil
	}

	// 从日志文件名提取表名
	tableName := ""
	if progress.LogName != "" {
		fileName := filepath.Base(progress.LogName)
		if idx := strings.LastIndex(fileName, "_"); idx > 0 {
			tableName = fileName[:idx]
		}
	}

	if tableName == "" {
		return &v1.CommonResponse{
			Code:    400,
			Message: "无法从日志文件提取表名",
		}, nil
	}

	// 获取现有参数
	params, err := s.ghostManager.GetParams(ctx, progress.LogName)
	if err != nil {
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("获取当前参数失败: %v", err),
		}, nil
	}

	// 更新可选参数
	if req.ChunkSize != nil {
		params.ChunkSize = int(*req.ChunkSize)
	}
	if req.MaxLagMillis != nil {
		params.MaxLagMillis = int(*req.MaxLagMillis)
	}
	// 使用 MaxLoad 或 CriticalLoad 来设置 ThreadsRunning
	if req.MaxLoad != nil {
		params.ThreadsRunning = int(*req.MaxLoad)
	} else if req.CriticalLoad != nil {
		params.ThreadsRunning = int(*req.CriticalLoad)
	}
	if req.NiceRatio != nil {
		params.NiceRatio = *req.NiceRatio
	}

	err = s.ghostManager.ModifyParams(ctx, tableName, req.WorkId, params)
	if err != nil {
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("修改参数失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "success",
	}, nil
}

// Stop 停止gh-ost执行
func (s *GhOstService) Stop(ctx context.Context, req *v1.StopRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if req.WorkId == "" {
		return &v1.CommonResponse{
			Code:    400,
			Message: "work_id参数必填",
		}, nil
	}

	// 从 work_id 获取表名
	progress, err := s.ghostManager.GetProgress(ctx, req.WorkId)
	if err != nil {
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("获取进度失败: %v", err),
		}, nil
	}

	// 从日志文件名提取表名
	tableName := ""
	if progress.LogName != "" {
		fileName := filepath.Base(progress.LogName)
		if idx := strings.LastIndex(fileName, "_"); idx > 0 {
			tableName = fileName[:idx]
		}
	}

	if tableName == "" {
		return &v1.CommonResponse{
			Code:    400,
			Message: "无法从日志文件提取表名",
		}, nil
	}

	err = s.ghostManager.Stop(ctx, tableName, req.WorkId)
	if err != nil {
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("停止失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "终止ghost成功,工单执行失败!",
	}, nil
}

// GetLog 获取gh-ost执行日志
func (s *GhOstService) GetLog(ctx context.Context, req *v1.GetLogRequest) (*v1.GetLogResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetLogResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if req.WorkId == "" {
		return &v1.GetLogResponse{
			Code:    400,
			Message: "work_id参数必填",
		}, nil
	}

	// 默认返回最后100行
	lines := int(req.Lines)
	if lines <= 0 {
		lines = 100
	}

	logLines, totalLines, logFile, err := s.ghostManager.GetLog(ctx, req.WorkId, lines, req.Offset)
	if err != nil {
		return &v1.GetLogResponse{
			Code:    404,
			Message: fmt.Sprintf("获取日志失败: %v", err),
		}, nil
	}

	return &v1.GetLogResponse{
		Code:       200,
		Message:    "查询成功",
		LogLines:   logLines,
		TotalLines: totalLines,
		LogFile:    logFile,
	}, nil
}
