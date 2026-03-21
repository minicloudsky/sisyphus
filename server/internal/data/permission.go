package data

import (
	"context"
	"sort"

	"gorm.io/gorm"

	"sisyphus/server/internal/biz"
)

type permissionRepo struct {
	data *Data
}

// NewPermissionRepo 创建权限仓储
func NewPermissionRepo(data *Data) biz.PermissionRepo {
	return &permissionRepo{data: data}
}

func (r *permissionRepo) GetUserPermissions(ctx context.Context, userID uint64) ([]*biz.Permission, error) {
	var permissions []Permission
	err := r.data.db.WithContext(ctx).
		Table("sisyphus_permissions").
		Joins("JOIN sisyphus_role_permissions ON sisyphus_permissions.id = sisyphus_role_permissions.permission_id").
		Joins("JOIN sisyphus_roles ON sisyphus_role_permissions.role_id = sisyphus_roles.id").
		Joins("JOIN sisyphus_user_roles ON sisyphus_roles.id = sisyphus_user_roles.role_id").
		Where("sisyphus_user_roles.user_id = ?", userID).
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	result := make([]*biz.Permission, len(permissions))
	for i, p := range permissions {
		result[i] = toBizPermission(&p)
	}
	return result, nil
}

func (r *permissionRepo) GetUserRoles(ctx context.Context, userID uint64) ([]*biz.Role, error) {
	var roles []Role
	err := r.data.db.WithContext(ctx).
		Table("sisyphus_roles").
		Joins("JOIN sisyphus_user_roles ON sisyphus_roles.id = sisyphus_user_roles.role_id").
		Where("sisyphus_user_roles.user_id = ?", userID).
		Find(&roles).Error

	if err != nil {
		return nil, err
	}

	result := make([]*biz.Role, len(roles))
	for i, role := range roles {
		result[i] = toBizRole(&role)
	}
	return result, nil
}

func (r *permissionRepo) CheckPermission(ctx context.Context, userID uint64, resource, action string) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).
		Table("sisyphus_permissions").
		Joins("JOIN sisyphus_role_permissions ON sisyphus_permissions.id = sisyphus_role_permissions.permission_id").
		Joins("JOIN sisyphus_roles ON sisyphus_role_permissions.role_id = sisyphus_roles.id").
		Joins("JOIN sisyphus_user_roles ON sisyphus_roles.id = sisyphus_user_roles.role_id").
		Where("sisyphus_user_roles.user_id = ? AND sisyphus_permissions.resource = ? AND sisyphus_permissions.action = ?", userID, resource, action).
		Count(&count).Error

	return count > 0, err
}

func (r *permissionRepo) UserHasInstanceScope(ctx context.Context, userID uint64) (bool, error) {
	var n int64
	if err := r.data.db.WithContext(ctx).Model(&UserInstancePermission{}).
		Where("user_id = ?", userID).Limit(1).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	var roleIDs []uint64
	if err := r.data.db.WithContext(ctx).Model(&UserRole{}).
		Where("user_id = ?", userID).Pluck("role_id", &roleIDs).Error; err != nil {
		return false, err
	}
	if len(roleIDs) == 0 {
		return false, nil
	}
	var n2 int64
	if err := r.data.db.WithContext(ctx).Model(&RoleInstancePermission{}).
		Where("role_id IN ?", roleIDs).Limit(1).Count(&n2).Error; err != nil {
		return false, err
	}
	return n2 > 0, nil
}

