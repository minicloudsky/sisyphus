package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	"sisyphus/server/internal/middleware"
	"sisyphus/server/internal/pkg/export"
)

// formatValue converts a database value to a string, handling []byte properly.
func formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	if b, ok := val.([]byte); ok {
		return string(b)
	}
	return fmt.Sprintf("%v", val)
}

// QueryService 数据查询服务
type QueryService struct {
	queryExecutor     *biz.QueryExecutor
	instanceUsecase   *biz.DatabaseInstanceUsecase
	historyUsecase    *biz.QueryHistoryUsecase
	permissionUsecase *biz.PermissionUsecase
	log               *log.Helper
}

// NewQueryService 创建数据查询服务
func NewQueryService(
	queryExecutor *biz.QueryExecutor,
	instanceUsecase *biz.DatabaseInstanceUsecase,
	historyUsecase *biz.QueryHistoryUsecase,
	permissionUsecase *biz.PermissionUsecase,
	logger log.Logger,
) *QueryService {
	return &QueryService{
		queryExecutor:     queryExecutor,
		instanceUsecase:   instanceUsecase,
		historyUsecase:    historyUsecase,
		permissionUsecase: permissionUsecase,
		log:               log.NewHelper(logger),
	}
}

// ========== Protobuf 接口实现 ==========

// MySQLQuery MySQL查询（实现 Protobuf 接口）
func (s *QueryService) MySQLQuery(ctx context.Context, req *v1.MySQLQueryRequest) (*v1.MySQLQueryResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.MySQLQueryResponse{Code: 401, Message: "未认证"}, nil
	}

	// 拒绝非 SELECT 语句，防止通过查询接口执行写操作
	trimmedSQL := strings.TrimSpace(strings.ToUpper(req.Sql))
	for _, prefix := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "RENAME", "REPLACE"} {
		if strings.HasPrefix(trimmedSQL, prefix) {
			return &v1.MySQLQueryResponse{Code: 400, Message: "查询接口仅支持 SELECT 语句"}, nil
		}
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 1000
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.MySQLQueryResponse{Code: 404, Message: "实例不存在"}, nil
	}

	permOK, permErr := s.permissionUsecase.CanAccessInstance(ctx, claims.UserID, req.InstanceId, biz.InstancePermQuery)
	if permErr != nil {
		return &v1.MySQLQueryResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", permErr)}, nil
	}
	if !permOK {
		return &v1.MySQLQueryResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instanceInfo := &biz.InstanceInfo{
		Host: instance.Host, Port: instance.Port,
		Username: instance.Username, Password: instance.Password,
	}

	result, err := s.queryExecutor.ExecuteMySQLQuery(ctx, instanceInfo, req.Sql, req.Database, limit)
	if err != nil {
		return &v1.MySQLQueryResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	if s.historyUsecase != nil {
		history := &biz.QueryHistory{
			UserID: claims.UserID, InstanceID: req.InstanceId,
			DatabaseName: req.Database, SQLText: req.Sql,
			ExecuteTime: result.Time.Seconds(), ResultCount: result.Count,
		}
		_, _ = s.historyUsecase.CreateHistory(ctx, history)
	}

	protoRows := make([]*v1.QueryRow, len(result.Rows))
	for i, row := range result.Rows {
		values := make([]string, len(row))
		for j, val := range row {
			values[j] = formatValue(val)
		}
		protoRows[i] = &v1.QueryRow{Values: values}
	}

	return &v1.MySQLQueryResponse{
		Code: 200, Message: "success",
		Data: &v1.QueryResult{
			Columns: result.Columns, Rows: protoRows,
			Count: int32(result.Count), Time: result.Time.String(),
		},
	}, nil
}

// RedisQuery Redis查询（实现 Protobuf 接口）
func (s *QueryService) RedisQuery(ctx context.Context, req *v1.RedisQueryRequest) (*v1.RedisQueryResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.RedisQueryResponse{Code: 401, Message: "未认证"}, nil
	}
	_ = claims

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.RedisQueryResponse{Code: 404, Message: "实例不存在"}, nil
	}

	permOK, permErr := s.permissionUsecase.CanAccessInstance(ctx, claims.UserID, req.InstanceId, biz.InstancePermQuery)
	if permErr != nil {
		return &v1.RedisQueryResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", permErr)}, nil
	}
	if !permOK {
		return &v1.RedisQueryResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instanceInfo := &biz.InstanceInfo{
		Host: instance.Host, Port: instance.Port,
		Username: instance.Username, Password: instance.Password,
	}

	result, err := s.queryExecutor.ExecuteRedisQuery(ctx, instanceInfo, req.Command)
	if err != nil {
		return &v1.RedisQueryResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &v1.RedisQueryResponse{Code: 500, Message: fmt.Sprintf("序列化结果失败: %v", err)}, nil
	}

	return &v1.RedisQueryResponse{Code: 200, Message: "success", Data: string(resultJSON)}, nil
}

