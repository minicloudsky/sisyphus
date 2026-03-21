package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	"sisyphus/server/internal/middleware"
)

// InstanceService 数据库实例管理服务
type InstanceService struct {
	instanceUsecase   *biz.DatabaseInstanceUsecase
	permissionUsecase *biz.PermissionUsecase
	log               *log.Helper
}

// NewInstanceService 创建实例管理服务
func NewInstanceService(instanceUsecase *biz.DatabaseInstanceUsecase, permissionUsecase *biz.PermissionUsecase, logger log.Logger) *InstanceService {
	return &InstanceService{
		instanceUsecase:   instanceUsecase,
		permissionUsecase: permissionUsecase,
		log:               log.NewHelper(logger),
	}
}

// ListInstances 获取实例列表
func (s *InstanceService) ListInstances(ctx context.Context, req *v1.ListInstancesRequest) (*v1.ListInstancesResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ListInstancesResponse{Code: 401, Message: "未认证"}, nil
	}

	pageReq := req.GetPageRequest()
	page := int(pageReq.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(pageReq.GetPageSize())
	if pageSize < 1 {
		pageSize = 10
	}

	var dbType *string
	if req.DbType != nil {
		dbType = req.DbType
	}

	var env *string
	if req.Env != nil {
		env = req.Env
	}

	var scope *string
	if req.InstancePermissionScope != nil && *req.InstancePermissionScope != "" {
		v := *req.InstancePermissionScope
		scope = &v
	}
	// 用户/角色管理员需要看到全量实例以便配置实例级授权；否则在「已启用实例 ACL 且本人无直授」时会出现空列表，
	// 而带 instance_permission_scope 的查询（如数据查询）仍可能有数据。
	var restrict bool
	var allowedIDs []uint64
	canManageUsers, errU := s.permissionUsecase.CheckPermission(ctx, claims.UserID, "user", "manage")
	canManageRoles, errR := s.permissionUsecase.CheckPermission(ctx, claims.UserID, "role", "manage")
	if (errU == nil && canManageUsers) || (errR == nil && canManageRoles) {
		restrict = false
		allowedIDs = nil
	} else {
		var err error
		restrict, allowedIDs, err = s.permissionUsecase.ResolveInstanceListFilter(ctx, claims.UserID, scope)
		if err != nil {
			return &v1.ListInstancesResponse{Code: 500, Message: fmt.Sprintf("权限解析失败: %v", err)}, nil
		}
	}

	instances, total, err := s.instanceUsecase.ListInstances(ctx, page, pageSize, dbType, env, restrict, allowedIDs)
	if err != nil {
		return &v1.ListInstancesResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	protoInstances := make([]*v1.DatabaseInstance, len(instances))
	for i, inst := range instances {
		protoInstances[i] = toProtoDatabaseInstance(inst)
	}

	return &v1.ListInstancesResponse{
		Code:    200,
		Message: "success",
		List:    protoInstances,
		PageResponse: &v1.PageResponse{
			Page:     int32(page),
			PageSize: int32(pageSize),
			Total:    total,
		},
	}, nil
}

// CreateInstance 创建实例
func (s *InstanceService) CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest) (*v1.CreateInstanceResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CreateInstanceResponse{Code: 401, Message: "未认证"}, nil
	}
	_ = claims

	instance := toBizDatabaseInstance(req)
	instance, err := s.instanceUsecase.CreateInstance(ctx, instance)
	if err != nil {
		return &v1.CreateInstanceResponse{Code: 500, Message: fmt.Sprintf("创建失败: %v", err)}, nil
	}

	return &v1.CreateInstanceResponse{
		Code:    200,
		Message: "success",
		Data:    toProtoDatabaseInstance(instance),
	}, nil
}

