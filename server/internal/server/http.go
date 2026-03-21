package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/crypto/bcrypt"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	"sisyphus/server/internal/conf"
	"sisyphus/server/internal/middleware"
	"sisyphus/server/internal/pkg/export"
	"sisyphus/server/internal/service"
)

// apiResponse 通用响应结构
type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, apiResponse{Code: 200, Message: "success", Data: data})
}

func writeError(w http.ResponseWriter, httpStatus int, code int, msg string) {
	writeJSON(w, httpStatus, apiResponse{Code: code, Message: msg})
}

// extractBearerToken 从 Authorization header 提取 Bearer token
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		auth = r.Header.Get("authorization")
	}
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return auth
}

// userItem 用户列表响应 DTO
type userItem struct {
	ID         uint64 `json:"id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	RealName   string `json:"real_name"`
	Phone      string `json:"phone"`
	Department string `json:"department"`
	Status     int    `json:"status"`
}

func bizUserToItem(u *biz.User) userItem {
	return userItem{
		ID:         u.ID,
		Username:   u.Username,
		Email:      u.Email,
		RealName:   u.RealName,
		Phone:      u.Phone,
		Department: u.Department,
		Status:     u.Status,
	}
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(
	c *conf.Server,
	authService *service.AuthService,
	sqlOrderService *service.SQLOrderService,
	queryService *service.QueryService,
	binlogService *service.BinlogService,
	redisService *service.RedisService,
	archiveService *service.ArchiveService,
	instanceService *service.InstanceService,
	ghostService *service.GhOstService,
	permissionService *service.PermissionService,
	favoriteService *service.FavoriteService,
	mybatisService *service.MyBatisService,
	widetableService *service.WidetableService,
	userUsecase *biz.UserUsecase,
	authUsecase *biz.AuthUsecase,
	partitionUsecase *biz.PartitionUsecase,
	archiveUsecase *biz.ArchiveUsecase,
	binlogUsecase *biz.BinlogRollbackUsecase,
	mybatisGenerator *biz.MyBatisGenerator,
	notificationService *biz.NotificationService,
	logger log.Logger,
) *kratoshttp.Server {
	var opts = []kratoshttp.ServerOption{
		kratoshttp.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			middleware.Auth(authUsecase),
		),
	}
	if c.HTTP.Network != "" {
		opts = append(opts, kratoshttp.Network(c.HTTP.Network))
	}
	if c.HTTP.Addr != "" {
		opts = append(opts, kratoshttp.Address(c.HTTP.Addr))
	}
	if c.HTTP.Timeout > 0 {
		opts = append(opts, kratoshttp.Timeout(c.HTTP.Timeout))
	}

	srv := kratoshttp.NewServer(opts...)

	// 注册 Protobuf 生成的路由
	registerHTTPServices(srv, authService, sqlOrderService, queryService, binlogService, redisService, archiveService, instanceService, ghostService, permissionService, favoriteService, mybatisService, widetableService)

	// 注册自定义路由
	registerCustomRoutes(srv, authUsecase, userUsecase, archiveUsecase, binlogUsecase, mybatisGenerator)

	// 注册分区管理路由
	registerPartitionRoutes(srv, authUsecase, partitionUsecase)

	// 注册通知配置路由
	registerNotificationRoutes(srv, authUsecase, notificationService)

	return srv
}

func registerHTTPServices(
	srv *kratoshttp.Server,
	authService *service.AuthService,
	sqlOrderService *service.SQLOrderService,
	queryService *service.QueryService,
	binlogService *service.BinlogService,
	redisService *service.RedisService,
	archiveService *service.ArchiveService,
	instanceService *service.InstanceService,
	ghostService *service.GhOstService,
	permissionService *service.PermissionService,
	favoriteService *service.FavoriteService,
	_ *service.MyBatisService,
	widetableService *service.WidetableService,
) {
	// 使用 Protobuf 生成的注册函数注册服务
	// 认证相关路由（不需要认证）
	v1.RegisterAuthServiceHTTPServer(srv, authService)

	// SQL工单相关路由（需要认证）
	v1.RegisterSQLOrderServiceHTTPServer(srv, sqlOrderService)

	// 数据查询服务（已迁移到 Protobuf）
	v1.RegisterQueryServiceHTTPServer(srv, queryService)

	// Binlog回滚服务（已迁移到 Protobuf）
	v1.RegisterBinlogServiceHTTPServer(srv, binlogService)

	// Redis分析服务（已迁移到 Protobuf）
	v1.RegisterRedisServiceHTTPServer(srv, redisService)

	// 数据归档服务（已迁移到 Protobuf）
	v1.RegisterArchiveServiceHTTPServer(srv, archiveService)

	// 实例管理服务（已迁移到 Protobuf）
	v1.RegisterInstanceServiceHTTPServer(srv, instanceService)

	// gh-ost管理服务（已迁移到 Protobuf）
	v1.RegisterGhOstServiceHTTPServer(srv, ghostService)

	// 权限管理服务（已迁移到 Protobuf）
	v1.RegisterPermissionServiceHTTPServer(srv, permissionService)

	// 收藏夹服务
	v1.RegisterFavoriteServiceHTTPServer(srv, favoriteService)


	// 宽表服务
	v1.RegisterWidetableServiceHTTPServer(srv, widetableService)
}

// registerCustomRoutes 注册自定义 HTTP 路由（不依赖 proto）
func registerCustomRoutes(srv *kratoshttp.Server, authUsecase *biz.AuthUsecase, userUsecase *biz.UserUsecase, archiveUsecase *biz.ArchiveUsecase, binlogUsecase *biz.BinlogRollbackUsecase, mybatisGenerator *biz.MyBatisGenerator) {
	r := srv.Route("/")

	// ----------------------------------------------------------------
	// GET /health — 负载均衡 / 容器探活（无需 JWT）
	// ----------------------------------------------------------------
	r.GET("/health", func(ctx kratoshttp.Context) error {
		w := ctx.Response()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return nil
	})

	// ----------------------------------------------------------------
	// POST /api/v1/mybatis/generate — 本地 MyBatis XML 生成
	// ----------------------------------------------------------------
	r.POST("/api/v1/mybatis/generate", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, 400, "读取请求体失败")
			return nil
		}
		var reqBody struct {
			DDL string `json:"ddl"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil || reqBody.DDL == "" {
			writeError(w, http.StatusBadRequest, 400, "缺少 ddl 字段")
			return nil
		}

		xml, err := mybatisGenerator.GenerateXML(reqBody.DDL)
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		writeSuccess(w, map[string]string{"xml": xml})
		return nil
	})

	// ----------------------------------------------------------------
	// POST /api/v1/auth/refresh — 刷新 token
	// ----------------------------------------------------------------
	r.POST("/api/v1/auth/refresh", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			// 也尝试从 request body 读取 token 字段
			body, _ := io.ReadAll(req.Body)
			var reqBody struct {
				Token string `json:"token"`
			}
			_ = json.Unmarshal(body, &reqBody)
			tokenStr = reqBody.Token
		}
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "缺少token")
			return nil
		}

		newToken, err := authUsecase.RefreshToken(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, 401, err.Error())
			return nil
		}

		writeSuccess(w, map[string]string{"token": newToken})
		return nil
	})

	// ----------------------------------------------------------------
	// GET /api/v1/permission/users — 用户列表（支持分页：page, page_size）
	// ----------------------------------------------------------------
	r.GET("/api/v1/permission/users", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		query := req.URL.Query()
		page, _ := strconv.Atoi(query.Get("page"))
		pageSize, _ := strconv.Atoi(query.Get("page_size"))
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		users, total, err := userUsecase.ListUsers(req.Context(), page, pageSize)
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		list := make([]userItem, 0, len(users))
		for _, u := range users {
			list = append(list, bizUserToItem(u))
		}
		writeSuccess(w, map[string]interface{}{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
		return nil
	})

	// ----------------------------------------------------------------
	// POST /api/v1/permission/users — 创建用户
	// ----------------------------------------------------------------
	r.POST("/api/v1/permission/users", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, 400, "读取请求体失败")
			return nil
		}
		var reqBody struct {
			Username   string `json:"username"`
			Password   string `json:"password"`
			Email      string `json:"email"`
			RealName   string `json:"real_name"`
			Phone      string `json:"phone"`
			Department string `json:"department"`
			Status     int    `json:"status"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			writeError(w, http.StatusBadRequest, 400, "请求参数格式错误")
			return nil
		}
		if reqBody.Username == "" || reqBody.Password == "" || reqBody.Email == "" {
			writeError(w, http.StatusBadRequest, 400, "用户名、密码、邮箱为必填项")
			return nil
		}

		// 加密密码
		hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(reqBody.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			writeError(w, http.StatusInternalServerError, 500, "密码加密失败")
			return nil
		}

		status := reqBody.Status
		if status == 0 {
			status = 1 // 默认启用
		}

		user := &biz.User{
			Username:   reqBody.Username,
			Password:   string(hashedPassword),
			Email:      reqBody.Email,
			RealName:   reqBody.RealName,
			Phone:      reqBody.Phone,
			Department: reqBody.Department,
			Status:     status,
		}

		created, createErr := userUsecase.CreateUser(req.Context(), user)
		if createErr != nil {
			if createErr == biz.ErrUserExists {
				writeError(w, http.StatusBadRequest, 400, "用户名已存在")
			} else if createErr == biz.ErrEmailExists {
				writeError(w, http.StatusBadRequest, 400, "邮箱已存在")
			} else {
				writeError(w, http.StatusInternalServerError, 500, createErr.Error())
			}
			return nil
		}

		writeSuccess(w, bizUserToItem(created))
		return nil
	})

	// ----------------------------------------------------------------
	// DELETE /api/v1/permission/users/{id} — 删除用户
	// ----------------------------------------------------------------
	r.DELETE("/api/v1/permission/users/{id}", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		idStr := ctx.Vars().Get("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, http.StatusBadRequest, 400, "无效的用户ID")
			return nil
		}

		if err := userUsecase.DeleteUser(req.Context(), id); err != nil {
			if err == biz.ErrUserNotFound {
				writeError(w, http.StatusNotFound, 404, "用户不存在")
			} else {
				writeError(w, http.StatusInternalServerError, 500, err.Error())
			}
			return nil
		}

		writeSuccess(w, nil)
		return nil
	})

	// ----------------------------------------------------------------
	// PUT /api/v1/permission/users/{id} — 更新用户（支持密码修改）
	// ----------------------------------------------------------------

	// ----------------------------------------------------------------
	// POST /api/v1/archive/configs/{id}/execute — 手动触发一次归档
	// ----------------------------------------------------------------
	r.POST("/api/v1/archive/configs/{id}/execute", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		idStr := ctx.Vars().Get("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, http.StatusBadRequest, 400, "无效的归档配置ID")
			return nil
		}

		archiveUsecase.TriggerArchiveNow(req.Context(), id)
		writeSuccess(w, map[string]string{"message": "归档任务已触发"})
		return nil
	})

	// ----------------------------------------------------------------
	// GET /api/v1/binlog/orders/{id}/sqls/download — 下载回滚SQL文件
	// ----------------------------------------------------------------
	r.GET("/api/v1/binlog/orders/{id}/sqls/download", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		idStr := ctx.Vars().Get("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, http.StatusBadRequest, 400, "无效的工单ID")
			return nil
		}

		sqls, err := binlogUsecase.GetRollbackSQLsByOrderID(req.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		var sb strings.Builder
		sb.WriteString("-- Sisyphus Binlog Rollback SQLs\n")
		sb.WriteString("-- Generated at: " + "now" + "\n\n")
		for _, s := range sqls {
			sb.WriteString(s.RollbackSQL)
			sb.WriteString(";\n")
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=\"rollback_"+idStr+".sql\"")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sb.String()))
		return nil
	})

	// ----------------------------------------------------------------
	// GET /api/v1/query/download/{filename} — 下载查询导出文件
	// ----------------------------------------------------------------
	r.GET("/api/v1/query/download/{filename}", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		name := ctx.Vars().Get("filename")
		if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
			writeError(w, http.StatusBadRequest, 400, "无效的文件名")
			return nil
		}
		name = filepath.Base(name)

		exportDir, err := filepath.Abs(export.GetExportDir())
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, "导出目录不可用")
			return nil
		}
		fullPath := filepath.Join(exportDir, name)
		fullPath, err = filepath.Abs(fullPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, "路径解析失败")
			return nil
		}
		rel, err := filepath.Rel(exportDir, fullPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			writeError(w, http.StatusForbidden, 403, "禁止访问该路径")
			return nil
		}

		data, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				writeError(w, http.StatusNotFound, 404, "文件不存在或已过期")
				return nil
			}
			writeError(w, http.StatusInternalServerError, 500, "读取文件失败")
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		contentType := "application/octet-stream"
		switch ext {
		case ".xlsx":
			contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case ".csv":
			contentType = "text/csv; charset=utf-8"
		case ".json":
			contentType = "application/json; charset=utf-8"
		case ".sql":
			contentType = "text/plain; charset=utf-8"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return nil
	})

	r.PUT("/api/v1/permission/users/{id}", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		idStr := ctx.Vars().Get("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, http.StatusBadRequest, 400, "无效的用户ID")
			return nil
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, 400, "读取请求体失败")
			return nil
		}
		var reqBody struct {
			Email      string `json:"email"`
			RealName   string `json:"real_name"`
			Phone      string `json:"phone"`
			Department string `json:"department"`
			Status     *int   `json:"status"`
			Password   string `json:"password"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			writeError(w, http.StatusBadRequest, 400, "请求参数格式错误")
			return nil
		}

		// 获取现有用户
		existing, getErr := userUsecase.GetUser(req.Context(), id)
		if getErr != nil {
			writeError(w, http.StatusNotFound, 404, "用户不存在")
			return nil
		}

		// 仅更新提供的字段
		if reqBody.Email != "" {
			existing.Email = reqBody.Email
		}
		if reqBody.RealName != "" {
			existing.RealName = reqBody.RealName
		}
		if reqBody.Phone != "" {
			existing.Phone = reqBody.Phone
		}
		if reqBody.Department != "" {
			existing.Department = reqBody.Department
		}
		if reqBody.Status != nil {
			existing.Status = *reqBody.Status
		}
		// 任务3：若 password 非空则用 bcrypt 加密后更新
		if reqBody.Password != "" {
			hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(reqBody.Password), bcrypt.DefaultCost)
			if hashErr != nil {
				writeError(w, http.StatusInternalServerError, 500, "密码加密失败")
				return nil
			}
			existing.Password = string(hashedPassword)
		}

		if updateErr := userUsecase.UpdateUser(req.Context(), existing); updateErr != nil {
			if updateErr == biz.ErrEmailExists {
				writeError(w, http.StatusBadRequest, 400, "邮箱已被其他用户使用")
			} else {
				writeError(w, http.StatusInternalServerError, 500, updateErr.Error())
			}
			return nil
		}

		// 返回更新后的用户信息
		updated, getErr2 := userUsecase.GetUser(req.Context(), id)
		if getErr2 != nil {
			writeSuccess(w, bizUserToItem(existing))
			return nil
		}
		writeSuccess(w, bizUserToItem(updated))
		return nil
	})
}

