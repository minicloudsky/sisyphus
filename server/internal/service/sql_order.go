package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	asynqPkg "sisyphus/server/internal/pkg/asynq"
)

// SQLOrderService SQL工单服务
type SQLOrderService struct {
	sqlOrderUsecase     *biz.SQLOrderUsecase
	sqlExecutor         *biz.SQLExecutor
	instanceUsecase     *biz.DatabaseInstanceUsecase
	executorSelector    *biz.SQLExecutorSelector
	inceptionConfig     *biz.InceptionConfig
	notificationService *biz.NotificationService
	rollbackUsecase     *biz.RollbackUsecase
	asynqClient         *asynqPkg.Client
	authUsecase         *biz.AuthUsecase
	permissionUsecase   *biz.PermissionUsecase
	log                 *log.Helper
}

// NewSQLOrderService 创建SQL工单服务
func NewSQLOrderService(
	sqlOrderUsecase *biz.SQLOrderUsecase,
	sqlExecutor *biz.SQLExecutor,
	instanceUsecase *biz.DatabaseInstanceUsecase,
	executorSelector *biz.SQLExecutorSelector,
	inceptionConfig *biz.InceptionConfig,
	notificationService *biz.NotificationService,
	rollbackUsecase *biz.RollbackUsecase,
	asynqClient *asynqPkg.Client,
	authUsecase *biz.AuthUsecase,
	permissionUsecase *biz.PermissionUsecase,
	logger log.Logger,
) *SQLOrderService {
	return &SQLOrderService{
		sqlOrderUsecase:     sqlOrderUsecase,
		sqlExecutor:         sqlExecutor,
		instanceUsecase:     instanceUsecase,
		executorSelector:    executorSelector,
		inceptionConfig:     inceptionConfig,
		notificationService: notificationService,
		rollbackUsecase:     rollbackUsecase,
		asynqClient:         asynqClient,
		authUsecase:         authUsecase,
		permissionUsecase:   permissionUsecase,
		log:                 log.NewHelper(logger),
	}
}

// extractTableName 从SQL中提取表名（简化实现）
func (s *SQLOrderService) extractTableName(sql string) string {
	sql = strings.TrimSpace(sql)
	sqlLower := strings.ToLower(sql)

	if strings.HasPrefix(sqlLower, "alter table") {
		parts := strings.Fields(sql)
		if len(parts) >= 3 {
			tableName := parts[2]
			// 移除反引号
			tableName = strings.Trim(tableName, "`")
			// 处理database.table格式
			if strings.Contains(tableName, ".") {
				tableName = strings.Split(tableName, ".")[1]
			}
			return tableName
		}
	}

	return ""
}

// executeWithGoInception 使用goInception执行
func (s *SQLOrderService) executeWithGoInception(ctx context.Context, instance *biz.InstanceInfo, order *biz.SQLOrder) error {
	// 从配置中获取goInception服务地址和端口
	inceptionHost := s.inceptionConfig.Host
	inceptionPort := int(s.inceptionConfig.Port)
	inceptionUsername := s.inceptionConfig.Username
	inceptionPassword := s.inceptionConfig.Password

	// 构建goInception客户端
	client := biz.NewGoInceptionClient(
		inceptionHost,     // goInception服务地址
		inceptionPort,     // goInception服务端口
		instance.Host,     // 目标数据库地址
		instance.Port,     // 目标数据库端口
		instance.Username, // 目标数据库用户名
		instance.Password, // 目标数据库密码
	)

	// 如果配置了goInception的用户名和密码，使用配置的值
	if inceptionUsername != "" {
		client.SetInceptionCredentials(inceptionUsername, inceptionPassword)
	}

	// 执行SQL
	results, err := s.sqlExecutor.ExecuteWithGoInception(ctx, client, order.SQLContent, order.DatabaseName, order.NeedBackup)
	if err != nil {
		return err
	}

	// 如果执行成功且需要备份，生成并保存回滚SQL
	if s.rollbackUsecase != nil && order.NeedBackup {
		for _, result := range results {
			if result.Success && result.BackupDBName != "" && result.Sequence != "" {
				// 异步生成并保存回滚SQL
				go func(backupDBName, opidTime string) {
					if err := s.rollbackUsecase.GenerateAndSaveRollbackSQL(
						context.Background(),
						order.ID,
						nil, // orderItemID暂时为nil，后续可以从result中获取
						backupDBName,
						opidTime,
					); err != nil {
						s.log.Warnf("生成回滚SQL失败: order_id=%d, error=%v", order.ID, err)
					}
				}(result.BackupDBName, result.Sequence)
			}
		}
	}

	return nil
}

