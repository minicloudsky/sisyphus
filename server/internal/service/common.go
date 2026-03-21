package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/transport"

	"sisyphus/server/internal/biz"
	"sisyphus/server/internal/middleware"
)

var (
	ErrUnauthorized = errors.New("未认证")
)

// getUserIDFromContext 从上下文中获取用户ID
func getUserIDFromContext(ctx context.Context) uint64 {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return 0
	}
	return claims.UserID
}

// getClaimsFromContext 从上下文中获取用户信息（支持手动提取token验证）
func getClaimsFromContext(ctx context.Context, authUsecase *biz.AuthUsecase) (*biz.Claims, error) {
	// 首先尝试从context中获取（如果认证中间件已启用）
	if claims, ok := middleware.FromContext(ctx); ok {
		return claims, nil
	}

	// 如果context中没有，尝试从HTTP transport中手动提取token
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}

	// 从Header获取token
	token := ""
	authHeader := tr.RequestHeader().Get("Authorization")
	if authHeader == "" {
		// 尝试其他可能的header名称
		authHeader = tr.RequestHeader().Get("authorization")
	}
	
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// 也支持没有 "Bearer " 前缀的情况
			token = authHeader
		}
	}

	if token == "" {
		// 添加调试日志
		fmt.Printf("[getClaimsFromContext] DEBUG: No token found in Authorization header\n")
		fmt.Printf("[getClaimsFromContext] DEBUG: All headers: %v\n", tr.RequestHeader())
		return nil, ErrUnauthorized
	}

	// 验证token
	claims, err := authUsecase.ValidateToken(token)
	if err != nil {
		// 添加调试日志
		fmt.Printf("[getClaimsFromContext] DEBUG: Token validation failed: %v\n", err)
		return nil, ErrUnauthorized
	}

	return claims, nil
}
