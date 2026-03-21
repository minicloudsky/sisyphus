package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	"sisyphus/server/internal/middleware"
)

// WidetableService 宽表服务
type WidetableService struct {
	v1.UnimplementedWidetableServiceServer

	uc  *biz.WidetableUsecase
	log *log.Helper
}

// NewWidetableService 创建宽表服务
func NewWidetableService(uc *biz.WidetableUsecase, logger log.Logger) *WidetableService {
	return &WidetableService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

// CreateConfig 创建宽表配置
func (s *WidetableService) CreateConfig(ctx context.Context, req *v1.CreateWidetableConfigRequest) (*v1.CreateWidetableConfigResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		return &v1.CreateWidetableConfigResponse{Code: 401, Message: "未认证"}, nil
	}

	config := &biz.WidetableConfig{
		Name:             req.Name,
		Description:      req.Description,
		SourceInstanceID: req.SourceInstanceId,
		MainDatabase:     req.MainDatabase,
		MainTable:        req.MainTable,
		JoinConfig:       req.JoinConfig,
		FieldConfig:      req.FieldConfig,
		SyncCondition:    req.SyncCondition,
		TargetInstanceID: req.TargetInstanceId,
		TargetDatabase:   req.TargetDatabase,
		TargetTable:      req.TargetTable,
		CreatedBy:        userID,
	}

	result, err := s.uc.CreateConfig(ctx, config)
	if err != nil {
		return &v1.CreateWidetableConfigResponse{Code: 500, Message: fmt.Sprintf("创建失败: %v", err)}, nil
	}

	return &v1.CreateWidetableConfigResponse{Code: 200, Message: "success", Id: result.ID}, nil
}

// UpdateConfig 更新宽表配置
func (s *WidetableService) UpdateConfig(ctx context.Context, req *v1.UpdateWidetableConfigRequest) (*v1.CommonResponse, error) {
	_, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{Code: 401, Message: "未认证"}, nil
	}

	config := &biz.WidetableConfig{
		ID:               req.Id,
		Name:             req.Name,
		Description:      req.Description,
		SourceInstanceID: req.SourceInstanceId,
		MainDatabase:     req.MainDatabase,
		MainTable:        req.MainTable,
		JoinConfig:       req.JoinConfig,
		FieldConfig:      req.FieldConfig,
		SyncCondition:    req.SyncCondition,
		TargetInstanceID: req.TargetInstanceId,
		TargetDatabase:   req.TargetDatabase,
		TargetTable:      req.TargetTable,
	}

	if err := s.uc.UpdateConfig(ctx, config); err != nil {
		return &v1.CommonResponse{Code: 500, Message: fmt.Sprintf("更新失败: %v", err)}, nil
	}

	return &v1.CommonResponse{Code: 200, Message: "success"}, nil
}

// DeleteConfig 删除宽表配置
func (s *WidetableService) DeleteConfig(ctx context.Context, req *v1.DeleteWidetableConfigRequest) (*v1.CommonResponse, error) {
	_, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{Code: 401, Message: "未认证"}, nil
	}

	if err := s.uc.DeleteConfig(ctx, req.Id); err != nil {
		return &v1.CommonResponse{Code: 500, Message: fmt.Sprintf("删除失败: %v", err)}, nil
	}

	return &v1.CommonResponse{Code: 200, Message: "success"}, nil
}

// GetConfig 获取宽表配置详情
func (s *WidetableService) GetConfig(ctx context.Context, req *v1.GetWidetableConfigRequest) (*v1.GetWidetableConfigResponse, error) {
	_, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetWidetableConfigResponse{Code: 401, Message: "未认证"}, nil
	}

	config, err := s.uc.GetConfig(ctx, req.Id)
	if err != nil {
		return &v1.GetWidetableConfigResponse{Code: 404, Message: fmt.Sprintf("配置不存在: %v", err)}, nil
	}

	return &v1.GetWidetableConfigResponse{
		Code:    200,
		Message: "success",
		Data:    toProtoWidetableConfig(config),
	}, nil
}

// ListConfigs 获取宽表配置列表
func (s *WidetableService) ListConfigs(ctx context.Context, req *v1.ListWidetableConfigsRequest) (*v1.ListWidetableConfigsResponse, error) {
	_, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ListWidetableConfigsResponse{Code: 401, Message: "未认证"}, nil
	}

	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	configs, total, err := s.uc.ListConfigs(ctx, page, pageSize, req.Keyword)
	if err != nil {
		return &v1.ListWidetableConfigsResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	protoConfigs := make([]*v1.WidetableConfig, len(configs))
	for i, c := range configs {
		protoConfigs[i] = toProtoWidetableConfig(c)
	}

	return &v1.ListWidetableConfigsResponse{
		Code:    200,
		Message: "success",
		List:    protoConfigs,
		Page: &v1.PageResponse{
			Total:    int64(total),
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
	}, nil
}

// PreviewSQL 预览宽表查询SQL
func (s *WidetableService) PreviewSQL(ctx context.Context, req *v1.PreviewWidetableSQLRequest) (*v1.PreviewWidetableSQLResponse, error) {
	_, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.PreviewWidetableSQLResponse{Code: 401, Message: "未认证"}, nil
	}

	config, err := s.uc.GetConfig(ctx, req.Id)
	if err != nil {
		return &v1.PreviewWidetableSQLResponse{Code: 404, Message: "配置不存在"}, nil
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}

	sql, err := s.uc.BuildSQL(config, "", "", limit, 0)
	if err != nil {
		return &v1.PreviewWidetableSQLResponse{Code: 500, Message: fmt.Sprintf("构建SQL失败: %v", err)}, nil
	}

	return &v1.PreviewWidetableSQLResponse{Code: 200, Message: "success", Sql: sql}, nil
}