// registerPartitionRoutes 注册分区管理相关路由
func registerPartitionRoutes(srv *kratoshttp.Server, authUsecase *biz.AuthUsecase, partitionUsecase *biz.PartitionUsecase) {
	r := srv.Route("/")

	// ----------------------------------------------------------------
	// POST /api/v1/partitions/configs — 创建分区配置
	// ----------------------------------------------------------------
	r.POST("/api/v1/partitions/configs", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, 400, "读取请求体失败")
			return nil
		}
		var reqBody struct {
			InstanceID   uint64  `json:"instance_id"`
			InstanceName string  `json:"instance_name"`
			DatabaseName string  `json:"database_name"`
			TableName    string  `json:"table_name"`
			PartType     string  `json:"part_type"`
			PartKey      string  `json:"part_key"`
			PartCycle    string  `json:"part_cycle"`
			PartNum      int     `json:"part_num"`
			IsCrontab    string  `json:"is_crontab"`
			TableSize    float64 `json:"table_size"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			writeError(w, http.StatusBadRequest, 400, "请求参数格式错误")
			return nil
		}
		if reqBody.InstanceID == 0 || reqBody.DatabaseName == "" || reqBody.TableName == "" {
			writeError(w, http.StatusBadRequest, 400, "instance_id、database_name、table_name为必填项")
			return nil
		}

		if reqBody.IsCrontab == "" {
			reqBody.IsCrontab = "否"
		}

		info := &biz.PartitionTableInfo{
			InstanceID:   reqBody.InstanceID,
			InstanceName: reqBody.InstanceName,
			DatabaseName: reqBody.DatabaseName,
			TableName:    reqBody.TableName,
			PartType:     reqBody.PartType,
			PartKey:      reqBody.PartKey,
			PartCycle:    reqBody.PartCycle,
			PartNum:      reqBody.PartNum,
			IsCrontab:    reqBody.IsCrontab,
			TableSize:    reqBody.TableSize,
		}

		created, err := partitionUsecase.CreateConfig(req.Context(), info)
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		writeSuccess(w, partitionTableInfoToMap(created))
		return nil
	})

	// ----------------------------------------------------------------
	// GET /api/v1/partitions/configs — 列出分区配置（分页）
	// ----------------------------------------------------------------
	r.GET("/api/v1/partitions/configs", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		query := req.URL.Query()
		page, _ := strconv.Atoi(query.Get("page"))
		pageSize, _ := strconv.Atoi(query.Get("page_size"))
		instanceID, _ := strconv.ParseUint(query.Get("instance_id"), 10, 64)
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		list, total, err := partitionUsecase.ListConfigs(req.Context(), instanceID, page, pageSize)
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		items := make([]map[string]interface{}, 0, len(list))
		for _, info := range list {
			items = append(items, partitionTableInfoToMap(info))
		}

		writeSuccess(w, map[string]interface{}{
			"list":      items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
		return nil
	})

	// ----------------------------------------------------------------
	// GET /api/v1/partitions/configs/{id} — 获取单个分区配置
	// ----------------------------------------------------------------
	r.GET("/api/v1/partitions/configs/{id}", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		idStr := ctx.Vars().Get("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, http.StatusBadRequest, 400, "无效的ID")
			return nil
		}

		info, err := partitionUsecase.GetConfigByID(req.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, 404, "分区配置不存在")
			return nil
		}

		writeSuccess(w, partitionTableInfoToMap(info))
		return nil
	})

	// ----------------------------------------------------------------
	// PUT /api/v1/partitions/configs/{id} — 更新分区配置
	// ----------------------------------------------------------------
	r.PUT("/api/v1/partitions/configs/{id}", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		idStr := ctx.Vars().Get("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, http.StatusBadRequest, 400, "无效的ID")
			return nil
		}

		existing, err := partitionUsecase.GetConfigByID(req.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, 404, "分区配置不存在")
			return nil
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, 400, "读取请求体失败")
			return nil
		}
		var reqBody struct {
			InstanceID   *uint64  `json:"instance_id"`
			InstanceName *string  `json:"instance_name"`
			DatabaseName *string  `json:"database_name"`
			TableName    *string  `json:"table_name"`
			PartType     *string  `json:"part_type"`
			PartKey      *string  `json:"part_key"`
			PartCycle    *string  `json:"part_cycle"`
			PartNum      *int     `json:"part_num"`
			IsCrontab    *string  `json:"is_crontab"`
			TableSize    *float64 `json:"table_size"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			writeError(w, http.StatusBadRequest, 400, "请求参数格式错误")
			return nil
		}

		if reqBody.InstanceID != nil {
			existing.InstanceID = *reqBody.InstanceID
		}
		if reqBody.InstanceName != nil {
			existing.InstanceName = *reqBody.InstanceName
		}
		if reqBody.DatabaseName != nil {
			existing.DatabaseName = *reqBody.DatabaseName
		}
		if reqBody.TableName != nil {
			existing.TableName = *reqBody.TableName
		}
		if reqBody.PartType != nil {
			existing.PartType = *reqBody.PartType
		}
		if reqBody.PartKey != nil {
			existing.PartKey = *reqBody.PartKey
		}
		if reqBody.PartCycle != nil {
			existing.PartCycle = *reqBody.PartCycle
		}
		if reqBody.PartNum != nil {
			existing.PartNum = *reqBody.PartNum
		}
		if reqBody.IsCrontab != nil {
			existing.IsCrontab = *reqBody.IsCrontab
		}
		if reqBody.TableSize != nil {
			existing.TableSize = *reqBody.TableSize
		}

		if err := partitionUsecase.UpdateConfig(req.Context(), existing); err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		updated, err := partitionUsecase.GetConfigByID(req.Context(), id)
		if err != nil {
			writeSuccess(w, partitionTableInfoToMap(existing))
			return nil
		}
		writeSuccess(w, partitionTableInfoToMap(updated))
		return nil
	})

	// ----------------------------------------------------------------
	// DELETE /api/v1/partitions/configs/{id} — 删除分区配置
	// ----------------------------------------------------------------
	r.DELETE("/api/v1/partitions/configs/{id}", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		idStr := ctx.Vars().Get("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, http.StatusBadRequest, 400, "无效的ID")
			return nil
		}

		if err := partitionUsecase.DeleteConfig(req.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		writeSuccess(w, nil)
		return nil
	})

	// ----------------------------------------------------------------
	// GET /api/v1/partitions/configs/{id}/records — 获取分区操作记录
	// ----------------------------------------------------------------
	r.GET("/api/v1/partitions/configs/{id}/records", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		idStr := ctx.Vars().Get("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			writeError(w, http.StatusBadRequest, 400, "无效的ID")
			return nil
		}

		query := req.URL.Query()
		page, _ := strconv.Atoi(query.Get("page"))
		pageSize, _ := strconv.Atoi(query.Get("page_size"))
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		records, total, err := partitionUsecase.ListRecordsByConfigID(req.Context(), id, page, pageSize)
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		items := make([]map[string]interface{}, 0, len(records))
		for _, rec := range records {
			items = append(items, partitionOperationRecordToMap(rec))
		}

		writeSuccess(w, map[string]interface{}{
			"list":      items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
		return nil
	})
}