func (r *permissionRepo) GetEffectiveInstancePermissionTypes(ctx context.Context, userID, instanceID uint64) ([]string, error) {
	seen := map[string]struct{}{}
	var fromUser []string
	if err := r.data.db.WithContext(ctx).Model(&UserInstancePermission{}).
		Where("user_id = ? AND instance_id = ?", userID, instanceID).
		Pluck("permission_type", &fromUser).Error; err != nil {
		return nil, err
	}
	for _, t := range fromUser {
		seen[t] = struct{}{}
	}
	var roleIDs []uint64
	if err := r.data.db.WithContext(ctx).Model(&UserRole{}).
		Where("user_id = ?", userID).Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) > 0 {
		var fromRole []string
		q := r.data.db.WithContext(ctx).Model(&RoleInstancePermission{}).
			Where("role_id IN ? AND instance_id = ?", roleIDs, instanceID)
		if err := q.Pluck("permission_type", &fromRole).Error; err != nil {
			return nil, err
		}
		for _, t := range fromRole {
			seen[t] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, nil
}

func (r *permissionRepo) ListAllowedInstanceIDs(ctx context.Context, userID uint64, permType *string) ([]uint64, error) {
	seen := map[uint64]struct{}{}
	qUser := r.data.db.WithContext(ctx).Model(&UserInstancePermission{}).Where("user_id = ?", userID)
	if permType != nil && *permType != "" {
		qUser = qUser.Where("permission_type = ?", *permType)
	}
	var uids []uint64
	if err := qUser.Distinct("instance_id").Pluck("instance_id", &uids).Error; err != nil {
		return nil, err
	}
	for _, id := range uids {
		seen[id] = struct{}{}
	}
	var roleIDs []uint64
	if err := r.data.db.WithContext(ctx).Model(&UserRole{}).
		Where("user_id = ?", userID).Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) > 0 {
		qRole := r.data.db.WithContext(ctx).Model(&RoleInstancePermission{}).Where("role_id IN ?", roleIDs)
		if permType != nil && *permType != "" {
			qRole = qRole.Where("permission_type = ?", *permType)
		}
		var rids []uint64
		if err := qRole.Distinct("instance_id").Pluck("instance_id", &rids).Error; err != nil {
			return nil, err
		}
		for _, id := range rids {
			seen[id] = struct{}{}
		}
	}
	out := make([]uint64, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func (r *permissionRepo) ListUserInstancePermissionRows(ctx context.Context, userID uint64) ([]biz.InstancePermissionRow, error) {
	var rows []UserInstancePermission
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]biz.InstancePermissionRow, len(rows))
	for i, row := range rows {
		out[i] = biz.InstancePermissionRow{InstanceID: row.InstanceID, PermissionType: row.PermissionType}
	}
	return out, nil
}

func (r *permissionRepo) ReplaceUserInstancePermissions(ctx context.Context, userID uint64, rows []biz.InstancePermissionRow) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&UserInstancePermission{}).Error; err != nil {
			return err
		}
		for _, row := range rows {
			rec := UserInstancePermission{UserID: userID, InstanceID: row.InstanceID, PermissionType: row.PermissionType}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *permissionRepo) ListRoleInstancePermissionRows(ctx context.Context, roleID uint64) ([]biz.InstancePermissionRow, error) {
	var rows []RoleInstancePermission
	if err := r.data.db.WithContext(ctx).Where("role_id = ?", roleID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]biz.InstancePermissionRow, len(rows))
	for i, row := range rows {
		out[i] = biz.InstancePermissionRow{InstanceID: row.InstanceID, PermissionType: row.PermissionType}
	}
	return out, nil
}

func (r *permissionRepo) ReplaceRoleInstancePermissions(ctx context.Context, roleID uint64, rows []biz.InstancePermissionRow) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&RoleInstancePermission{}).Error; err != nil {
			return err
		}
		for _, row := range rows {
			rec := RoleInstancePermission{RoleID: roleID, InstanceID: row.InstanceID, PermissionType: row.PermissionType}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Permission 数据库模型
type Permission struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	Name        string `gorm:"column:name;type:varchar(100);not null"`
	Code        string `gorm:"column:code;type:varchar(100);uniqueIndex;not null"`
	Resource    string `gorm:"column:resource;type:varchar(100);not null;index"`
	Action      string `gorm:"column:action;type:varchar(50);not null;index"`
	Description string `gorm:"column:description;type:varchar(255);default:''"`
}

func (Permission) TableName() string {
	return "sisyphus_permissions"
}

// Role 数据库模型
type Role struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	Name        string `gorm:"column:name;type:varchar(50);uniqueIndex;not null"`
	Code        string `gorm:"column:code;type:varchar(50);uniqueIndex;not null"`
	Description string `gorm:"column:description;type:varchar(255);default:''"`
}

func (Role) TableName() string {
	return "sisyphus_roles"
}