// ========== Protobuf 接口实现 ==========

// CreateOrder 创建工单（实现 Protobuf 接口）
func (s *SQLOrderService) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
	claims, err := getClaimsFromContext(ctx, s.authUsecase)
	if err != nil {
		return &v1.CreateOrderResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}

	order := toBizSQLOrder(req, claims.UserID)

	// 获取实例信息（用于生成工单编号）
	instance, err := s.instanceUsecase.GetInstanceByID(ctx, order.InstanceID)
	if err != nil {
		return &v1.CreateOrderResponse{
			Code:    500,
			Message: fmt.Sprintf("获取实例信息失败: %v", err),
		}, nil
	}

	instPerm := biz.InstancePermForSQLOrderType(order.OrderType)
	ok, err := s.permissionUsecase.CanAccessInstance(ctx, claims.UserID, order.InstanceID, instPerm)
	if err != nil {
		return &v1.CreateOrderResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !ok {
		return &v1.CreateOrderResponse{Code: 403, Message: "无该实例上对应工单类型的权限"}, nil
	}

	// 创建工单（传入实例名用于生成工单编号）
	order, err = s.sqlOrderUsecase.CreateOrder(ctx, order, instance.Name)
	if err != nil {
		return &v1.CreateOrderResponse{
			Code:    500,
			Message: fmt.Sprintf("创建工单失败: %v", err),
		}, nil
	}

	// 创建工单后，立即进行SQL审核并保存审核记录
	if err != nil {
		s.log.Warnf("获取实例信息失败，跳过审核记录保存: %v", err)
	} else {
		// 调用goInception进行审核
		client := biz.NewGoInceptionClient(
			s.inceptionConfig.Host,
			int(s.inceptionConfig.Port),
			instance.Host,
			instance.Port,
			instance.Username,
			instance.Password,
		)
		results, err := client.ReviewSQL(ctx, order.SQLContent, order.DatabaseName)
		if err != nil {
			s.log.Warnf("审核SQL失败，跳过审核记录保存: %v", err)
		} else {
			// 保存审核记录
			// goInception会按照分号拆分SQL，每条SQL返回一个结果
			for i, result := range results {
				// 获取状态的中文说明
				statusInfo := biz.GetGoInceptionStatusText(result.StageStatus)

				// 使用原始SQL内容，如果goInception返回的SQL为空，则使用原始SQL
				sqlText := strings.TrimSpace(result.SQLText)
				if sqlText == "" {
					// 如果goInception返回的SQL为空，尝试从原始SQL中提取
					// 按照分号拆分原始SQL
					sqlParts := strings.Split(order.SQLContent, ";")
					if i < len(sqlParts) {
						sqlText = strings.TrimSpace(sqlParts[i])
					} else {
						sqlText = order.SQLContent
					}
				}

				record := &biz.SQLReviewRecord{
					OrderID:      order.ID,
					ItemID:       0, // 数据库表中没有item_id字段，默认为0
					Sequence:     fmt.Sprintf("%d", i+1),
					Stage:        "", // 数据库表中可能没有stage字段，设为空字符串
					ErrorLevel:   result.ErrorLevel,
					StageStatus:  statusInfo.Text, // 使用转换后的中文状态
					ErrorMessage: result.ErrorMessage,
					SQLText:      sqlText,
					AffectedRows: result.AffectedRows,
					BackupDBName: result.BackupDBName,
					ExecuteTime:  result.ExecuteTime,
					SQLHash:      result.SQLHash,
				}
				err = s.sqlOrderUsecase.CreateReviewRecord(ctx, record)
				if err != nil {
					s.log.Warnf("保存审核记录失败: %v", err)
				} else {
					s.log.Infof("保存审核记录成功: orderID=%d, sequence=%d, sqlText=%s", order.ID, i+1, sqlText)
				}
			}
		}
	}

	return &v1.CreateOrderResponse{
		Code:    200,
		Message: "success",
		Data:    toProtoSQLOrder(order),
	}, nil
}