// partitionTableInfoToMap 将分区表信息转为 map
func partitionTableInfoToMap(info *biz.PartitionTableInfo) map[string]interface{} {
	return map[string]interface{}{
		"id":                 info.ID,
		"instance_id":        info.InstanceID,
		"instance_name":      info.InstanceName,
		"connection_string":  info.ConnectionString,
		"database_name":      info.DatabaseName,
		"table_name":         info.TableName,
		"part_type":          info.PartType,
		"part_key":           info.PartKey,
		"part_cycle":         info.PartCycle,
		"part_num":           info.PartNum,
		"part_maxvalue":      info.PartMaxValue,
		"part_maxvalue_day":  info.PartMaxValueDay,
		"have_maxvalue":      info.HaveMaxValue,
		"table_size":         info.TableSize,
		"table_rows":         info.TableRows,
		"max_part_rows":      info.MaxPartRows,
		"is_crontab":         info.IsCrontab,
		"created_at":         info.CreatedAt,
		"updated_at":         info.UpdatedAt,
	}
}

// partitionOperationRecordToMap 将分区操作记录转为 map
func partitionOperationRecordToMap(rec *biz.PartitionOperationRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":                rec.ID,
		"work_id":           rec.WorkID,
		"instance_id":       rec.InstanceID,
		"instance_name":     rec.InstanceName,
		"connection_string": rec.ConnectionString,
		"table_schema":      rec.TableSchema,
		"table_name":        rec.TableName,
		"operate_type":      rec.OperateType,
		"exe_status":        rec.ExeStatus,
		"check_status":      rec.CheckStatus,
		"exe_pct":           rec.ExePct,
		"exe_result":        rec.ExeResult,
		"exe_cmd":           rec.ExeCmd,
		"where_condition":   rec.WhereCondition,
		"add_user":          rec.AddUser,
		"update_user":       rec.UpdateUser,
		"exe_time":          rec.ExeTime,
		"created_at":        rec.CreatedAt,
		"updated_at":        rec.UpdatedAt,
	}
}

