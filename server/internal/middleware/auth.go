package middleware

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"sisyphus/server/internal/biz"
)

type authKey struct{}

// publicOperations 不需要认证的操作
var publicOperations = map[string]bool{
	"/api.v1.AuthService/Login":    true,
	"/api.v1.AuthService/Register": true,
}

// Auth 认证中间件（跳过公开路由）
func Auth(authUsecase *biz.AuthUsecase) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 公开路由直接放行
			if publicOperations[tr.Operation()] {
				return handler(ctx, req)
			}

			// 健康检查等无认证 HTTP 路径（K8s / Docker / CI 探活）
			if tr.Kind() == transport.KindHTTP {
				if ht, ok := tr.(khttp.Transporter); ok && ht.Request() != nil {
					p := strings.TrimSuffix(ht.Request().URL.Path, "/")
					if p == "" {
						p = "/"
					}
					if p == "/health" {
						return handler(ctx, req)
					}
				}
			}

			// 从Header获取token
			token := ""
			if header := tr.RequestHeader().Get("Authorization"); header != "" {
				if strings.HasPrefix(header, "Bearer ") {
					token = strings.TrimPrefix(header, "Bearer ")
				}
			}

			if token == "" {
				return nil, errors.Unauthorized("UNAUTHORIZED", "token not found")
			}

			// 验证token
			claims, err := authUsecase.ValidateToken(token)
			if err != nil {
				return nil, errors.Unauthorized("UNAUTHORIZED", "invalid token")
			}

			// 将用户信息存入context
			ctx = context.WithValue(ctx, authKey{}, claims)

			return handler(ctx, req)
		}
	}
}

// FromContext 从context获取用户信息
func FromContext(ctx context.Context) (*biz.Claims, bool) {
	claims, ok := ctx.Value(authKey{}).(*biz.Claims)
	return claims, ok
}