// HBaseQuery HBase查询（实现 Protobuf 接口）
func (s *QueryService) HBaseQuery(ctx context.Context, req *v1.HBaseQueryRequest) (*v1.HBaseQueryResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.HBaseQueryResponse{Code: 401, Message: "未认证"}, nil
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 1000
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.HBaseQueryResponse{Code: 404, Message: "实例不存在"}, nil
	}

	permOK, permErr := s.permissionUsecase.CanAccessInstance(ctx, claims.UserID, req.InstanceId, biz.InstancePermQuery)
	if permErr != nil {
		return &v1.HBaseQueryResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", permErr)}, nil
	}
	if !permOK {
		return &v1.HBaseQueryResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instanceInfo := &biz.InstanceInfo{
		Host: instance.Host, Port: instance.Port,
		Username: instance.Username, Password: instance.Password,
	}

	result, err := s.queryExecutor.ExecuteHBaseQuery(ctx, instanceInfo, req.Sql, limit)
	if err != nil {
		return &v1.HBaseQueryResponse{Code: 500, Message: fmt.Sprintf("HBase查询失败: %v", err)}, nil
	}

	if s.historyUsecase != nil {
		history := &biz.QueryHistory{
			UserID: claims.UserID, InstanceID: req.InstanceId,
			DatabaseName: "hbase", SQLText: req.Sql,
			ExecuteTime: result.Time.Seconds(), ResultCount: result.Count,
		}
		_, _ = s.historyUsecase.CreateHistory(ctx, history)
	}

	protoRows := make([]*v1.QueryRow, len(result.Rows))
	for i, row := range result.Rows {
		values := make([]string, len(row))
		for j, val := range row {
			values[j] = formatValue(val)
		}
		protoRows[i] = &v1.QueryRow{Values: values}
	}

	return &v1.HBaseQueryResponse{
		Code: 200, Message: "success",
		Data: &v1.QueryResult{
			Columns: result.Columns, Rows: protoRows,
			Count: int32(result.Count), Time: result.Time.String(),
		},
	}, nil
}