// GetInstance 获取实例详情
func (s *InstanceService) GetInstance(ctx context.Context, req *v1.GetInstanceRequest) (*v1.GetInstanceResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetInstanceResponse{Code: 401, Message: "未认证"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.Id)
	if err != nil {
		return &v1.GetInstanceResponse{Code: 404, Message: "实例不存在"}, nil
	}

	viewOK, err := s.permissionUsecase.CanViewInstanceMeta(ctx, claims.UserID, req.Id)
	if err != nil {
		return &v1.GetInstanceResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !viewOK {
		return &v1.GetInstanceResponse{Code: 403, Message: "无权限查看该实例"}, nil
	}

	protoInstance := toProtoDatabaseInstance(instance)
	protoInstance.Password = "" // 隐藏密码

	return &v1.GetInstanceResponse{Code: 200, Message: "success", Data: protoInstance}, nil
}

// UpdateInstance 更新实例
func (s *InstanceService) UpdateInstance(ctx context.Context, req *v1.UpdateInstanceRequest) (*v1.CommonResponse, error) {
	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.Id)
	if err != nil {
		return &v1.CommonResponse{Code: 404, Message: "实例不存在"}, nil
	}

	if req.Name != nil {
		instance.Name = *req.Name
	}
	if req.Host != nil {
		instance.Host = *req.Host
	}
	if req.Port != nil {
		instance.Port = int(*req.Port)
	}
	if req.Username != nil {
		instance.Username = *req.Username
	}
	if req.Password != nil {
		instance.Password = *req.Password
	}
	if req.Env != nil {
		instance.Env = *req.Env
	}
	if req.Description != nil {
		instance.Description = *req.Description
	}
	if req.IsEnabled != nil {
		if *req.IsEnabled {
			instance.Status = 1
		} else {
			instance.Status = 0
		}
	}

	err = s.instanceUsecase.UpdateInstance(ctx, instance)
	if err != nil {
		return &v1.CommonResponse{Code: 500, Message: fmt.Sprintf("更新失败: %v", err)}, nil
	}

	return &v1.CommonResponse{Code: 200, Message: "更新成功"}, nil
}

// DeleteInstance 删除实例
func (s *InstanceService) DeleteInstance(ctx context.Context, req *v1.DeleteInstanceRequest) (*v1.CommonResponse, error) {
	err := s.instanceUsecase.DeleteInstance(ctx, req.Id)
	if err != nil {
		return &v1.CommonResponse{Code: 500, Message: fmt.Sprintf("删除失败: %v", err)}, nil
	}

	return &v1.CommonResponse{Code: 200, Message: "删除成功"}, nil
}

// TestConnection 测试连接
func (s *InstanceService) TestConnection(ctx context.Context, req *v1.TestConnectionRequest) (*v1.TestConnectionResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.TestConnectionResponse{Code: 401, Message: "未认证", Success: false}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.Id)
	if err != nil {
		return &v1.TestConnectionResponse{Code: 404, Message: "实例不存在", Success: false}, nil
	}

	viewOK, err := s.permissionUsecase.CanViewInstanceMeta(ctx, claims.UserID, req.Id)
	if err != nil {
		return &v1.TestConnectionResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err), Success: false}, nil
	}
	if !viewOK {
		return &v1.TestConnectionResponse{Code: 403, Message: "无权限操作该实例", Success: false}, nil
	}

	err = s.instanceUsecase.TestConnection(ctx, instance)
	if err != nil {
		return &v1.TestConnectionResponse{Code: 500, Message: fmt.Sprintf("连接测试失败: %v", err), Success: false}, nil
	}

	return &v1.TestConnectionResponse{Code: 200, Message: "连接测试成功", Success: true}, nil
}