func toBizPermission(p *Permission) *biz.Permission {
	return &biz.Permission{
		ID:          p.ID,
		Name:        p.Name,
		Code:        p.Code,
		Resource:    p.Resource,
		Action:      p.Action,
		Description: p.Description,
	}
}

func toBizRole(r *Role) *biz.Role {
	return &biz.Role{
		ID:          r.ID,
		Name:        r.Name,
		Code:        r.Code,
		Description: r.Description,
	}
}

// CreatePermission 创建权限
func (r *permissionRepo) CreatePermission(ctx context.Context, permission *biz.Permission) (*biz.Permission, error) {
	p := &Permission{
		Name:        permission.Name,
		Code:        permission.Code,
		Resource:    permission.Resource,
		Action:      permission.Action,
		Description: permission.Description,
	}
	if err := r.data.db.WithContext(ctx).Create(p).Error; err != nil {
		return nil, err
	}
	return toBizPermission(p), nil
}

// UpdatePermission 更新权限
func (r *permissionRepo) UpdatePermission(ctx context.Context, permission *biz.Permission) error {
	return r.data.db.WithContext(ctx).Model(&Permission{}).
		Where("id = ?", permission.ID).
		Updates(map[string]interface{}{
			"name":        permission.Name,
			"code":        permission.Code,
			"resource":    permission.Resource,
			"action":      permission.Action,
			"description": permission.Description,
		}).Error
}

// DeletePermission 删除权限
func (r *permissionRepo) DeletePermission(ctx context.Context, id uint64) error {
	return r.data.db.WithContext(ctx).Delete(&Permission{}, id).Error
}

// GetPermissionByID 根据ID获取权限
func (r *permissionRepo) GetPermissionByID(ctx context.Context, id uint64) (*biz.Permission, error) {
	var p Permission
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return toBizPermission(&p), nil
}

// GetPermissionByCode 根据Code获取权限
func (r *permissionRepo) GetPermissionByCode(ctx context.Context, code string) (*biz.Permission, error) {
	var p Permission
	if err := r.data.db.WithContext(ctx).Where("code = ?", code).First(&p).Error; err != nil {
		return nil, err
	}
	return toBizPermission(&p), nil
}

// ListPermissions 列出权限
func (r *permissionRepo) ListPermissions(ctx context.Context, page, pageSize int) ([]*biz.Permission, int64, error) {
	var permissions []Permission
	var total int64

	offset := (page - 1) * pageSize
	query := r.data.db.WithContext(ctx).Model(&Permission{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(pageSize).Find(&permissions).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.Permission, len(permissions))
	for i, p := range permissions {
		result[i] = toBizPermission(&p)
	}
	return result, total, nil
}

// CreateRole 创建角色
func (r *permissionRepo) CreateRole(ctx context.Context, role *biz.Role) (*biz.Role, error) {
	ro := &Role{
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
	}
	if err := r.data.db.WithContext(ctx).Create(ro).Error; err != nil {
		return nil, err
	}
	return toBizRole(ro), nil
}

// UpdateRole 更新角色
func (r *permissionRepo) UpdateRole(ctx context.Context, role *biz.Role) error {
	return r.data.db.WithContext(ctx).Model(&Role{}).
		Where("id = ?", role.ID).
		Updates(map[string]interface{}{
			"name":        role.Name,
			"code":        role.Code,
			"description": role.Description,
		}).Error
}

// DeleteRole 删除角色
func (r *permissionRepo) DeleteRole(ctx context.Context, id uint64) error {
	return r.data.db.WithContext(ctx).Delete(&Role{}, id).Error
}

// GetRoleByID 根据ID获取角色
func (r *permissionRepo) GetRoleByID(ctx context.Context, id uint64) (*biz.Role, error) {
	var ro Role
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&ro).Error; err != nil {
		return nil, err
	}
	return toBizRole(&ro), nil
}

// GetRoleByCode 根据Code获取角色
func (r *permissionRepo) GetRoleByCode(ctx context.Context, code string) (*biz.Role, error) {
	var ro Role
	if err := r.data.db.WithContext(ctx).Where("code = ?", code).First(&ro).Error; err != nil {
		return nil, err
	}
	return toBizRole(&ro), nil
}