// ListOrders 获取工单列表（实现 Protobuf 接口）
func (s *SQLOrderService) ListOrders(ctx context.Context, req *v1.ListOrdersRequest) (*v1.ListOrdersResponse, error) {
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}

	var statusPtr *int
	if req.Status != 0 {
		status := int(req.Status)
		statusPtr = &status
	}

	var submitterIDPtr *uint64
	if req.SubmitterId != 0 {
		submitterIDPtr = &req.SubmitterId
	}

	orders, total, err := s.sqlOrderUsecase.ListOrders(ctx, page, pageSize, statusPtr, submitterIDPtr)
	if err != nil {
		return &v1.ListOrdersResponse{
			Code:    500,
			Message: fmt.Sprintf("查询工单失败: %v", err),
		}, nil
	}

	protoOrders := make([]*v1.SQLOrder, len(orders))
	for i, order := range orders {
		protoOrders[i] = toProtoSQLOrder(order)
	}

	return &v1.ListOrdersResponse{
		Code:    200,
		Message: "success",
		Data: &v1.ListOrdersData{
			List: protoOrders,
			Page: &v1.PageResponse{
				Page:     int32(page),
				PageSize: int32(pageSize),
				Total:    total,
			},
		},
	}, nil
}

// GetOrder 获取工单详情（实现 Protobuf 接口）
func (s *SQLOrderService) GetOrder(ctx context.Context, req *v1.GetOrderRequest) (*v1.GetOrderResponse, error) {
	order, err := s.sqlOrderUsecase.GetOrder(ctx, req.Id)
	if err != nil {
		return &v1.GetOrderResponse{
			Code:    404,
			Message: "工单不存在",
		}, nil
	}

	return &v1.GetOrderResponse{
		Code:    200,
		Message: "success",
		Data:    toProtoSQLOrder(order),
	}, nil
}

// ApproveOrder 审核通过（实现 Protobuf 接口）
func (s *SQLOrderService) ApproveOrder(ctx context.Context, req *v1.ApproveOrderRequest) (*v1.ApproveOrderResponse, error) {
	claims, err := getClaimsFromContext(ctx, s.authUsecase)
	if err != nil {
		s.log.Errorf("ApproveOrder认证失败: %v", err)
		return &v1.ApproveOrderResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}
	s.log.Infof("ApproveOrder认证成功: user_id=%d, order_id=%d", claims.UserID, req.Id)

	err = s.sqlOrderUsecase.ApproveOrder(ctx, req.Id, claims.UserID)
	if err != nil {
		return &v1.ApproveOrderResponse{
			Code:    500,
			Message: fmt.Sprintf("审核失败: %v", err),
		}, nil
	}

	// 获取工单信息并发送通知
	order, err := s.sqlOrderUsecase.GetOrder(ctx, req.Id)
	if err == nil && s.notificationService != nil {
		_ = s.notificationService.SendSQLOrderNotification(ctx, order, "审核通过", claims.Username)
	}

	return &v1.ApproveOrderResponse{
		Code:    200,
		Message: "工单已审核通过",
	}, nil
}

