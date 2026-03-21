package middleware

import (
	"context"
	"strconv"

	"github.com/go-kratos/kratos/v2/transport/http"
	"sisyphus/server/internal/biz"
)

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	CheckUserPermission(ctx context.Context, userID uint64, resource string, action string) (bool, error)
	CheckInstancePermission(ctx context.Context, userID uint64, instanceID uint64, action string) (bool, error)
}

// Permission 权限检查中间件（用于HTTP处理器中手动调用）
func Permission(permissionChecker PermissionChecker, ctx http.Context) error {
	claims, ok := FromContext(ctx.Request().Context())
	if !ok {
		return ctx.JSON(401, map[string]interface{}{"code": 401, "message": "未认证"})
	}

	// 从请求路径中提取资源信息
	path := ctx.Request().URL.Path
	method := ctx.Request().Method

	// 检查实例级权限
	if instanceID := extractInstanceID(path); instanceID > 0 {
		action := methodToAction(method)
		allowed, err := permissionChecker.CheckInstancePermission(ctx.Request().Context(), claims.UserID, instanceID, action)
		if err != nil {
			return ctx.JSON(500, map[string]interface{}{"code": 500, "message": "权限检查失败"})
		}
		if !allowed {
			return ctx.JSON(403, map[string]interface{}{"code": 403, "message": "无权限访问该实例"})
		}
	}

	// 检查资源级权限
	resource := extractResource(path)
	if resource != "" {
		action := methodToAction(method)
		allowed, err := permissionChecker.CheckUserPermission(ctx.Request().Context(), claims.UserID, resource, action)
		if err != nil {
			return ctx.JSON(500, map[string]interface{}{"code": 500, "message": "权限检查失败"})
		}
		if !allowed {
			return ctx.JSON(403, map[string]interface{}{"code": 403, "message": "无权限执行该操作"})
		}
	}

	return nil
}

// extractInstanceID 从路径中提取实例ID
func extractInstanceID(path string) uint64 {
	// 匹配 /api/v1/instances/{id} 或 /api/v1/sql-orders?instance_id={id} 等
	// 简化实现，实际需要更复杂的路由解析
	if idx := findSubstring(path, "/instances/"); idx >= 0 {
		rest := path[idx+len("/instances/"):]
		for i := 0; i < len(rest) && i < 20; i++ {
			if rest[i] == '/' || rest[i] == '?' {
				if id, err := strconv.ParseUint(rest[:i], 10, 64); err == nil {
					return id
				}
				break
			}
		}
	}
	return 0
}

// extractResource 从路径中提取资源类型
func extractResource(path string) string {
	// 简化实现，从路径中提取资源类型
	if contains(path, "/sql-orders") {
		return "sql_order"
	}
	if contains(path, "/archive") {
		return "archive"
	}
	if contains(path, "/binlog") {
		return "binlog"
	}
	if contains(path, "/redis") {
		return "redis"
	}
	return ""
}

// methodToAction 将HTTP方法转换为操作动作
func methodToAction(method string) string {
	switch method {
	case "GET":
		return "read"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return "unknown"
	}
}

// findSubstring 查找子字符串位置
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return findSubstring(s, substr) >= 0
}

// PermissionCheckerImpl 权限检查器实现
type PermissionCheckerImpl struct {
	permissionUsecase *biz.PermissionUsecase
}

// NewPermissionChecker 创建权限检查器
func NewPermissionChecker(permissionUsecase *biz.PermissionUsecase) PermissionChecker {
	return &PermissionCheckerImpl{
		permissionUsecase: permissionUsecase,
	}
}

// CheckUserPermission 检查用户权限
func (p *PermissionCheckerImpl) CheckUserPermission(ctx context.Context, userID uint64, resource string, action string) (bool, error) {
	// 使用PermissionUsecase检查权限
	return p.permissionUsecase.CheckUserPermission(ctx, userID, resource, action)
}

// CheckInstancePermission 检查实例权限
func (p *PermissionCheckerImpl) CheckInstancePermission(ctx context.Context, userID uint64, instanceID uint64, action string) (bool, error) {
	// 检查用户是否有该实例的权限
	// 这里简化实现，实际需要查询用户-实例权限表
	// 暂时返回true，表示有权限（实际应该查询数据库）
	return true, nil
}
