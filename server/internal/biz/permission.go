package biz

import (
	"context"
	"errors"
)

// 实例级能力（与 RBAC 的 sql_query / sql_order 等配合）：无任意实例授权记录时不对实例做额外限制（兼容旧库）
const (
	InstancePermQuery  = "query"
	InstancePermDDL    = "ddl"
	InstancePermDML    = "dml"
	InstancePermExport = "export"
)

var validInstancePermissionTypes = map[string]bool{
	InstancePermQuery: true, InstancePermDDL: true, InstancePermDML: true, InstancePermExport: true,
}

// ValidInstancePermissionType 是否为受支持的实例权限类型
func ValidInstancePermissionType(t string) bool {
	return validInstancePermissionTypes[t]
}

// InstancePermForSQLOrderType 工单类型对应的实例权限
func InstancePermForSQLOrderType(orderType int) string {
	switch orderType {
	case OrderTypeDDL:
		return InstancePermDDL
	case OrderTypeDML:
		return InstancePermDML
	case OrderTypeEXPORT:
		return InstancePermExport
	default:
		return InstancePermDML
	}
}

// InstancePermissionRow 用户或角色在某一实例上的一条能力
type InstancePermissionRow struct {
	InstanceID       uint64
	PermissionType string
}

// 内置角色 code，不允许删除
var builtinRoleCodes = map[string]bool{
	"admin":     true,
	"dba":       true,
	"developer": true,
	"readonly":  true,
}

// Permission 权限实体
type Permission struct {
	ID          uint64
	Name        string
	Code        string
	Resource    string
	Action      string
	Description string
}

// Role 角色实体
type Role struct {
	ID          uint64
	Name        string
	Code        string
	Description string
	Permissions []*Permission
}

// UserRole 用户角色关联
type UserRole struct {
	UserID uint64
	RoleID uint64
}

// PermissionRepo 权限仓储接口
type PermissionRepo interface {
	// 权限相关
	CreatePermission(ctx context.Context, permission *Permission) (*Permission, error)
	UpdatePermission(ctx context.Context, permission *Permission) error
	DeletePermission(ctx context.Context, id uint64) error
	GetPermissionByID(ctx context.Context, id uint64) (*Permission, error)
	GetPermissionByCode(ctx context.Context, code string) (*Permission, error)
	ListPermissions(ctx context.Context, page, pageSize int) ([]*Permission, int64, error)

	// 角色相关
	CreateRole(ctx context.Context, role *Role) (*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id uint64) error
	GetRoleByID(ctx context.Context, id uint64) (*Role, error)
	GetRoleByCode(ctx context.Context, code string) (*Role, error)
	ListRoles(ctx context.Context, page, pageSize int) ([]*Role, int64, error)

	// 角色权限关联
	AssignPermissionsToRole(ctx context.Context, roleID uint64, permissionIDs []uint64) error
	RemovePermissionsFromRole(ctx context.Context, roleID uint64, permissionIDs []uint64) error
	GetRolePermissions(ctx context.Context, roleID uint64) ([]*Permission, error)

	// 用户角色关联
	AssignRolesToUser(ctx context.Context, userID uint64, roleIDs []uint64) error
	RemoveRolesFromUser(ctx context.Context, userID uint64, roleIDs []uint64) error

	// 原有方法
	GetUserPermissions(ctx context.Context, userID uint64) ([]*Permission, error)
	GetUserRoles(ctx context.Context, userID uint64) ([]*Role, error)
	CheckPermission(ctx context.Context, userID uint64, resource, action string) (bool, error)
	UserHasInstanceScope(ctx context.Context, userID uint64) (bool, error)
	GetEffectiveInstancePermissionTypes(ctx context.Context, userID, instanceID uint64) ([]string, error)
	// ListAllowedInstanceIDs permType 非空时仅包含拥有该能力的实例
	ListAllowedInstanceIDs(ctx context.Context, userID uint64, permType *string) ([]uint64, error)
	ListUserInstancePermissionRows(ctx context.Context, userID uint64) ([]InstancePermissionRow, error)
	ReplaceUserInstancePermissions(ctx context.Context, userID uint64, rows []InstancePermissionRow) error
	ListRoleInstancePermissionRows(ctx context.Context, roleID uint64) ([]InstancePermissionRow, error)
	ReplaceRoleInstancePermissions(ctx context.Context, roleID uint64, rows []InstancePermissionRow) error
}

// PermissionUsecase 权限用例
type PermissionUsecase struct {
	repo PermissionRepo
}

// NewPermissionUsecase 创建权限用例
func NewPermissionUsecase(repo PermissionRepo) *PermissionUsecase {
	return &PermissionUsecase{repo: repo}
}