// registerNotificationRoutes 注册通知渠道配置相关路由
func registerNotificationRoutes(srv *kratoshttp.Server, authUsecase *biz.AuthUsecase, notificationService *biz.NotificationService) {
	r := srv.Route("/")

	// ----------------------------------------------------------------
	// GET /api/v1/notification/configs — 获取所有通知渠道配置
	// ----------------------------------------------------------------
	r.GET("/api/v1/notification/configs", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		configs, err := notificationService.GetConfigs(req.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		list := make([]map[string]interface{}, 0, len(configs))
		for _, c := range configs {
			list = append(list, notificationConfigToMap(c))
		}
		writeSuccess(w, list)
		return nil
	})

	// ----------------------------------------------------------------
	// PUT /api/v1/notification/configs/{channel} — 更新指定渠道配置
	// ----------------------------------------------------------------
	r.PUT("/api/v1/notification/configs/{channel}", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		channel := ctx.Vars().Get("channel")
		if channel == "" {
			writeError(w, http.StatusBadRequest, 400, "渠道名称不能为空")
			return nil
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, 400, "读取请求体失败")
			return nil
		}
		var reqBody struct {
			Enabled     bool   `json:"enabled"`
			WebhookURL  string `json:"webhook_url"`
			ExtraConfig string `json:"extra_config"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			writeError(w, http.StatusBadRequest, 400, "请求参数格式错误")
			return nil
		}

		config := &biz.NotificationConfig{
			Channel:     channel,
			Enabled:     reqBody.Enabled,
			WebhookURL:  reqBody.WebhookURL,
			ExtraConfig: reqBody.ExtraConfig,
		}
		if err := notificationService.UpsertConfig(req.Context(), config); err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return nil
		}

		writeSuccess(w, notificationConfigToMap(config))
		return nil
	})

	// ----------------------------------------------------------------
	// POST /api/v1/notification/test/{channel} — 发送测试消息
	// ----------------------------------------------------------------
	r.POST("/api/v1/notification/test/{channel}", func(ctx kratoshttp.Context) error {
		req := ctx.Request()
		w := ctx.Response()

		tokenStr := extractBearerToken(req)
		if tokenStr == "" {
			writeError(w, http.StatusUnauthorized, 401, "未认证")
			return nil
		}
		if _, err := authUsecase.ValidateToken(tokenStr); err != nil {
			writeError(w, http.StatusUnauthorized, 401, "token无效或已过期")
			return nil
		}

		channel := ctx.Vars().Get("channel")
		if err := notificationService.TestChannel(req.Context(), channel); err != nil {
			writeError(w, http.StatusBadRequest, 400, err.Error())
			return nil
		}

		writeSuccess(w, map[string]string{"message": "测试消息发送成功"})
		return nil
	})
}

func notificationConfigToMap(c *biz.NotificationConfig) map[string]interface{} {
	return map[string]interface{}{
		"id":           c.ID,
		"channel":      c.Channel,
		"enabled":      c.Enabled,
		"webhook_url":  c.WebhookURL,
		"extra_config": c.ExtraConfig,
		"created_at":   c.CreatedAt,
		"updated_at":   c.UpdatedAt,
	}
}