// ClickHouseQuery ClickHouse查询（实现 Protobuf 接口）
func (s *QueryService) ClickHouseQuery(ctx context.Context, req *v1.ClickHouseQueryRequest) (*v1.ClickHouseQueryResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ClickHouseQueryResponse{Code: 401, Message: "未认证"}, nil
	}
	_ = claims

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.ClickHouseQueryResponse{Code: 404, Message: "实例不存在"}, nil
	}

	permOK, permErr := s.permissionUsecase.CanAccessInstance(ctx, claims.UserID, req.InstanceId, biz.InstancePermQuery)
	if permErr != nil {
		return &v1.ClickHouseQueryResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", permErr)}, nil
	}
	if !permOK {
		return &v1.ClickHouseQueryResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instanceInfo := &biz.InstanceInfo{
		Host: instance.Host, Port: instance.Port,
		Username: instance.Username, Password: instance.Password,
	}

	result, err := s.queryExecutor.ExecuteClickHouseQuery(ctx, instanceInfo, req.Sql, req.Database)
	if err != nil {
		return &v1.ClickHouseQueryResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	protoRows := make([]*v1.QueryRow, len(result.Rows))
	for i, row := range result.Rows {
		values := make([]string, len(row))
		for j, val := range row {
			values[j] = formatValue(val)
		}
		protoRows[i] = &v1.QueryRow{Values: values}
	}

	return &v1.ClickHouseQueryResponse{
		Code: 200, Message: "success",
		Data: &v1.QueryResult{
			Columns: result.Columns, Rows: protoRows,
			Count: int32(result.Count), Time: result.Time.String(),
		},
	}, nil
}

// ExportQuery 导出查询结果（实现 Protobuf 接口）
func (s *QueryService) ExportQuery(ctx context.Context, req *v1.ExportQueryRequest) (*v1.ExportQueryResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ExportQueryResponse{Code: 401, Message: "未认证"}, nil
	}
	_ = claims

	if req.Format != "excel" && req.Format != "csv" && req.Format != "json" && req.Format != "sql" {
		return &v1.ExportQueryResponse{Code: 400, Message: "不支持的导出格式，支持: excel, csv, json, sql"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.ExportQueryResponse{Code: 404, Message: "实例不存在"}, nil
	}

	permOK, permErr := s.permissionUsecase.CanAccessInstance(ctx, claims.UserID, req.InstanceId, biz.InstancePermExport)
	if permErr != nil {
		return &v1.ExportQueryResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", permErr)}, nil
	}
	if !permOK {
		return &v1.ExportQueryResponse{Code: 403, Message: "无该实例的导出权限"}, nil
	}

	dbType := strings.ToLower(strings.TrimSpace(instance.DBType))
	if req.Format == "sql" && dbType != "mysql" {
		return &v1.ExportQueryResponse{Code: 400, Message: "INSERT SQL 导出仅支持 MySQL 实例"}, nil
	}
	if dbType != "mysql" && dbType != "clickhouse" {
		return &v1.ExportQueryResponse{Code: 400, Message: "该实例类型不支持查询结果导出"}, nil
	}

	if req.Format == "sql" {
		if err := export.ValidateExportTableName(req.GetTableName()); err != nil {
			return &v1.ExportQueryResponse{Code: 400, Message: err.Error()}, nil
		}
	}

	instanceInfo := &biz.InstanceInfo{
		Host: instance.Host, Port: instance.Port,
		Username: instance.Username, Password: instance.Password,
	}

	var result *biz.QueryResult
	if dbType == "clickhouse" {
		result, err = s.queryExecutor.ExecuteClickHouseQuery(ctx, instanceInfo, req.Sql, req.Database)
	} else {
		result, err = s.queryExecutor.ExecuteMySQLQuery(ctx, instanceInfo, req.Sql, req.Database, 10000)
	}
	if err != nil {
		return &v1.ExportQueryResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	exportData := &export.ExportData{Columns: result.Columns, Rows: result.Rows}

	var fileFormat string
	switch req.Format {
	case "excel":
		fileFormat = "xlsx"
	case "csv":
		fileFormat = "csv"
	case "json":
		fileFormat = "json"
	case "sql":
		fileFormat = "sql"
	}
	filePath := export.GetExportFilePath(fileFormat)

	var exportErr error
	switch req.Format {
	case "excel":
		exportErr = export.ExportToExcel(exportData, filePath)
	case "csv":
		exportErr = export.ExportToCSV(exportData, filePath)
	case "json":
		exportErr = export.ExportToJSON(exportData, filePath)
	case "sql":
		exportErr = export.ExportToInsertSQL(exportData, filePath, req.GetTableName())
	}

	if exportErr != nil {
		return &v1.ExportQueryResponse{Code: 500, Message: fmt.Sprintf("导出失败: %v", exportErr)}, nil
	}

	fileName := filepath.Base(filePath)
	return &v1.ExportQueryResponse{
		Code: 200, Message: "success",
		Data: &v1.ExportData{
			FileName: fileName, FilePath: filePath,
			DownloadUrl: fmt.Sprintf("/api/v1/query/download/%s", fileName),
		},
	}, nil
}

// ListHistory 查询历史列表（实现 Protobuf 接口）
func (s *QueryService) ListHistory(ctx context.Context, req *v1.ListHistoryRequest) (*v1.ListHistoryResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ListHistoryResponse{Code: 401, Message: "未认证"}, nil
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}

	var instanceID *uint64
	if req.InstanceId > 0 {
		instanceID = &req.InstanceId
	}

	histories, total, err := s.historyUsecase.ListHistory(ctx, claims.UserID, page, pageSize, instanceID)
	if err != nil {
		return &v1.ListHistoryResponse{Code: 500, Message: fmt.Sprintf("获取查询历史失败: %v", err)}, nil
	}

	protoHistories := make([]*v1.QueryHistory, len(histories))
	for i, h := range histories {
		protoHistories[i] = &v1.QueryHistory{
			Id: h.ID, UserId: h.UserID, InstanceId: h.InstanceID,
			DatabaseName: h.DatabaseName, SqlText: h.SQLText,
			ExecuteTime: h.ExecuteTime, AffectedRows: int32(h.AffectedRows),
			ResultCount: int32(h.ResultCount), CreatedAt: h.CreatedAt.Format(time.RFC3339),
		}
	}

	return &v1.ListHistoryResponse{
		Code: 200, Message: "success",
		Data: &v1.ListHistoryData{
			List: protoHistories,
			Page: &v1.PageResponse{Page: int32(page), PageSize: int32(pageSize), Total: total},
		},
	}, nil
}

// DeleteHistory 删除查询历史（实现 Protobuf 接口）
func (s *QueryService) DeleteHistory(ctx context.Context, req *v1.DeleteHistoryRequest) (*v1.DeleteHistoryResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.DeleteHistoryResponse{Code: 401, Message: "未认证"}, nil
	}

	history, err := s.historyUsecase.GetHistory(ctx, req.Id)
	if err != nil {
		return &v1.DeleteHistoryResponse{Code: 404, Message: "查询历史不存在"}, nil
	}
	if history.UserID != claims.UserID {
		return &v1.DeleteHistoryResponse{Code: 403, Message: "无权删除他人的查询历史"}, nil
	}

	if err := s.historyUsecase.DeleteHistory(ctx, req.Id); err != nil {
		return &v1.DeleteHistoryResponse{Code: 500, Message: fmt.Sprintf("删除查询历史失败: %v", err)}, nil
	}

	return &v1.DeleteHistoryResponse{Code: 200, Message: "删除成功"}, nil
}