// CheckPermission 检查权限（admin 角色直通）
func (uc *PermissionUsecase) CheckPermission(ctx context.Context, userID uint64, resource, action string) (bool, error) {
	// admin 角色拥有所有权限，无需逐条检查
	if ok, _ := uc.IsAdmin(ctx, userID); ok {
		return true, nil
	}
	return uc.repo.CheckPermission(ctx, userID, resource, action)
}

// IsAdmin 判断用户是否拥有 admin 角色
func (uc *PermissionUsecase) IsAdmin(ctx context.Context, userID uint64) (bool, error) {
	roles, err := uc.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r.Code == "admin" {
			return true, nil
		}
	}
	return false, nil
}

// GetUserPermissions 获取用户权限
func (uc *PermissionUsecase) GetUserPermissions(ctx context.Context, userID uint64) ([]*Permission, error) {
	return uc.repo.GetUserPermissions(ctx, userID)
}

// GetUserRoles 获取用户角色
func (uc *PermissionUsecase) GetUserRoles(ctx context.Context, userID uint64) ([]*Role, error) {
	return uc.repo.GetUserRoles(ctx, userID)
}

// GetInstancePermissions 获取用户对实例的有效权限类型（角色 ∪ 用户）
func (uc *PermissionUsecase) GetInstancePermissions(ctx context.Context, userID, instanceID uint64) ([]string, error) {
	return uc.repo.GetEffectiveInstancePermissionTypes(ctx, userID, instanceID)
}

// CanViewInstanceMeta 是否可查看实例元数据（任一类实例授权即可；未启用实例 ACL 时均可）
func (uc *PermissionUsecase) CanViewInstanceMeta(ctx context.Context, userID, instanceID uint64) (bool, error) {
	ok, err := uc.repo.CheckPermission(ctx, userID, "instance", "manage")
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	scoped, err := uc.repo.UserHasInstanceScope(ctx, userID)
	if err != nil {
		return false, err
	}
	if !scoped {
		return true, nil
	}
	types, err := uc.repo.GetEffectiveInstancePermissionTypes(ctx, userID, instanceID)
	if err != nil {
		return false, err
	}
	return len(types) > 0, nil
}

// CanAccessInstance 实例级校验：instance:manage 直通；未配置任何实例授权时直通；否则要求具备 permType
func (uc *PermissionUsecase) CanAccessInstance(ctx context.Context, userID, instanceID uint64, permType string) (bool, error) {
	if !ValidInstancePermissionType(permType) {
		return false, errors.New("无效的实例权限类型")
	}
	ok, err := uc.repo.CheckPermission(ctx, userID, "instance", "manage")
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	scoped, err := uc.repo.UserHasInstanceScope(ctx, userID)
	if err != nil {
		return false, err
	}
	if !scoped {
		return true, nil
	}
	types, err := uc.repo.GetEffectiveInstancePermissionTypes(ctx, userID, instanceID)
	if err != nil {
		return false, err
	}
	for _, t := range types {
		if t == permType {
			return true, nil
		}
	}
	return false, nil
}

// ResolveInstanceListFilter 列表过滤：需限制时返回 allowed IDs；permScope 为空表示「任一类实例能力即可」
func (uc *PermissionUsecase) ResolveInstanceListFilter(ctx context.Context, userID uint64, permScope *string) (restrict bool, allowedIDs []uint64, err error) {
	if permScope != nil && *permScope != "" && !ValidInstancePermissionType(*permScope) {
		return false, nil, errors.New("无效的实例权限范围")
	}
	ok, err := uc.repo.CheckPermission(ctx, userID, "instance", "manage")
	if err != nil {
		return false, nil, err
	}
	if ok {
		return false, nil, nil
	}
	scoped, err := uc.repo.UserHasInstanceScope(ctx, userID)
	if err != nil {
		return false, nil, err
	}
	if !scoped {
		return false, nil, nil
	}
	var typeFilter *string
	if permScope != nil && *permScope != "" {
		typeFilter = permScope
	}
	ids, err := uc.repo.ListAllowedInstanceIDs(ctx, userID, typeFilter)
	if err != nil {
		return false, nil, err
	}
	return true, ids, nil
}

// ListUserDirectInstancePermissionRows 用户直授权（不含角色）
func (uc *PermissionUsecase) ListUserDirectInstancePermissionRows(ctx context.Context, userID uint64) ([]InstancePermissionRow, error) {
	return uc.repo.ListUserInstancePermissionRows(ctx, userID)
}

// ReplaceUserDirectInstancePermissions 全量替换用户直授权
func (uc *PermissionUsecase) ReplaceUserDirectInstancePermissions(ctx context.Context, userID uint64, rows []InstancePermissionRow) error {
	for _, row := range rows {
		if !ValidInstancePermissionType(row.PermissionType) {
			return errors.New("无效的实例权限类型")
		}
	}
	return uc.repo.ReplaceUserInstancePermissions(ctx, userID, rows)
}