// ListRoles 列出角色
func (r *permissionRepo) ListRoles(ctx context.Context, page, pageSize int) ([]*biz.Role, int64, error) {
	var roles []Role
	var total int64

	offset := (page - 1) * pageSize
	query := r.data.db.WithContext(ctx).Model(&Role{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(pageSize).Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.Role, len(roles))
	for i, ro := range roles {
		result[i] = toBizRole(&ro)
	}
	return result, total, nil
}

// RolePermission 角色权限关联表
type RolePermission struct {
	RoleID       uint64 `gorm:"column:role_id;primaryKey;index"`
	PermissionID uint64 `gorm:"column:permission_id;primaryKey;index"`
}

func (RolePermission) TableName() string {
	return "sisyphus_role_permissions"
}

// AssignPermissionsToRole 为角色分配权限
func (r *permissionRepo) AssignPermissionsToRole(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	var rps []RolePermission
	for _, pid := range permissionIDs {
		rps = append(rps, RolePermission{
			RoleID:       roleID,
			PermissionID: pid,
		})
	}
	return r.data.db.WithContext(ctx).Create(&rps).Error
}

// RemovePermissionsFromRole 从角色移除权限
func (r *permissionRepo) RemovePermissionsFromRole(ctx context.Context, roleID uint64, permissionIDs []uint64) error {
	return r.data.db.WithContext(ctx).
		Where("role_id = ? AND permission_id IN ?", roleID, permissionIDs).
		Delete(&RolePermission{}).Error
}

// GetRolePermissions 获取角色的权限列表
func (r *permissionRepo) GetRolePermissions(ctx context.Context, roleID uint64) ([]*biz.Permission, error) {
	var permissions []Permission
	err := r.data.db.WithContext(ctx).
		Table("sisyphus_permissions").
		Joins("JOIN sisyphus_role_permissions ON sisyphus_permissions.id = sisyphus_role_permissions.permission_id").
		Where("sisyphus_role_permissions.role_id = ?", roleID).
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	result := make([]*biz.Permission, len(permissions))
	for i, p := range permissions {
		result[i] = toBizPermission(&p)
	}
	return result, nil
}

// UserRole 用户角色关联表
type UserRole struct {
	UserID uint64 `gorm:"column:user_id;primaryKey;index"`
	RoleID uint64 `gorm:"column:role_id;primaryKey;index"`
}

func (UserRole) TableName() string {
	return "sisyphus_user_roles"
}

// AssignRolesToUser 为用户分配角色
func (r *permissionRepo) AssignRolesToUser(ctx context.Context, userID uint64, roleIDs []uint64) error {
	if len(roleIDs) == 0 {
		return nil
	}

	var urs []UserRole
	for _, rid := range roleIDs {
		urs = append(urs, UserRole{
			UserID: userID,
			RoleID: rid,
		})
	}
	return r.data.db.WithContext(ctx).Create(&urs).Error
}

// RemoveRolesFromUser 从用户移除角色
func (r *permissionRepo) RemoveRolesFromUser(ctx context.Context, userID uint64, roleIDs []uint64) error {
	return r.data.db.WithContext(ctx).
		Where("user_id = ? AND role_id IN ?", userID, roleIDs).
		Delete(&UserRole{}).Error
}

// UserInstancePermission 用户实例授权（与角色授权合并）
type UserInstancePermission struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID         uint64 `gorm:"column:user_id;not null;index"`
	InstanceID     uint64 `gorm:"column:instance_id;not null;index"`
	PermissionType string `gorm:"column:permission_type;type:varchar(50);not null"`
}

func (UserInstancePermission) TableName() string {
	return "sisyphus_user_instance_permissions"
}

// RoleInstancePermission 角色实例授权
type RoleInstancePermission struct {
	ID             uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	RoleID         uint64 `gorm:"column:role_id;not null;index"`
	InstanceID     uint64 `gorm:"column:instance_id;not null;index"`
	PermissionType string `gorm:"column:permission_type;type:varchar(50);not null"`
}

func (RoleInstancePermission) TableName() string {
	return "sisyphus_role_instance_permissions"
}
