package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	"sisyphus/server/internal/middleware"
)

// ArchiveService 数据归档服务
type ArchiveService struct {
	archiveUsecase *biz.ArchiveUsecase
	log            *log.Helper
}

// NewArchiveService 创建数据归档服务
func NewArchiveService(archiveUsecase *biz.ArchiveUsecase, logger log.Logger) *ArchiveService {
	return &ArchiveService{
		archiveUsecase: archiveUsecase,
		log:            log.NewHelper(logger),
	}
}

// ListArchiveConfigs 获取归档配置列表
func (s *ArchiveService) ListArchiveConfigs(ctx context.Context, req *v1.ListArchiveConfigsRequest) (*v1.ListArchiveConfigsResponse, error) {
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

	configs, total, err := s.archiveUsecase.ListConfigs(ctx, page, pageSize)
	if err != nil {
		return &v1.ListArchiveConfigsResponse{
			Code:    500,
			Message: fmt.Sprintf("查询失败: %v", err),
		}, nil
	}

	var protoConfigs []*v1.ArchiveConfig
	for _, config := range configs {
		protoConfigs = append(protoConfigs, toProtoArchiveConfig(config))
	}

	return &v1.ListArchiveConfigsResponse{
		Code:    200,
		Message: "查询成功",
		List:    protoConfigs,
		PageResponse: &v1.PageResponse{
			Total:    int64(total),
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
	}, nil
}

// CreateArchiveConfig 创建归档配置
func (s *ArchiveService) CreateArchiveConfig(ctx context.Context, req *v1.CreateArchiveConfigRequest) (*v1.CreateArchiveConfigResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CreateArchiveConfigResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}

	config := toBizArchiveConfig(req, claims.Username)
	config, err := s.archiveUsecase.CreateArchiveConfig(ctx, config)
	if err != nil {
		return &v1.CreateArchiveConfigResponse{
			Code:    500,
			Message: fmt.Sprintf("创建失败: %v", err),
		}, nil
	}

	return &v1.CreateArchiveConfigResponse{
		Code:    200,
		Message: "创建成功",
		Data:    toProtoArchiveConfig(config),
	}, nil
}

// GetArchiveConfig 获取归档配置详情
func (s *ArchiveService) GetArchiveConfig(ctx context.Context, req *v1.GetArchiveConfigRequest) (*v1.GetArchiveConfigResponse, error) {
	config, err := s.archiveUsecase.GetConfigByID(ctx, req.Id)
	if err != nil {
		return &v1.GetArchiveConfigResponse{
			Code:    404,
			Message: "配置不存在",
		}, nil
	}

	return &v1.GetArchiveConfigResponse{
		Code:    200,
		Message: "查询成功",
		Data:    toProtoArchiveConfig(config),
	}, nil
}

// UpdateArchiveConfig 更新归档配置
func (s *ArchiveService) UpdateArchiveConfig(ctx context.Context, req *v1.UpdateArchiveConfigRequest) (*v1.CommonResponse, error) {
	// 获取现有配置
	config, err := s.archiveUsecase.GetConfigByID(ctx, req.Id)
	if err != nil {
		return &v1.CommonResponse{
			Code:    404,
			Message: "配置不存在",
		}, nil
	}

	// 更新可选字段
	if req.ArchiveCondition != nil {
		config.ArchiveCondition = *req.ArchiveCondition
	}
	if req.IsEnabled != nil {
		config.IsEnabled = *req.IsEnabled
	}
	if req.CronExpression != nil {
		config.CronExpression = *req.CronExpression
	}
	if req.RetentionDays != nil {
		config.RetentionDays = int(*req.RetentionDays)
	}

	err = s.archiveUsecase.UpdateArchiveConfig(ctx, config)
	if err != nil {
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("更新失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "更新成功",
	}, nil
}

// DeleteArchiveConfig 删除归档配置
func (s *ArchiveService) DeleteArchiveConfig(ctx context.Context, req *v1.DeleteArchiveConfigRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	err := s.archiveUsecase.DeleteArchiveConfig(ctx, req.Id)
	if err != nil {
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("删除失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "删除成功",
	}, nil
}

// ListArchiveRecords 获取归档记录列表
func (s *ArchiveService) ListArchiveRecords(ctx context.Context, req *v1.ListArchiveRecordsRequest) (*v1.ListArchiveRecordsResponse, error) {
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

	records, total, err := s.archiveUsecase.ListRecords(ctx, req.Id, page, pageSize)
	if err != nil {
		return &v1.ListArchiveRecordsResponse{
			Code:    500,
			Message: fmt.Sprintf("查询失败: %v", err),
		}, nil
	}

	var protoRecords []*v1.ArchiveRecord
	for _, record := range records {
		protoRecords = append(protoRecords, toProtoArchiveRecord(record))
	}

	return &v1.ListArchiveRecordsResponse{
		Code:    200,
		Message: "查询成功",
		List:    protoRecords,
		PageResponse: &v1.PageResponse{
			Total:    int64(total),
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
	}, nil
}