// ExecuteQuery 执行宽表查询
func (s *WidetableService) ExecuteQuery(ctx context.Context, req *v1.ExecuteWidetableQueryRequest) (*v1.ExecuteWidetableQueryResponse, error) {
	_, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ExecuteWidetableQueryResponse{Code: 401, Message: "未认证"}, nil
	}

	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 50
	}

	result, err := s.uc.ExecuteQuery(ctx, req.Id, page, pageSize, req.ExtraCondition, req.OrderBy)
	if err != nil {
		return &v1.ExecuteWidetableQueryResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	// 转换rows
	protoRows := make([]*v1.QueryRow, len(result.Rows))
	for i, row := range result.Rows {
		protoRows[i] = &v1.QueryRow{Values: row}
	}

	return &v1.ExecuteWidetableQueryResponse{
		Code:        200,
		Message:     "success",
		Columns:     result.Columns,
		Rows:        protoRows,
		Total:       result.Total,
		ExecuteTime: result.ExecuteTime,
	}, nil
}

// StartFullSync 启动全量同步
func (s *WidetableService) StartFullSync(ctx context.Context, req *v1.StartWidetableSyncRequest) (*v1.StartWidetableSyncResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		return &v1.StartWidetableSyncResponse{Code: 401, Message: "未认证"}, nil
	}

	record, err := s.uc.StartFullSync(ctx, req.Id, int(req.BatchSize), userID)
	if err != nil {
		return &v1.StartWidetableSyncResponse{Code: 500, Message: fmt.Sprintf("启动同步失败: %v", err)}, nil
	}

	return &v1.StartWidetableSyncResponse{
		Code:     200,
		Message:  "同步任务已启动",
		RecordId: record.ID,
	}, nil
}

// ListSyncRecords 获取同步记录
func (s *WidetableService) ListSyncRecords(ctx context.Context, req *v1.ListWidetableSyncRecordsRequest) (*v1.ListWidetableSyncRecordsResponse, error) {
	_, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ListWidetableSyncRecordsResponse{Code: 401, Message: "未认证"}, nil
	}

	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	records, total, err := s.uc.ListSyncRecords(ctx, req.ConfigId, page, pageSize)
	if err != nil {
		return &v1.ListWidetableSyncRecordsResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	protoRecords := make([]*v1.WidetableSyncRecord, len(records))
	for i, r := range records {
		protoRecords[i] = toProtoSyncRecord(r)
	}

	return &v1.ListWidetableSyncRecordsResponse{
		Code:    200,
		Message: "success",
		List:    protoRecords,
		Page: &v1.PageResponse{
			Total:    int64(total),
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
	}, nil
}

func toProtoWidetableConfig(c *biz.WidetableConfig) *v1.WidetableConfig {
	return &v1.WidetableConfig{
		Id:               c.ID,
		Name:             c.Name,
		Description:      c.Description,
		SourceInstanceId: c.SourceInstanceID,
		MainDatabase:     c.MainDatabase,
		MainTable:        c.MainTable,
		JoinConfig:       c.JoinConfig,
		FieldConfig:      c.FieldConfig,
		SyncCondition:    c.SyncCondition,
		TargetInstanceId: c.TargetInstanceID,
		TargetDatabase:   c.TargetDatabase,
		TargetTable:      c.TargetTable,
		Status:           int32(c.Status),
		CreatedBy:        c.CreatedBy,
		CreatedAt:        c.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:        c.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func toProtoSyncRecord(r *biz.WidetableSyncRecord) *v1.WidetableSyncRecord {
	var startedAt, finishedAt string
	if !r.StartedAt.IsZero() {
		startedAt = r.StartedAt.Format("2006-01-02 15:04:05")
	}
	if !r.FinishedAt.IsZero() {
		finishedAt = r.FinishedAt.Format("2006-01-02 15:04:05")
	}
	return &v1.WidetableSyncRecord{
		Id:            r.ID,
		ConfigId:      r.ConfigID,
		SyncType:      int32(r.SyncType),
		Status:        int32(r.Status),
		ProcessedRows: r.ProcessedRows,
		ErrorMessage:  r.ErrorMessage,
		CostSeconds:   r.CostSeconds,
		StartedAt:     startedAt,
		FinishedAt:    finishedAt,
		CreatedAt:     r.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
