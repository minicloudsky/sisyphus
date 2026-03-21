package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	"sisyphus/server/internal/middleware"
)

// PermissionService 权限管理服务
type PermissionService struct {
	permissionUsecase *biz.PermissionUsecase
	log               *log.Helper
}

// NewPermissionService 创建权限管理服务
func NewPermissionService(permissionUsecase *biz.PermissionUsecase, logger log.Logger) *PermissionService {
	return &PermissionService{
		permissionUsecase: permissionUsecase,
		log:               log.NewHelper(logger),
	}
}

// CreatePermission 创建权限
func (s *PermissionService) CreatePermission(ctx context.Context, req *v1.CreatePermissionRequest) (*v1.CreatePermissionResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CreatePermissionResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	permission := &biz.Permission{
		Name:        req.Name,
		Code:        req.Code,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
	}

	result, err := s.permissionUsecase.CreatePermission(ctx, permission)
	if err != nil {
		s.log.Errorf("创建权限失败: %v", err)
		return &v1.CreatePermissionResponse{
			Code:    500,
			Message: fmt.Sprintf("创建权限失败: %v", err),
		}, nil
	}

	return &v1.CreatePermissionResponse{
		Code:    200,
		Message: "创建成功",
		Data:    toProtoPermissionWithTimestamps(result),
	}, nil
}