// ListDatabases 获取数据库列表
func (s *InstanceService) ListDatabases(ctx context.Context, req *v1.ListDatabasesRequest) (*v1.ListDatabasesResponse, error) {
	claims, authed := middleware.FromContext(ctx)
	if !authed {
		return &v1.ListDatabasesResponse{Code: 401, Message: "未认证"}, nil
	}
	queryOK, err := s.instanceQueryAllowed(ctx, claims.UserID, req.InstanceId)
	if err != nil {
		return &v1.ListDatabasesResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !queryOK {
		return &v1.ListDatabasesResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.ListDatabasesResponse{Code: 404, Message: "实例不存在"}, nil
	}

	databases, err := s.instanceUsecase.ListDatabases(ctx, instance)
	if err != nil {
		return &v1.ListDatabasesResponse{Code: 500, Message: fmt.Sprintf("获取数据库列表失败: %v", err)}, nil
	}

	return &v1.ListDatabasesResponse{
		Code:     200,
		Message:  "success",
		Databases: databases,
	}, nil
}

// ListTables 获取表列表
func (s *InstanceService) ListTables(ctx context.Context, req *v1.ListTablesRequest) (*v1.ListTablesResponse, error) {
	claims, authed := middleware.FromContext(ctx)
	if !authed {
		return &v1.ListTablesResponse{Code: 401, Message: "未认证"}, nil
	}
	queryOK, err := s.instanceQueryAllowed(ctx, claims.UserID, req.InstanceId)
	if err != nil {
		return &v1.ListTablesResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !queryOK {
		return &v1.ListTablesResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.ListTablesResponse{Code: 404, Message: "实例不存在"}, nil
	}

	tables, err := s.instanceUsecase.ListTables(ctx, instance, req.Database)
	if err != nil {
		return &v1.ListTablesResponse{Code: 500, Message: fmt.Sprintf("获取表列表失败: %v", err)}, nil
	}

	return &v1.ListTablesResponse{
		Code:    200,
		Message: "success",
		Tables:  tables,
	}, nil
}

// GetTableDDL 获取表DDL
func (s *InstanceService) GetTableDDL(ctx context.Context, req *v1.GetTableDDLRequest) (*v1.GetTableDDLResponse, error) {
	claims, authed := middleware.FromContext(ctx)
	if !authed {
		return &v1.GetTableDDLResponse{Code: 401, Message: "未认证"}, nil
	}
	queryOK, err := s.instanceQueryAllowed(ctx, claims.UserID, req.InstanceId)
	if err != nil {
		return &v1.GetTableDDLResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !queryOK {
		return &v1.GetTableDDLResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.GetTableDDLResponse{Code: 404, Message: "实例不存在"}, nil
	}

	ddl, err := s.instanceUsecase.GetTableDDL(ctx, instance, req.Database, req.Table)
	if err != nil {
		return &v1.GetTableDDLResponse{Code: 500, Message: fmt.Sprintf("获取表DDL失败: %v", err)}, nil
	}

	return &v1.GetTableDDLResponse{
		Code:    200,
		Message: "success",
		Ddl:     ddl,
	}, nil
}

// GetExplain 获取执行计划
func (s *InstanceService) GetExplain(ctx context.Context, req *v1.GetExplainRequest) (*v1.GetExplainResponse, error) {
	claims, authed := middleware.FromContext(ctx)
	if !authed {
		return &v1.GetExplainResponse{Code: 401, Message: "未认证"}, nil
	}
	queryOK, err := s.instanceQueryAllowed(ctx, claims.UserID, req.InstanceId)
	if err != nil {
		return &v1.GetExplainResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !queryOK {
		return &v1.GetExplainResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.GetExplainResponse{Code: 404, Message: "实例不存在"}, nil
	}

	explain, err := s.instanceUsecase.GetExplain(ctx, instance, req.Database, req.Sql)
	if err != nil {
		return &v1.GetExplainResponse{Code: 500, Message: fmt.Sprintf("获取执行计划失败: %v", err)}, nil
	}

	return &v1.GetExplainResponse{
		Code:    200,
		Message: "success",
		Explain: explain,
	}, nil
}

// GetProcessList 获取进程列表
func (s *InstanceService) GetProcessList(ctx context.Context, req *v1.GetProcessListRequest) (*v1.GetProcessListResponse, error) {
	claims, authed := middleware.FromContext(ctx)
	if !authed {
		return &v1.GetProcessListResponse{Code: 401, Message: "未认证"}, nil
	}
	queryOK, err := s.instanceQueryAllowed(ctx, claims.UserID, req.InstanceId)
	if err != nil {
		return &v1.GetProcessListResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !queryOK {
		return &v1.GetProcessListResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.GetProcessListResponse{Code: 404, Message: "实例不存在"}, nil
	}

	processes, err := s.instanceUsecase.GetProcessList(ctx, instance)
	if err != nil {
		return &v1.GetProcessListResponse{Code: 500, Message: fmt.Sprintf("获取进程列表失败: %v", err)}, nil
	}

	var protoProcesses []*v1.ProcessInfoItem
	for _, p := range processes {
		protoProcesses = append(protoProcesses, &v1.ProcessInfoItem{
			Id:       p.ID,
			User:     p.User,
			Host:     p.Host,
			Db:       p.DB,
			Command:  p.Command,
			Time:     p.Time,
			State:    p.State,
			Info:     p.Info,
			QueryId:  p.QueryID,
			Progress: p.Progress,
		})
	}

	return &v1.GetProcessListResponse{
		Code:      200,
		Message:   "success",
		Processes: protoProcesses,
	}, nil
}

// KillProcess 杀死进程
func (s *InstanceService) KillProcess(ctx context.Context, req *v1.KillProcessRequest) (*v1.KillProcessResponse, error) {
	claims, authed := middleware.FromContext(ctx)
	if !authed {
		return &v1.KillProcessResponse{Code: 401, Message: "未认证"}, nil
	}
	queryOK, err := s.instanceQueryAllowed(ctx, claims.UserID, req.InstanceId)
	if err != nil {
		return &v1.KillProcessResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !queryOK {
		return &v1.KillProcessResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.KillProcessResponse{Code: 404, Message: "实例不存在"}, nil
	}

	if err := s.instanceUsecase.KillProcess(ctx, instance, req.ProcessId); err != nil {
		return &v1.KillProcessResponse{Code: 500, Message: fmt.Sprintf("终止进程失败: %v", err)}, nil
	}

	return &v1.KillProcessResponse{Code: 200, Message: "已发送终止信号"}, nil
}

// GetTableColumns 获取表列信息
func (s *InstanceService) GetTableColumns(ctx context.Context, req *v1.GetTableColumnsRequest) (*v1.GetTableColumnsResponse, error) {
	claims, authed := middleware.FromContext(ctx)
	if !authed {
		return &v1.GetTableColumnsResponse{Code: 401, Message: "未认证"}, nil
	}
	queryOK, err := s.instanceQueryAllowed(ctx, claims.UserID, req.InstanceId)
	if err != nil {
		return &v1.GetTableColumnsResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !queryOK {
		return &v1.GetTableColumnsResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.GetTableColumnsResponse{Code: 404, Message: "实例不存在"}, nil
	}

	columns, err := s.instanceUsecase.GetTableColumns(ctx, instance, req.Database, req.Table)
	if err != nil {
		return &v1.GetTableColumnsResponse{Code: 500, Message: fmt.Sprintf("获取表列信息失败: %v", err)}, nil
	}

	var protoColumns []*v1.TableColumnItem
	for _, col := range columns {
		protoColumns = append(protoColumns, &v1.TableColumnItem{
			Name:    col["name"],
			Type:    col["type"],
			Null:    col["null"],
			Key:     col["key"],
			Default: col["default"],
			Extra:   col["extra"],
			Comment: col["comment"],
		})
	}

	return &v1.GetTableColumnsResponse{
		Code:    200,
		Message: "success",
		Columns: protoColumns,
	}, nil
}

// GetAdvisorSuggestions 获取SQL优化建议
func (s *InstanceService) GetAdvisorSuggestions(ctx context.Context, req *v1.GetAdvisorSuggestionsRequest) (*v1.GetAdvisorSuggestionsResponse, error) {
	claims, authed := middleware.FromContext(ctx)
	if !authed {
		return &v1.GetAdvisorSuggestionsResponse{Code: 401, Message: "未认证"}, nil
	}
	queryOK, err := s.instanceQueryAllowed(ctx, claims.UserID, req.InstanceId)
	if err != nil {
		return &v1.GetAdvisorSuggestionsResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !queryOK {
		return &v1.GetAdvisorSuggestionsResponse{Code: 403, Message: "无该实例的数据查询权限"}, nil
	}

	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.GetAdvisorSuggestionsResponse{Code: 404, Message: "实例不存在"}, nil
	}

	// 获取EXPLAIN结果
	explainResult := ""
	if req.Sql != "" {
		explain, err := s.instanceUsecase.GetExplain(ctx, instance, req.Database, req.Sql)
		if err == nil {
			explainResult = explain
		}
	}

	// 获取表列信息（如果指定了表名）
	var columns []map[string]string
	indexes := ""
	if req.Table != "" {
		columns, _ = s.instanceUsecase.GetTableColumns(ctx, instance, req.Database, req.Table)
		indexes, _ = s.instanceUsecase.GetTableIndexes(ctx, instance, req.Database, req.Table)
	}

	advisor := biz.NewSQLAdvisor()
	suggestions := advisor.GetAdvisorSuggestions(ctx, instance.DBType, req.Sql, explainResult, columns, indexes)

	var protoSuggestions []*v1.AdvisorSuggestionItem
	for _, item := range suggestions {
		protoSuggestions = append(protoSuggestions, &v1.AdvisorSuggestionItem{
			Level:   item.Level,
			Rule:    item.Rule,
			Message: item.Message,
			Detail:  item.Detail,
		})
	}

	return &v1.GetAdvisorSuggestionsResponse{
		Code:        200,
		Message:     "success",
		Suggestions: protoSuggestions,
	}, nil
}

func (s *InstanceService) instanceQueryAllowed(ctx context.Context, userID, instanceID uint64) (bool, error) {
	return s.permissionUsecase.CanAccessInstance(ctx, userID, instanceID, biz.InstancePermQuery)
}