// RejectOrder 审核驳回（实现 Protobuf 接口）
func (s *SQLOrderService) RejectOrder(ctx context.Context, req *v1.RejectOrderRequest) (*v1.RejectOrderResponse, error) {
	claims, err := getClaimsFromContext(ctx, s.authUsecase)
	if err != nil {
		return &v1.RejectOrderResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}

	err = s.sqlOrderUsecase.RejectOrder(ctx, req.Id, claims.UserID, req.Reason)
	if err != nil {
		return &v1.RejectOrderResponse{
			Code:    500,
			Message: fmt.Sprintf("驳回失败: %v", err),
		}, nil
	}

	// 获取工单信息并发送通知
	order, err := s.sqlOrderUsecase.GetOrder(ctx, req.Id)
	if err == nil && s.notificationService != nil {
		_ = s.notificationService.SendSQLOrderNotification(ctx, order, "审核驳回", claims.Username)
	}

	return &v1.RejectOrderResponse{
		Code:    200,
		Message: "工单已驳回",
	}, nil
}

// ExecuteOrder 执行工单（实现 Protobuf 接口）
func (s *SQLOrderService) ExecuteOrder(ctx context.Context, req *v1.ExecuteOrderRequest) (*v1.ExecuteOrderResponse, error) {
	claims, err := getClaimsFromContext(ctx, s.authUsecase)
	if err != nil {
		return &v1.ExecuteOrderResponse{
			Code:    401,
			Message: "未认证",
		}, nil
	}

	order, err := s.sqlOrderUsecase.GetOrder(ctx, req.Id)
	if err != nil {
		return &v1.ExecuteOrderResponse{
			Code:    404,
			Message: "工单不存在",
		}, nil
	}

	instPerm := biz.InstancePermForSQLOrderType(order.OrderType)
	ok, err := s.permissionUsecase.CanAccessInstance(ctx, claims.UserID, order.InstanceID, instPerm)
	if err != nil {
		return &v1.ExecuteOrderResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !ok {
		return &v1.ExecuteOrderResponse{Code: 403, Message: "无该实例上执行该工单类型的权限"}, nil
	}

	if order.Status != biz.OrderStatusApproved {
		return &v1.ExecuteOrderResponse{
			Code:    400,
			Message: "工单状态不允许执行",
		}, nil
	}

	// 使用Asynq入队任务
	payload := biz.SQLOrderExecutePayload{
		OrderID:       order.ID,
		ExecutorID:    claims.UserID,
		ExecuteMethod: int(req.ExecuteMethod),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return &v1.ExecuteOrderResponse{
			Code:    500,
			Message: fmt.Sprintf("序列化任务负载失败: %v", err),
		}, nil
	}

	// 计算延迟执行时间（如果有）
	var opts []asynq.Option
	if order.DelayMinutes > 0 {
		opts = append(opts, asynq.ProcessIn(time.Duration(order.DelayMinutes)*time.Minute))
	}

	// 使用critical队列（高优先级）
	opts = append(opts, asynq.Queue("critical"))

	// 设置任务ID为工单ID+时间戳，避免重复执行时的ID冲突
	opts = append(opts, asynq.TaskID(fmt.Sprintf("sql_order:%d:%d", order.ID, time.Now().UnixNano())))

	taskInfo, err := s.asynqClient.Enqueue(asynqPkg.TaskTypeSQLOrderExecute, payloadBytes, opts...)
	if err != nil {
		return &v1.ExecuteOrderResponse{
			Code:    500,
			Message: fmt.Sprintf("入队任务失败: %v", err),
		}, nil
	}

	// TODO: 保存task_id到数据库，便于后续查询和取消

	return &v1.ExecuteOrderResponse{
		Code:    200,
		Message: fmt.Sprintf("工单执行任务已发起，任务ID: %s", taskInfo.ID),
	}, nil
}

// executeOrderAsyncProtobuf 异步执行工单（Protobuf版本）
func (s *SQLOrderService) executeOrderAsyncProtobuf(
	ctx context.Context,
	order *biz.SQLOrder,
	executeMethod int,
	executorID uint64,
) {
	// 更新工单状态为执行中
	order.Status = biz.OrderStatusExecuting
	order.ExecutorID = executorID
	now := time.Now()
	order.ExecuteTime = &now
	s.sqlOrderUsecase.UpdateOrder(ctx, order)

	// 获取实例信息
	instance, err := s.instanceUsecase.GetInstanceByID(ctx, order.InstanceID)
	if err != nil {
		s.log.Errorf("获取实例信息失败: %v", err)
		order.Status = biz.OrderStatusFailed
		s.sqlOrderUsecase.UpdateOrder(ctx, order)
		return
	}

	// 构建实例信息
	instanceInfo := &biz.InstanceInfo{
		Host:     instance.Host,
		Port:     instance.Port,
		Username: instance.Username,
		Password: instance.Password,
	}

	// 根据执行方式执行（如果未指定，则自动选择）
	if executeMethod == 0 {
		// 自动根据SQL类型选择执行方式
		method, err := s.executorSelector.SelectExecuteMethod(ctx, order.SQLContent)
		if err != nil {
			s.log.Warnf("自动选择执行方式失败，使用默认goInception: order_id=%d, error=%v", order.ID, err)
			method = biz.ExecuteMethodGoInception
		}
		executeMethod = method
		// 更新工单的执行方式
		order.ExecuteMethod = executeMethod
		s.sqlOrderUsecase.UpdateOrder(ctx, order)
	}

	var execErr error
	if executeMethod == biz.ExecuteMethodGhOst {
		// 使用gh-ost执行（需要提取表名）
		tableName := s.extractTableName(order.SQLContent)
		if tableName == "" {
			order.Status = biz.OrderStatusFailed
			s.sqlOrderUsecase.UpdateOrder(ctx, order)
			s.log.Errorf("无法从SQL中提取表名: order_id=%d", order.ID)
			return
		}
		// TODO: 从工单配置中获取delete_gho参数
		deleteGho := false
		execErr = s.sqlExecutor.ExecuteWithGhOst(ctx, instance, order.SQLContent, order.DatabaseName, tableName, order.WorkID, deleteGho)
	} else {
		// 使用goInception执行
		execErr = s.executeWithGoInception(ctx, instanceInfo, order)
	}

	// 更新工单状态
	if execErr != nil {
		order.Status = biz.OrderStatusFailed
		s.log.Errorf("工单执行失败: order_id=%d, error=%v", order.ID, execErr)
	} else {
		order.Status = biz.OrderStatusSuccess
	}
	s.sqlOrderUsecase.UpdateOrder(ctx, order)
}

// CheckSQLType 检查SQL类型（实现 Protobuf 接口）
func (s *SQLOrderService) CheckSQLType(ctx context.Context, req *v1.CheckSQLTypeRequest) (*v1.CheckSQLTypeResponse, error) {
	types, canAuto, err := s.sqlOrderUsecase.CheckSQLType(ctx, req.Sql)
	if err != nil {
		return &v1.CheckSQLTypeResponse{
			Code:    500,
			Message: fmt.Sprintf("检查SQL类型失败: %v", err),
		}, nil
	}

	checker := biz.NewSQLTypeChecker()
	isDangerous, _, _ := checker.IsDangerousSQL(ctx, req.Sql)

	return &v1.CheckSQLTypeResponse{
		Code:    200,
		Message: "success",
		Data: &v1.CheckSQLTypeData{
			Types:          types,
			CanAutoExecute: canAuto && !isDangerous,
		},
	}, nil
}

// AuditSQL 审核SQL（实现 Protobuf 接口）
func (s *SQLOrderService) AuditSQL(ctx context.Context, req *v1.AuditSQLRequest) (*v1.AuditSQLResponse, error) {
	claims, err := getClaimsFromContext(ctx, s.authUsecase)
	if err != nil {
		return &v1.AuditSQLResponse{Code: 401, Message: "未认证"}, nil
	}

	order, err := s.sqlOrderUsecase.GetOrder(ctx, req.Id)
	if err != nil {
		return &v1.AuditSQLResponse{
			Code:    404,
			Message: "工单不存在",
		}, nil
	}

	instPerm := biz.InstancePermForSQLOrderType(order.OrderType)
	ok, err := s.permissionUsecase.CanAccessInstance(ctx, claims.UserID, order.InstanceID, instPerm)
	if err != nil {
		return &v1.AuditSQLResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !ok {
		return &v1.AuditSQLResponse{Code: 403, Message: "无该实例上对应工单类型的权限"}, nil
	}

	// 使用goInception审核SQL
	instance, err := s.instanceUsecase.GetInstanceByID(ctx, order.InstanceID)
	if err != nil {
		return &v1.AuditSQLResponse{
			Code:    500,
			Message: "获取实例信息失败",
		}, nil
	}

	// 从配置中获取goInception服务地址和端口
	inceptionHost := s.inceptionConfig.Host
	inceptionPort := int(s.inceptionConfig.Port)

	client := biz.NewGoInceptionClient(
		inceptionHost,     // goInception服务地址
		inceptionPort,     // goInception服务端口
		instance.Host,     // 目标数据库地址
		instance.Port,     // 目标数据库端口
		instance.Username, // 目标数据库用户名
		instance.Password, // 目标数据库密码
	)

	results, err := client.ReviewSQL(ctx, order.SQLContent, req.DatabaseName)
	if err != nil {
		return &v1.AuditSQLResponse{
			Code:    500,
			Message: fmt.Sprintf("审核SQL失败: %v", err),
		}, nil
	}

	// 转换结果格式
	protoResults := make([]*v1.SQLReviewRecord, len(results))
	for i, r := range results {
		// OrderID 在 SQLReviewResult 中是 string 类型（序列号），不是工单ID
		sequence := int32(0)
		if seq, err := strconv.ParseInt(r.OrderID, 10, 32); err == nil {
			sequence = int32(seq)
		}

		// 获取状态的中文说明
		statusInfo := biz.GetGoInceptionStatusText(r.StageStatus)

		protoResults[i] = &v1.SQLReviewRecord{
			Id:           0, // 审核记录ID，需要从数据库获取
			OrderId:      order.ID,
			Sequence:     sequence,
			SqlText:      r.SQLText,
			ErrorLevel:   int32(r.ErrorLevel),
			ErrorMessage: r.ErrorMessage,
			StageStatus:  statusInfo.Text, // 使用转换后的中文状态
			AffectedRows: int32(r.AffectedRows),
		}
	}

	return &v1.AuditSQLResponse{
		Code:    200,
		Message: "success",
		Data:    protoResults,
	}, nil
}

// ReviewSQL 直接审核SQL（不需要工单ID，实现 Protobuf 接口）
func (s *SQLOrderService) ReviewSQL(ctx context.Context, req *v1.ReviewSQLRequest) (*v1.ReviewSQLResponse, error) {
	claims, err := getClaimsFromContext(ctx, s.authUsecase)
	if err != nil {
		return &v1.ReviewSQLResponse{Code: 401, Message: "未认证"}, nil
	}

	sqlTypes, _, err := s.sqlOrderUsecase.CheckSQLType(ctx, req.GetSqlContent())
	if err != nil {
		return &v1.ReviewSQLResponse{Code: 500, Message: fmt.Sprintf("解析SQL类型失败: %v", err)}, nil
	}
	needPerm := biz.InstancePermDML
	for _, t := range sqlTypes {
		if strings.EqualFold(strings.TrimSpace(t), "DDL") {
			needPerm = biz.InstancePermDDL
			break
		}
	}
	ok, err := s.permissionUsecase.CanAccessInstance(ctx, claims.UserID, req.InstanceId, needPerm)
	if err != nil {
		return &v1.ReviewSQLResponse{Code: 500, Message: fmt.Sprintf("权限校验失败: %v", err)}, nil
	}
	if !ok {
		return &v1.ReviewSQLResponse{Code: 403, Message: "无该实例上对应 SQL 类型的工单权限"}, nil
	}

	// 获取实例信息
	instance, err := s.instanceUsecase.GetInstanceByID(ctx, req.InstanceId)
	if err != nil {
		return &v1.ReviewSQLResponse{
			Code:    404,
			Message: "实例不存在",
		}, nil
	}

	// 从配置中获取goInception服务地址和端口
	inceptionHost := s.inceptionConfig.Host
	inceptionPort := int(s.inceptionConfig.Port)

	client := biz.NewGoInceptionClient(
		inceptionHost,     // goInception服务地址
		inceptionPort,     // goInception服务端口
		instance.Host,     // 目标数据库地址
		instance.Port,     // 目标数据库端口
		instance.Username, // 目标数据库用户名
		instance.Password, // 目标数据库密码
	)

	results, err := client.ReviewSQL(ctx, req.SqlContent, req.DatabaseName)
	if err != nil {
		s.log.Errorf("ReviewSQL failed: %v", err)
		return &v1.ReviewSQLResponse{
			Code:    500,
			Message: fmt.Sprintf("审核SQL失败: %v", err),
		}, nil
	}

	s.log.Infof("ReviewSQL returned %d results", len(results))

	// 转换结果格式
	protoResults := make([]*v1.SQLReviewRecord, len(results))
	for i, r := range results {
		// OrderID 在 SQLReviewResult 中是 string 类型（序列号），不是工单ID
		sequence := int32(0)
		if seq, err := strconv.ParseInt(r.OrderID, 10, 32); err == nil {
			sequence = int32(seq)
		}

		// 获取状态的中文说明
		statusInfo := biz.GetGoInceptionStatusText(r.StageStatus)

		// 调试日志
		s.log.Infof("Result[%d]: Original SQLText=%s, StageStatus=%s, ErrorLevel=%d, req.SqlContent=%s",
			i, r.SQLText, r.StageStatus, r.ErrorLevel, req.SqlContent)

		// 确保SQL语句不为空，优先使用原始SQL（因为goInception返回的SQL可能包含包装语句）
		// 重要：始终使用用户输入的原始SQL，因为goInception返回的SQL可能包含包装语句
		originalSQL := strings.TrimSpace(req.SqlContent)
		sqlText := originalSQL

		// 如果原始SQL为空，才尝试使用goInception返回的SQL
		if sqlText == "" {
			sqlText = strings.TrimSpace(r.SQLText)
			s.log.Warnf("Result[%d]: Original SQL is empty, using goInception SQL: %s", i, sqlText)
		} else {
			// 原始SQL不为空，直接使用原始SQL
			s.log.Infof("Result[%d]: Using original SQL: %s", i, sqlText)
		}

		// 最终确保sqlText不为空
		if sqlText == "" {
			s.log.Errorf("Result[%d]: SQLText is still empty after all attempts! Original: %s, GoInception: %s", i, originalSQL, r.SQLText)
		}

		// 确保状态转换生效
		statusText := statusInfo.Text
		// 如果转换后的文本和原始状态相同，说明转换失败，使用默认处理
		if statusText == r.StageStatus {
			s.log.Warnf("Result[%d]: Status conversion failed, original=%s, trying again", i, r.StageStatus)
			// 再次尝试转换
			statusInfo = biz.GetGoInceptionStatusText(r.StageStatus)
			statusText = statusInfo.Text
		}
		// 如果还是相同，说明状态映射中没有这个值，使用原始状态
		if statusText == r.StageStatus {
			s.log.Warnf("Result[%d]: Status not in mapping, using original: %s", i, r.StageStatus)
		}

		s.log.Infof("Result[%d]: Final SQLText=%s, StatusText=%s (original=%s)", i, sqlText, statusText, r.StageStatus)

		protoResults[i] = &v1.SQLReviewRecord{
			Id:           0, // 审核记录ID，直接审核时不需要
			OrderId:      0, // 工单ID，直接审核时不需要
			Sequence:     sequence,
			SqlText:      sqlText,
			ErrorLevel:   int32(r.ErrorLevel),
			ErrorMessage: r.ErrorMessage,
			StageStatus:  statusText, // 使用转换后的中文状态
			AffectedRows: int32(r.AffectedRows),
		}
	}

	return &v1.ReviewSQLResponse{
		Code:    200,
		Message: "success",
		Data:    protoResults,
	}, nil
}

// GetOrderRecords 获取执行记录（实现 Protobuf 接口）
func (s *SQLOrderService) GetOrderRecords(ctx context.Context, req *v1.GetOrderRecordsRequest) (*v1.GetOrderRecordsResponse, error) {
	records, err := s.sqlOrderUsecase.GetReviewRecords(ctx, req.Id)
	if err != nil {
		return &v1.GetOrderRecordsResponse{
			Code:    500,
			Message: fmt.Sprintf("获取执行记录失败: %v", err),
		}, nil
	}

	protoRecords := make([]*v1.SQLReviewRecord, len(records))
	for i, record := range records {
		protoRecords[i] = toProtoSQLReviewRecord(record)
	}

	return &v1.GetOrderRecordsResponse{
		Code:    200,
		Message: "success",
		Data:    protoRecords,
	}, nil
}

// GetOrderItems 获取工单明细（实现 Protobuf 接口）
func (s *SQLOrderService) GetOrderItems(ctx context.Context, req *v1.GetOrderItemsRequest) (*v1.GetOrderItemsResponse, error) {
	items, err := s.sqlOrderUsecase.GetOrderItems(ctx, req.Id)
	if err != nil {
		return &v1.GetOrderItemsResponse{
			Code:    500,
			Message: fmt.Sprintf("获取工单明细失败: %v", err),
		}, nil
	}

	protoItems := make([]*v1.SQLOrderItem, len(items))
	for i, item := range items {
		protoItems[i] = toProtoSQLOrderItem(item)
	}

	return &v1.GetOrderItemsResponse{
		Code:    200,
		Message: "success",
		Data:    protoItems,
	}, nil
}

// GetSQLOrderRollbackSQLs 获取回滚SQL列表（实现 Protobuf 接口）
func (s *SQLOrderService) GetSQLOrderRollbackSQLs(ctx context.Context, req *v1.GetSQLOrderRollbackSQLsRequest) (*v1.GetSQLOrderRollbackSQLsResponse, error) {
	if s.rollbackUsecase == nil {
		return &v1.GetSQLOrderRollbackSQLsResponse{
			Code:    500,
			Message: "回滚功能未启用",
		}, nil
	}

	rollbacks, err := s.rollbackUsecase.GetRollbackSQLs(ctx, req.Id)
	if err != nil {
		return &v1.GetSQLOrderRollbackSQLsResponse{
			Code:    500,
			Message: fmt.Sprintf("获取回滚SQL失败: %v", err),
		}, nil
	}

	protoRollbacks := make([]*v1.SQLOrderRollbackSQL, len(rollbacks))
	for i, rb := range rollbacks {
		protoRollbacks[i] = &v1.SQLOrderRollbackSQL{
			Id:           rb.ID,
			OrderId:      rb.OrderID,
			BackupDbName: rb.BackupDBName,
			OpidTime:     rb.OpidTime,
			TableName:    rb.TableName,
			RollbackSql:  rb.RollbackSQL,
			CreatedAt:    rb.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return &v1.GetSQLOrderRollbackSQLsResponse{
		Code:    200,
		Message: "success",
		Data:    protoRollbacks,
	}, nil
}