// UpdatePermission 更新权限
func (s *PermissionService) UpdatePermission(ctx context.Context, req *v1.UpdatePermissionRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	// 获取现有权限
	permission, err := s.permissionUsecase.GetPermissionByID(ctx, req.Id)
	if err != nil {
		return &v1.CommonResponse{
			Code:    404,
			Message: "权限不存在",
		}, nil
	}

	// 更新可选字段
	if req.Name != nil {
		permission.Name = *req.Name
	}
	if req.Description != nil {
		permission.Description = *req.Description
	}

	if err := s.permissionUsecase.UpdatePermission(ctx, permission); err != nil {
		s.log.Errorf("更新权限失败: %v", err)
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("更新权限失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "更新成功",
	}, nil
}

// DeletePermission 删除权限
func (s *PermissionService) DeletePermission(ctx context.Context, req *v1.DeletePermissionRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if err := s.permissionUsecase.DeletePermission(ctx, req.Id); err != nil {
		s.log.Errorf("删除权限失败: %v", err)
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("删除权限失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "删除成功",
	}, nil
}

// GetPermission 获取权限详情
func (s *PermissionService) GetPermission(ctx context.Context, req *v1.GetPermissionRequest) (*v1.GetPermissionResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetPermissionResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	permission, err := s.permissionUsecase.GetPermissionByID(ctx, req.Id)
	if err != nil {
		s.log.Errorf("获取权限失败: %v", err)
		return &v1.GetPermissionResponse{
			Code:    404,
			Message: "权限不存在",
		}, nil
	}

	return &v1.GetPermissionResponse{
		Code:    200,
		Message: "查询成功",
		Data:    toProtoPermissionWithTimestamps(permission),
	}, nil
}

// ListPermissions 获取权限列表
func (s *PermissionService) ListPermissions(ctx context.Context, req *v1.ListPermissionsRequest) (*v1.ListPermissionsResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ListPermissionsResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	page := 1
	pageSize := 20
	if req.PageRequest != nil {
		page = int(req.PageRequest.Page)
		pageSize = int(req.PageRequest.PageSize)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	permissions, total, err := s.permissionUsecase.ListPermissions(ctx, page, pageSize)
	if err != nil {
		s.log.Errorf("列出权限失败: %v", err)
		return &v1.ListPermissionsResponse{
			Code:    500,
			Message: fmt.Sprintf("列出权限失败: %v", err),
		}, nil
	}

	var protoPermissions []*v1.Permission
	for _, perm := range permissions {
		protoPermissions = append(protoPermissions, toProtoPermissionWithTimestamps(perm))
	}

	return &v1.ListPermissionsResponse{
		Code:    200,
		Message: "查询成功",
		List:    protoPermissions,
		PageResponse: &v1.PageResponse{
			Total:    int64(total),
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
	}, nil
}

// CreateRole 创建角色
func (s *PermissionService) CreateRole(ctx context.Context, req *v1.CreateRoleRequest) (*v1.CreateRoleResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CreateRoleResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	role := &biz.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
	}

	result, err := s.permissionUsecase.CreateRole(ctx, role)
	if err != nil {
		s.log.Errorf("创建角色失败: %v", err)
		return &v1.CreateRoleResponse{
			Code:    500,
			Message: fmt.Sprintf("创建角色失败: %v", err),
		}, nil
	}

	return &v1.CreateRoleResponse{
		Code:    200,
		Message: "创建成功",
		Data:    toProtoRoleWithTimestamps(result),
	}, nil
}

// UpdateRole 更新角色
func (s *PermissionService) UpdateRole(ctx context.Context, req *v1.UpdateRoleRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	// 获取现有角色
	role, err := s.permissionUsecase.GetRoleByID(ctx, req.Id)
	if err != nil {
		return &v1.CommonResponse{
			Code:    404,
			Message: "角色不存在",
		}, nil
	}

	// 更新可选字段
	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}

	if err := s.permissionUsecase.UpdateRole(ctx, role); err != nil {
		s.log.Errorf("更新角色失败: %v", err)
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("更新角色失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "更新成功",
	}, nil
}

// DeleteRole 删除角色
func (s *PermissionService) DeleteRole(ctx context.Context, req *v1.DeleteRoleRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if err := s.permissionUsecase.DeleteRole(ctx, req.Id); err != nil {
		s.log.Errorf("删除角色失败: %v", err)
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("删除角色失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "删除成功",
	}, nil
}

// GetRole 获取角色详情
func (s *PermissionService) GetRole(ctx context.Context, req *v1.GetRoleRequest) (*v1.GetRoleResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetRoleResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	role, err := s.permissionUsecase.GetRoleByID(ctx, req.Id)
	if err != nil {
		s.log.Errorf("获取角色失败: %v", err)
		return &v1.GetRoleResponse{
			Code:    404,
			Message: "角色不存在",
		}, nil
	}

	return &v1.GetRoleResponse{
		Code:    200,
		Message: "查询成功",
		Data:    toProtoRoleWithTimestamps(role),
	}, nil
}

// ListRoles 获取角色列表
func (s *PermissionService) ListRoles(ctx context.Context, req *v1.ListRolesRequest) (*v1.ListRolesResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ListRolesResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	page := 1
	pageSize := 20
	if req.PageRequest != nil {
		page = int(req.PageRequest.Page)
		pageSize = int(req.PageRequest.PageSize)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	roles, total, err := s.permissionUsecase.ListRoles(ctx, page, pageSize)
	if err != nil {
		s.log.Errorf("列出角色失败: %v", err)
		return &v1.ListRolesResponse{
			Code:    500,
			Message: fmt.Sprintf("列出角色失败: %v", err),
		}, nil
	}

	var protoRoles []*v1.Role
	for _, role := range roles {
		protoRoles = append(protoRoles, toProtoRoleWithTimestamps(role))
	}

	return &v1.ListRolesResponse{
		Code:    200,
		Message: "查询成功",
		List:    protoRoles,
		PageResponse: &v1.PageResponse{
			Total:    int64(total),
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
	}, nil
}

// GetRolePermissions 获取角色权限列表
func (s *PermissionService) GetRolePermissions(ctx context.Context, req *v1.GetRolePermissionsRequest) (*v1.GetRolePermissionsResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetRolePermissionsResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	permissions, err := s.permissionUsecase.GetRolePermissions(ctx, req.Id)
	if err != nil {
		s.log.Errorf("获取角色权限失败: %v", err)
		return &v1.GetRolePermissionsResponse{
			Code:    500,
			Message: fmt.Sprintf("获取角色权限失败: %v", err),
		}, nil
	}

	var protoPermissions []*v1.Permission
	for _, perm := range permissions {
		protoPermissions = append(protoPermissions, toProtoPermissionWithTimestamps(perm))
	}

	return &v1.GetRolePermissionsResponse{
		Code:    200,
		Message: "查询成功",
		List:    protoPermissions,
	}, nil
}

// AssignPermissionsToRole 分配权限给角色
func (s *PermissionService) AssignPermissionsToRole(ctx context.Context, req *v1.AssignPermissionsToRoleRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if err := s.permissionUsecase.AssignPermissionsToRole(ctx, req.Id, req.PermissionIds); err != nil {
		s.log.Errorf("分配权限失败: %v", err)
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("分配权限失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "分配成功",
	}, nil
}

// RemovePermissionsFromRole 从角色移除权限
func (s *PermissionService) RemovePermissionsFromRole(ctx context.Context, req *v1.RemovePermissionsFromRoleRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if err := s.permissionUsecase.RemovePermissionsFromRole(ctx, req.Id, req.PermissionIds); err != nil {
		s.log.Errorf("移除权限失败: %v", err)
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("移除权限失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "移除成功",
	}, nil
}

// AssignRolesToUser 分配角色给用户
func (s *PermissionService) AssignRolesToUser(ctx context.Context, req *v1.AssignRolesToUserRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if err := s.permissionUsecase.AssignRolesToUser(ctx, req.Id, req.RoleIds); err != nil {
		s.log.Errorf("分配角色失败: %v", err)
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("分配角色失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "分配成功",
	}, nil
}

// RemoveRolesFromUser 从用户移除角色
func (s *PermissionService) RemoveRolesFromUser(ctx context.Context, req *v1.RemoveRolesFromUserRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	if err := s.permissionUsecase.RemoveRolesFromUser(ctx, req.Id, req.RoleIds); err != nil {
		s.log.Errorf("移除角色失败: %v", err)
		return &v1.CommonResponse{
			Code:    500,
			Message: fmt.Sprintf("移除角色失败: %v", err),
		}, nil
	}

	return &v1.CommonResponse{
		Code:    200,
		Message: "移除成功",
	}, nil
}

// GetUserRoles 获取用户角色列表
func (s *PermissionService) GetUserRoles(ctx context.Context, req *v1.GetUserRolesRequest) (*v1.GetUserRolesResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetUserRolesResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	roles, err := s.permissionUsecase.GetUserRoles(ctx, req.Id)
	if err != nil {
		s.log.Errorf("获取用户角色失败: %v", err)
		return &v1.GetUserRolesResponse{
			Code:    500,
			Message: fmt.Sprintf("获取用户角色失败: %v", err),
		}, nil
	}

	var protoRoles []*v1.Role
	for _, role := range roles {
		protoRoles = append(protoRoles, toProtoRoleWithTimestamps(role))
	}

	return &v1.GetUserRolesResponse{
		Code:    200,
		Message: "查询成功",
		List:    protoRoles,
	}, nil
}

// GetUserPermissions 获取用户权限列表
func (s *PermissionService) GetUserPermissions(ctx context.Context, req *v1.GetUserPermissionsRequest) (*v1.GetUserPermissionsResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetUserPermissionsResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	_ = claims

	permissions, err := s.permissionUsecase.GetUserPermissions(ctx, req.Id)
	if err != nil {
		s.log.Errorf("获取用户权限失败: %v", err)
		return &v1.GetUserPermissionsResponse{
			Code:    500,
			Message: fmt.Sprintf("获取用户权限失败: %v", err),
		}, nil
	}

	var protoPermissions []*v1.Permission
	for _, perm := range permissions {
		protoPermissions = append(protoPermissions, toProtoPermissionWithTimestamps(perm))
	}

	return &v1.GetUserPermissionsResponse{
		Code:    200,
		Message: "查询成功",
		List:    protoPermissions,
	}, nil
}

func (s *PermissionService) rbacOK(ctx context.Context, userID uint64, resource, action string) bool {
	ok, err := s.permissionUsecase.CheckPermission(ctx, userID, resource, action)
	return err == nil && ok
}

// GetUserInstancePermissions 用户直授的实例权限（不含角色合并结果）
func (s *PermissionService) GetUserInstancePermissions(ctx context.Context, req *v1.GetUserInstancePermissionsRequest) (*v1.GetUserInstancePermissionsResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetUserInstancePermissionsResponse{Code: 401, Message: "未认证"}, nil
	}
	if claims.UserID != req.Id && !s.rbacOK(ctx, claims.UserID, "user", "manage") {
		return &v1.GetUserInstancePermissionsResponse{Code: 403, Message: "无权限查看该用户的实例授权"}, nil
	}
	rows, err := s.permissionUsecase.ListUserDirectInstancePermissionRows(ctx, req.Id)
	if err != nil {
		return &v1.GetUserInstancePermissionsResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}
	list := make([]*v1.InstancePermissionItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &v1.InstancePermissionItem{
			InstanceId:       row.InstanceID,
			PermissionType: row.PermissionType,
		})
	}
	return &v1.GetUserInstancePermissionsResponse{Code: 200, Message: "查询成功", List: list}, nil
}

// ReplaceUserInstancePermissions 全量替换用户直授实例权限
func (s *PermissionService) ReplaceUserInstancePermissions(ctx context.Context, req *v1.ReplaceUserInstancePermissionsRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{Code: 401, Message: "未认证"}, nil
	}
	if !s.rbacOK(ctx, claims.UserID, "user", "manage") {
		return &v1.CommonResponse{Code: 403, Message: "需要用户管理权限"}, nil
	}
	bizRows := make([]biz.InstancePermissionRow, 0, len(req.Items))
	for _, it := range req.Items {
		bizRows = append(bizRows, biz.InstancePermissionRow{
			InstanceID:       it.InstanceId,
			PermissionType: it.PermissionType,
		})
	}
	if err := s.permissionUsecase.ReplaceUserDirectInstancePermissions(ctx, req.Id, bizRows); err != nil {
		return &v1.CommonResponse{Code: 400, Message: err.Error()}, nil
	}
	return &v1.CommonResponse{Code: 200, Message: "保存成功"}, nil
}

// GetRoleInstancePermissions 角色实例权限
func (s *PermissionService) GetRoleInstancePermissions(ctx context.Context, req *v1.GetRoleInstancePermissionsRequest) (*v1.GetRoleInstancePermissionsResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.GetRoleInstancePermissionsResponse{Code: 401, Message: "未认证"}, nil
	}
	if !s.rbacOK(ctx, claims.UserID, "role", "manage") {
		return &v1.GetRoleInstancePermissionsResponse{Code: 403, Message: "需要角色管理权限"}, nil
	}
	rows, err := s.permissionUsecase.ListRoleInstancePermissionRows(ctx, req.Id)
	if err != nil {
		return &v1.GetRoleInstancePermissionsResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}
	list := make([]*v1.InstancePermissionItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &v1.InstancePermissionItem{
			InstanceId:       row.InstanceID,
			PermissionType: row.PermissionType,
		})
	}
	return &v1.GetRoleInstancePermissionsResponse{Code: 200, Message: "查询成功", List: list}, nil
}

// ReplaceRoleInstancePermissions 全量替换角色实例权限
func (s *PermissionService) ReplaceRoleInstancePermissions(ctx context.Context, req *v1.ReplaceRoleInstancePermissionsRequest) (*v1.CommonResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.CommonResponse{Code: 401, Message: "未认证"}, nil
	}
	if !s.rbacOK(ctx, claims.UserID, "role", "manage") {
		return &v1.CommonResponse{Code: 403, Message: "需要角色管理权限"}, nil
	}
	bizRows := make([]biz.InstancePermissionRow, 0, len(req.Items))
	for _, it := range req.Items {
		bizRows = append(bizRows, biz.InstancePermissionRow{
			InstanceID:       it.InstanceId,
			PermissionType: it.PermissionType,
		})
	}
	if err := s.permissionUsecase.ReplaceRoleInstancePermissions(ctx, req.Id, bizRows); err != nil {
		return &v1.CommonResponse{Code: 400, Message: err.Error()}, nil
	}
	return &v1.CommonResponse{Code: 200, Message: "保存成功"}, nil
}