// ListRoleInstancePermissionRows 角色实例授权
func (uc *PermissionUsecase) ListRoleInstancePermissionRows(ctx context.Context, roleID uint64) ([]InstancePermissionRow, error) {
	return uc.repo.ListRoleInstancePermissionRows(ctx, roleID)
}

// ReplaceRoleInstancePermissions 全量替换角色实例授权
func (uc *PermissionUsecase) ReplaceRoleInstancePermissions(ctx context.Context, roleID uint64, rows []InstancePermissionRow) error {
	for _, row := range rows {
		if !ValidInstancePermissionType(row.PermissionType) {
			return errors.New("无效的实例权限类型")
		}
	}
	return uc.repo.ReplaceRoleInstancePermissions(ctx, roleID, rows)
}

// CheckUserPermission 检查用户权限（用于权限中间件）
func (uc *PermissionUsecase) CheckUserPermission(ctx context.Context, userID uint64, resource, action string) (bool, error) {
	return uc.CheckPermission(ctx, userID, resource, action)
}

// CreatePermission 创建权限
func (uc *PermissionUsecase) CreatePermission(ctx context.Context, permission *Permission) (*Permission, error) {
	return uc.repo.CreatePermission(ctx, permission)
}

// UpdatePermission 更新权限
func (uc *PermissionUsecase) UpdatePermission(ctx context.Context, permission *Permission) error {
	return uc.repo.UpdatePermission(ctx, permission)
}

// DeletePermission 删除权限
func (uc *PermissionUsecase) DeletePermission(ctx context.Context, id uint64) error {
	return uc.repo.DeletePermission(ctx, id)
}

// GetPermissionByID 根据ID获取权限
func (uc *PermissionUsecase) GetPermissionByID(ctx context.Context, id uint64) (*Permission, error) {
	return uc.repo.GetPermissionByID(ctx, id)
}

// ListPermissions 列出权限
func (uc *PermissionUsecase) ListPermissions(ctx context.Context, page, pageSize int) ([]*Permission, int64, error) {
	return uc.repo.ListPermissions(ctx, page, pageSize)
}

// CreateRole 创建角色
func (uc *PermissionUsecase) CreateRole(ctx context.Context, role *Role) (*Role, error) {
	return uc.repo.CreateRole(ctx, role)
}

// UpdateRole 更新角色
func (uc *PermissionUsecase) UpdateRole(ctx context.Context, role *Role) error {
	return uc.repo.UpdateRole(ctx, role)
}

// DeleteRole 删除角色（内置角色不允许删除）
func (uc *PermissionUsecase) DeleteRole(ctx context.Context, id uint64) error {
	role, err := uc.repo.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if builtinRoleCodes[role.Code] {
		return errors.New("内置角色不可删除")
	}
	return uc.repo.DeleteRole(ctx, id)
}

// GetRoleByID 根据ID获取角色
func (uc *PermissionUsecase) GetRoleByID(ctx context.Context, id uint64) (*Role, error) {
	return uc.repo.GetRoleByID(ctx, id)
}

// ListRoles 列出角色
func (uc *PermissionUsecase) ListRoles(ctx context.Context, page, pageSize int) ([]*Role, int64, error) {
	return uc.repo.ListRoles(ctx, page, pageSize)
}

// AssignPermissionsToRole 为角色分配权限
func (uc *PermissionUsecase) AssignPermissionsToRole(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return uc.repo.AssignPermissionsToRole(ctx, roleID, permissionIDs)
}

// RemovePermissionsFromRole 从角色移除权限
func (uc *PermissionUsecase) RemovePermissionsFromRole(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return uc.repo.RemovePermissionsFromRole(ctx, roleID, permissionIDs)
}

// GetRolePermissions 获取角色的权限列表
func (uc *PermissionUsecase) GetRolePermissions(ctx context.Context, roleID uint64) ([]*Permission, error) {
	return uc.repo.GetRolePermissions(ctx, roleID)
}

// AssignRolesToUser 为用户分配角色
func (uc *PermissionUsecase) AssignRolesToUser(ctx context.Context, userID uint64, roleIDs []uint64) error {
	return uc.repo.AssignRolesToUser(ctx, userID, roleIDs)
}

// RemoveRolesFromUser 从用户移除角色
func (uc *PermissionUsecase) RemoveRolesFromUser(ctx context.Context, userID uint64, roleIDs []uint64) error {
	return uc.repo.RemoveRolesFromUser(ctx, userID, roleIDs)
}
