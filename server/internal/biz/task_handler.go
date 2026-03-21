package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

// SQLOrderExecutePayload SQL工单执行任务负载
type SQLOrderExecutePayload struct {
	OrderID       uint64 `json:"order_id"`
	ExecutorID    uint64 `json:"executor_id"`
	ExecuteMethod int    `json:"execute_method"` // 0=自动选择, 1=goInception, 2=gh-ost
}

// BinlogParsePayload Binlog解析任务负载
type BinlogParsePayload struct {
	OrderID    uint64   `json:"order_id"`
	InstanceID uint64   `json:"instance_id"`
	Database   string   `json:"database"`
	Tables     []string `json:"tables"`
	StartTime  string   `json:"start_time"`
	EndTime    string   `json:"end_time"`
	CreatedBy  uint64   `json:"created_by"`
}

// RedisRDBParsePayload Redis RDB解析任务负载
type RedisRDBParsePayload struct {
	TaskID     uint64 `json:"task_id"`
	InstanceID uint64 `json:"instance_id"`
	CreatedBy  uint64 `json:"created_by"`
}

// ArchiveExecutePayload 数据归档任务负载
type ArchiveExecutePayload struct {
	ConfigID uint64 `json:"config_id"`
	WorkID   string `json:"work_id"`
	Command  string `json:"command"`
}

// CKCheckConfig ckcheck测试实例配置
type CKCheckConfig struct {
	TestHost     string
	TestPort     int
	TestUsername string
	TestPassword string
}

// TaskHandler 任务处理器
type TaskHandler struct {
	sqlOrderUsecase      *SQLOrderUsecase
	sqlExecutor          *SQLExecutor
	instanceUsecase      *DatabaseInstanceUsecase
	executorSelector     *SQLExecutorSelector
	inceptionConfig      *InceptionConfig
	ckCheckConfig        *CKCheckConfig
	ckCheckExecutor      *CKCheckExecutor
	notificationService  *NotificationService
	rollbackUsecase      *RollbackUsecase
	binlogUsecase        *BinlogRollbackUsecase
	redisAnalysisUsecase *RedisAnalysisUsecase
	archiveUsecase       *ArchiveUsecase
	log                  *log.Helper
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(
	sqlOrderUsecase *SQLOrderUsecase,
	sqlExecutor *SQLExecutor,
	instanceUsecase *DatabaseInstanceUsecase,
	executorSelector *SQLExecutorSelector,
	inceptionConfig *InceptionConfig,
	ckCheckConfig *CKCheckConfig,
	notificationService *NotificationService,
	rollbackUsecase *RollbackUsecase,
	binlogUsecase *BinlogRollbackUsecase,
	redisAnalysisUsecase *RedisAnalysisUsecase,
	archiveUsecase *ArchiveUsecase,
	logger log.Logger,
) *TaskHandler {
	return &TaskHandler{
		sqlOrderUsecase:      sqlOrderUsecase,
		sqlExecutor:          sqlExecutor,
		instanceUsecase:      instanceUsecase,
		executorSelector:     executorSelector,
		inceptionConfig:      inceptionConfig,
		ckCheckConfig:        ckCheckConfig,
		ckCheckExecutor:      NewCKCheckExecutor(),
		notificationService:  notificationService,
		rollbackUsecase:      rollbackUsecase,
		binlogUsecase:        binlogUsecase,
		redisAnalysisUsecase: redisAnalysisUsecase,
		archiveUsecase:       archiveUsecase,
		log:                  log.NewHelper(logger),
	}
}

// HandleSQLOrderExecute 处理SQL工单执行任务
func (h *TaskHandler) HandleSQLOrderExecute(ctx context.Context, t *asynq.Task) error {
	var payload SQLOrderExecutePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("解析任务负载失败: %w", err)
	}

	h.log.Infof("开始执行SQL工单: order_id=%d, executor_id=%d, method=%d",
		payload.OrderID, payload.ExecutorID, payload.ExecuteMethod)

	// 获取工单
	order, err := h.sqlOrderUsecase.GetOrder(ctx, payload.OrderID)
	if err != nil {
		return fmt.Errorf("获取工单失败: %w", err)
	}

	// 更新工单状态为执行中
	order.Status = OrderStatusExecuting
	order.ExecutorID = payload.ExecutorID
	now := time.Now()
	order.ExecuteTime = &now
	if err := h.sqlOrderUsecase.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("更新工单状态失败: %w", err)
	}

	// 获取实例信息
	instance, err := h.instanceUsecase.GetInstanceByID(ctx, order.InstanceID)
	if err != nil {
		order.Status = OrderStatusFailed
		h.sqlOrderUsecase.UpdateOrder(ctx, order)
		return fmt.Errorf("获取实例信息失败: %w", err)
	}

	// 根据执行方式执行
	// 优先使用数据库中的execute_method，如果为0或payload中指定了，则使用payload中的
	executeMethod := int(order.ExecuteMethod)
	if executeMethod == 0 {
		// 如果数据库中的execute_method为0，使用payload中的
		executeMethod = payload.ExecuteMethod
	}
	if executeMethod == 0 {
		// 自动选择执行方式
		method, err := h.executorSelector.SelectExecuteMethod(ctx, order.SQLContent)
		if err != nil {
			h.log.Warnf("自动选择执行方式失败，使用默认goInception: order_id=%d, error=%v", order.ID, err)
			method = ExecuteMethodGoInception
		}
		executeMethod = method
		order.ExecuteMethod = executeMethod
		h.sqlOrderUsecase.UpdateOrder(ctx, order)
	}

	var execErr error
	h.log.Infof("执行方式: order_id=%d, execute_method=%d, ExecuteMethodGhOst=%d, ExecuteMethodGoInception=%d, ExecuteMethodCKCheck=%d",
		order.ID, executeMethod, ExecuteMethodGhOst, ExecuteMethodGoInception, ExecuteMethodCKCheck)
	if executeMethod == ExecuteMethodGhOst {
		// 使用gh-ost执行
		tableName := extractTableName(order.SQLContent)
		if tableName == "" {
			order.Status = OrderStatusFailed
			h.sqlOrderUsecase.UpdateOrder(ctx, order)
			return fmt.Errorf("无法从SQL中提取表名")
		}
		deleteGho := false // TODO: 从配置获取
		execErr = h.sqlExecutor.ExecuteWithGhOst(ctx, instance, order.SQLContent, order.DatabaseName, tableName, order.WorkID, deleteGho)
	} else if executeMethod == ExecuteMethodCKCheck {
		// 使用ckcheck执行ClickHouse SQL
		execErr = h.executeWithCKCheck(ctx, instance, order)
	} else {
		// 使用goInception执行（包括ExecuteMethodGoInception和ExecuteMethodNative）
		instanceInfo := &InstanceInfo{
			Host:     instance.Host,
			Port:     instance.Port,
			Username: instance.Username,
			Password: instance.Password,
		}
		execErr = h.executeWithGoInception(ctx, instanceInfo, order)
	}

	// 更新工单状态
	if execErr != nil {
		order.Status = OrderStatusFailed
		h.log.Errorf("工单执行失败: order_id=%d, error=%v", order.ID, execErr)
	} else {
		order.Status = OrderStatusSuccess
		h.log.Infof("工单执行成功: order_id=%d", order.ID)
	}

	if err := h.sqlOrderUsecase.UpdateOrder(ctx, order); err != nil {
		h.log.Errorf("更新工单状态失败: order_id=%d, error=%v", order.ID, err)
	}

	// 发送通知
	if h.notificationService != nil {
		// TODO: 发送通知
	}

	return execErr
}

// executeWithGoInception 使用goInception执行
func (h *TaskHandler) executeWithGoInception(ctx context.Context, instance *InstanceInfo, order *SQLOrder) error {
	client := NewGoInceptionClient(
		h.inceptionConfig.Host,
		int(h.inceptionConfig.Port),
		instance.Host,
		instance.Port,
		instance.Username,
		instance.Password,
	)

	// 如果配置了goInception的用户名和密码，使用配置的值
	if h.inceptionConfig.Username != "" {
		client.SetInceptionCredentials(h.inceptionConfig.Username, h.inceptionConfig.Password)
	}

	// 使用SQLExecutor执行
	results, err := h.sqlExecutor.ExecuteWithGoInception(ctx, client, order.SQLContent, order.DatabaseName, order.NeedBackup)
	if err != nil {
		return fmt.Errorf("goInception执行失败: %w", err)
	}

	// 处理执行结果
	hasError := false

	h.log.Infof("开始处理执行结果: order_id=%d, results_count=%d", order.ID, len(results))

	// 按分号拆分原始SQL，用于匹配执行结果
	// 注意：goInception会为每条SQL返回一个结果，包括注释行
	allSqlParts := strings.Split(order.SQLContent, ";")
	sqlParts := []string{} // 实际的SQL语句（排除注释和空行）
	for _, part := range allSqlParts {
		trimmed := strings.TrimSpace(part)
		trimmedLower := strings.ToLower(trimmed)

		// 排除空行、纯注释行和 use database 语句
		// 注意：如果一行包含注释+SQL（如 "-- comment\nINSERT INTO..."），需要提取SQL部分
		if trimmed == "" {
			continue
		}

		// 排除纯注释行（整行都是注释）
		if strings.HasPrefix(trimmedLower, "--") && !strings.Contains(trimmed, "\n") {
			// 纯注释行（没有换行符），跳过
			continue
		}

		// 排除多行注释
		if strings.HasPrefix(trimmedLower, "/*") {
			continue
		}

		// 排除 use database 语句
		if trimmedLower == "use `"+strings.ToLower(order.DatabaseName)+"`" {
			continue
		}

		// 如果包含注释+SQL（如 "-- comment\nINSERT INTO..."），提取SQL部分
		if strings.Contains(trimmed, "\n") {
			// 按换行符拆分，找到第一个非注释行
			lines := strings.Split(trimmed, "\n")
			for _, line := range lines {
				lineTrimmed := strings.TrimSpace(line)
				lineLower := strings.ToLower(lineTrimmed)
				// 跳过空行和注释行
				if lineTrimmed != "" &&
					!strings.HasPrefix(lineLower, "--") &&
					!strings.HasPrefix(lineLower, "/*") {
					// 找到实际的SQL语句
					sqlParts = append(sqlParts, lineTrimmed)
					break // 只取第一个非注释行
				}
			}
		} else {
			// 单行SQL，直接添加
			sqlParts = append(sqlParts, trimmed)
		}
	}

	h.log.Infof("原始SQL拆分结果: order_id=%d, sql_parts_count=%d", order.ID, len(sqlParts))

	// 处理goInception返回的结果
	// 注意：goInception返回的结果中，第一条通常是 use database 或注释
	// 我们需要跳过这些，然后按顺序匹配实际的SQL
	sequence := 1
	resultIndex := 0 // goInception返回结果的索引
	sqlIndex := 0    // 实际SQL的索引

	// 遍历所有结果，处理有效的执行结果
	for resultIndex < len(results) {
		result := results[resultIndex]
		sqlTextFromResult := strings.TrimSpace(result.SQLText)

		// 判断是否跳过：use database、空结果
		// 注意：即使SQL文本是注释，如果affected_rows>0，说明实际执行了SQL，不应该跳过
		isSkip := false
		if sqlTextFromResult == "use `"+order.DatabaseName+"`" {
			// use database语句，跳过
			isSkip = true
		} else if sqlTextFromResult == "" && result.AffectedRows == 0 {
			// 空结果且无影响行数，跳过
			isSkip = true
		}
		// 注意：即使SQL文本是注释（如 "-- 1. 进行中的归档任务"），如果affected_rows>0，说明实际执行了SQL，不应该跳过

		if isSkip {
			resultIndex++
			continue
		}

		// 获取SQL文本：goInception返回的SQL文本可能是注释，需要从原始SQL中提取
		// 关键：goInception返回的结果中，SQL文本字段可能是注释，而不是实际的SQL语句
		// 我们需要从原始SQL中按顺序提取实际的SQL语句
		// 重要：每个result对应一条SQL，必须从sqlParts中按顺序提取，即使SQL文本是注释
		var sqlText string
		// 关键修复：如果sqlTextFromResult是注释（以"--"开头），必须从sqlParts中提取对应的SQL
		// 因为goInception返回的SQL文本可能是注释，而不是实际的SQL语句
		if strings.HasPrefix(sqlTextFromResult, "--") || strings.HasPrefix(sqlTextFromResult, "/*") {
			// goInception返回的是注释，必须从sqlParts中提取对应的SQL
			if sqlIndex < len(sqlParts) {
				sqlText = sqlParts[sqlIndex]
				sqlIndex++ // 立即递增，确保下一条SQL能正确匹配
			} else if len(sqlParts) > 0 {
				// 如果索引超出，使用最后一条SQL
				sqlText = sqlParts[len(sqlParts)-1]
			} else {
				// 如果都没有，使用原始SQL
				sqlText = order.SQLContent
			}
		} else if sqlIndex < len(sqlParts) {
			// goInception返回的是实际SQL，但也要从sqlParts中提取以确保一致性
			sqlText = sqlParts[sqlIndex]
			sqlIndex++ // 立即递增，确保下一条SQL能正确匹配
		} else if len(sqlParts) > 0 {
			// 如果索引超出，使用最后一条SQL
			sqlText = sqlParts[len(sqlParts)-1]
		} else {
			// 如果都没有，使用原始SQL
			sqlText = order.SQLContent
		}
		// 记录执行结果
		h.log.Infof("执行结果: order_id=%d, sequence=%d, result_index=%d, sql_index=%d, success=%v, affected_rows=%d, error=%s, sql_from_result=%s",
			order.ID, sequence, resultIndex, sqlIndex, result.Success, result.AffectedRows, result.ErrorMessage, sqlTextFromResult)

		// 记录最终使用的SQL文本
		h.log.Infof("SQL文本: order_id=%d, sequence=%d, sql=%s", order.ID, sequence, sqlText)

		// 确定执行状态：0-待执行, 1-执行成功, 2-执行失败
		itemStatus := 0
		if result.Success {
			itemStatus = 1 // 执行成功
		} else {
			itemStatus = 2 // 执行失败
			hasError = true
			h.log.Errorf("SQL执行失败: order_id=%d, sequence=%d, sql=%s, error=%s",
				order.ID, sequence, sqlText, result.ErrorMessage)
		}

		// 解析执行耗时（秒）
		executeTime := 0.0
		if result.ExecuteTime != "" {
			// 尝试解析执行耗时（格式可能是 "0.123s" 或 "123ms"）
			timeStr := strings.TrimSpace(result.ExecuteTime)
			if strings.HasSuffix(timeStr, "s") {
				var timeVal float64
				if _, err := fmt.Sscanf(strings.TrimSuffix(timeStr, "s"), "%f", &timeVal); err == nil {
					executeTime = timeVal
				}
			} else if strings.HasSuffix(timeStr, "ms") {
				var timeVal float64
				if _, err := fmt.Sscanf(strings.TrimSuffix(timeStr, "ms"), "%f", &timeVal); err == nil {
					executeTime = timeVal / 1000.0
				}
			} else {
				// 尝试直接解析为数字
				var timeVal float64
				if _, err := fmt.Sscanf(timeStr, "%f", &timeVal); err == nil {
					executeTime = timeVal
				}
			}
		}

		// 保存执行明细到数据库
		// 先检查是否已存在相同sequence的记录，如果存在则删除（避免重复）
		existingItems, _ := h.sqlOrderUsecase.GetItemsByOrderID(ctx, order.ID)
		for _, existing := range existingItems {
			if existing.Sequence == sequence {
				// 删除已存在的记录，避免重复
				if err := h.sqlOrderUsecase.DeleteItem(ctx, existing.ID); err != nil {
					h.log.Warnf("删除已存在的执行明细失败: order_id=%d, sequence=%d, item_id=%d, error=%v", order.ID, sequence, existing.ID, err)
				}
			}
		}

		item := &SQLOrderItem{
			OrderID:      order.ID,
			Sequence:     sequence,
			SQLText:      sqlText,
			Status:       itemStatus,
			ExecuteTime:  executeTime,
			AffectedRows: result.AffectedRows,
			ErrorMessage: result.ErrorMessage,
		}
		if err := h.sqlOrderUsecase.CreateItem(ctx, item); err != nil {
			h.log.Warnf("保存执行明细失败: order_id=%d, sequence=%d, error=%v", order.ID, sequence, err)
		} else {
			h.log.Infof("保存执行明细成功: order_id=%d, sequence=%d, status=%d", order.ID, sequence, itemStatus)
		}

		// 如果执行成功且有备份，保存回滚SQL
		if result.Success && h.rollbackUsecase != nil && result.BackupDBName != "" && result.Sequence != "" {
			// 使用GenerateAndSaveRollbackSQL方法
			if err := h.rollbackUsecase.GenerateAndSaveRollbackSQL(ctx, order.ID, nil, result.BackupDBName, result.Sequence); err != nil {
				h.log.Warnf("保存回滚SQL失败: order_id=%d, error=%v", order.ID, err)
			}
		}

		sequence++
		resultIndex++
	}

	// 如果还有未处理的SQL，说明goInception返回的结果不够
	if sqlIndex < len(sqlParts) {
		h.log.Warnf("警告: order_id=%d, 还有 %d 条SQL未处理，results_count=%d, sql_parts_count=%d",
			order.ID, len(sqlParts)-sqlIndex, len(results), len(sqlParts))
	}

	// 如果一条执行明细都没保存，说明有问题
	if sequence == 1 {
		h.log.Errorf("错误: order_id=%d, 没有保存任何执行明细，results_count=%d, sql_parts_count=%d",
			order.ID, len(results), len(sqlParts))
		// 至少保存一条执行明细，使用原始SQL
		item := &SQLOrderItem{
			OrderID:      order.ID,
			Sequence:     1,
			SQLText:      order.SQLContent,
			Status:       2, // 标记为失败，因为无法确定执行结果
			ExecuteTime:  0.0,
			AffectedRows: 0,
			ErrorMessage: "无法解析执行结果，请查看goInception日志",
		}
		if err := h.sqlOrderUsecase.CreateItem(ctx, item); err != nil {
			h.log.Errorf("保存默认执行明细失败: order_id=%d, error=%v", order.ID, err)
		}
	}

	// 如果有错误，返回错误
	if hasError {
		return fmt.Errorf("部分SQL执行失败，请查看执行结果")
	}

	return nil
}

// HandleBinlogParse 处理Binlog解析任务
func (h *TaskHandler) HandleBinlogParse(ctx context.Context, t *asynq.Task) error {
	var payload BinlogParsePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("解析任务负载失败: %w", err)
	}

	h.log.Infof("开始解析Binlog: order_id=%d, instance_id=%d", payload.OrderID, payload.InstanceID)

	// 获取工单
	order, err := h.binlogUsecase.GetOrderByID(ctx, payload.OrderID)
	if err != nil {
		return fmt.Errorf("获取Binlog工单失败: %w", err)
	}

	// 更新工单状态为执行中
	order.Status = 1 // 执行中
	if err := h.binlogUsecase.UpdateOrder(ctx, order); err != nil {
		h.log.Warnf("更新Binlog工单状态失败: order_id=%d, error=%v", payload.OrderID, err)
	}

	// 获取实例信息
	instance, err := h.instanceUsecase.GetInstanceByID(ctx, payload.InstanceID)
	if err != nil {
		order.Status = 2 // 失败
		h.binlogUsecase.UpdateOrder(ctx, order)
		return fmt.Errorf("获取实例信息失败: %w", err)
	}

	instanceInfo := &InstanceInfo{
		Host:     instance.Host,
		Port:     instance.Port,
		Username: instance.Username,
		Password: instance.Password,
	}

	// 解析时间
	startTime, err := time.Parse("2006-01-02 15:04:05", payload.StartTime)
	if err != nil {
		order.Status = 2
		h.binlogUsecase.UpdateOrder(ctx, order)
		return fmt.Errorf("解析开始时间失败: %w", err)
	}
	endTime, err := time.Parse("2006-01-02 15:04:05", payload.EndTime)
	if err != nil {
		order.Status = 2
		h.binlogUsecase.UpdateOrder(ctx, order)
		return fmt.Errorf("解析结束时间失败: %w", err)
	}

	// 执行Binlog解析
	err = h.binlogUsecase.ParseBinlog(ctx, payload.OrderID, instanceInfo, order.BinlogFile, startTime, endTime, payload.Tables)
	if err != nil {
		order.Status = 2 // 失败
		h.binlogUsecase.UpdateOrder(ctx, order)
		h.log.Errorf("Binlog解析失败: order_id=%d, error=%v", payload.OrderID, err)
		return fmt.Errorf("Binlog解析失败: %w", err)
	}

	h.log.Infof("Binlog解析成功: order_id=%d", payload.OrderID)
	return nil
}

// HandleRedisRDBParse 处理Redis RDB解析任务
func (h *TaskHandler) HandleRedisRDBParse(ctx context.Context, t *asynq.Task) error {
	var payload RedisRDBParsePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("解析任务负载失败: %w", err)
	}

	h.log.Infof("开始解析Redis RDB: task_id=%d, instance_id=%d", payload.TaskID, payload.InstanceID)

	// 获取任务
	task, err := h.redisAnalysisUsecase.GetTaskByID(ctx, payload.TaskID)
	if err != nil {
		return fmt.Errorf("获取Redis分析任务失败: %w", err)
	}

	// 更新任务状态为执行中
	task.Status = 1 // 执行中
	now := time.Now()
	task.StartedAt = &now
	if err := h.redisAnalysisUsecase.UpdateTask(ctx, task); err != nil {
		h.log.Warnf("更新Redis分析任务状态失败: task_id=%d, error=%v", payload.TaskID, err)
	}

	// 获取实例信息
	instance, err := h.instanceUsecase.GetInstanceByID(ctx, payload.InstanceID)
	if err != nil {
		task.Status = 3 // 失败
		h.redisAnalysisUsecase.UpdateTask(ctx, task)
		return fmt.Errorf("获取实例信息失败: %w", err)
	}

	instanceInfo := &InstanceInfo{
		Host:     instance.Host,
		Port:     instance.Port,
		Username: instance.Username,
		Password: instance.Password,
	}

	// 根据任务类型执行分析
	var result map[string]interface{}
	if task.TaskType == "hotkey" {
		// 分析热key
		hotKeys, err := h.redisAnalysisUsecase.AnalyzeHotKeys(ctx, instanceInfo, 60, 100)
		if err != nil {
			task.Status = 3
			h.redisAnalysisUsecase.UpdateTask(ctx, task)
			return fmt.Errorf("热key分析失败: %w", err)
		}
		result = map[string]interface{}{
			"hotkeys": hotKeys,
			"count":   len(hotKeys),
		}
	} else if task.TaskType == "bigkey" {
		// 分析大key
		bigKeys, err := h.redisAnalysisUsecase.AnalyzeBigKeys(ctx, instanceInfo, 1024*1024) // 1MB阈值
		if err != nil {
			task.Status = 3
			h.redisAnalysisUsecase.UpdateTask(ctx, task)
			return fmt.Errorf("大key分析失败: %w", err)
		}
		result = map[string]interface{}{
			"bigkeys": bigKeys,
			"count":   len(bigKeys),
		}
	} else {
		task.Status = 3
		h.redisAnalysisUsecase.UpdateTask(ctx, task)
		return fmt.Errorf("未知的任务类型: %s", task.TaskType)
	}

	// 更新任务状态为完成
	task.Status = 2 // 完成
	task.Result = result
	finished := time.Now()
	task.FinishedAt = &finished
	if err := h.redisAnalysisUsecase.UpdateTask(ctx, task); err != nil {
		h.log.Warnf("更新Redis分析任务状态失败: task_id=%d, error=%v", payload.TaskID, err)
	}

	h.log.Infof("Redis RDB解析成功: task_id=%d", payload.TaskID)
	return nil
}

// HandleArchiveExecute 处理数据归档任务
func (h *TaskHandler) HandleArchiveExecute(ctx context.Context, t *asynq.Task) error {
	var payload ArchiveExecutePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("解析任务负载失败: %w", err)
	}

	h.log.Infof("开始执行数据归档: config_id=%d, work_id=%s", payload.ConfigID, payload.WorkID)

	// 获取归档配置
	config, err := h.archiveUsecase.GetConfigByID(ctx, payload.ConfigID)
	if err != nil {
		return fmt.Errorf("获取归档配置失败: %w", err)
	}

	if !config.IsEnabled {
		return fmt.Errorf("归档配置未启用: config_id=%d", payload.ConfigID)
	}

	// 获取或创建归档记录
	var record *ArchiveRecord
	if payload.WorkID != "" {
		// 尝试通过workID查找记录
		records, _, _ := h.archiveUsecase.ListRecords(ctx, payload.ConfigID, 1, 1)
		for _, r := range records {
			if r.WorkID == payload.WorkID {
				record = r
				break
			}
		}
	}

	if record == nil {
		// 创建新的归档记录
		record = &ArchiveRecord{
			ConfigID: payload.ConfigID,
			WorkID:   payload.WorkID,
			Status:   1, // 执行中
		}
		now := time.Now()
		record.ExecuteTime = &now
		record, err = h.archiveUsecase.CreateRecord(ctx, record)
		if err != nil {
			return fmt.Errorf("创建归档记录失败: %w", err)
		}
	} else {
		// 更新记录状态为执行中
		record.Status = 1
		now := time.Now()
		record.ExecuteTime = &now
		if err := h.archiveUsecase.UpdateRecord(ctx, record); err != nil {
			h.log.Warnf("更新归档记录失败: record_id=%d, error=%v", record.ID, err)
		}
	}

	// 执行pt-archiver命令
	execCmd := exec.CommandContext(ctx, "sh", "-c", payload.Command)
	output, err := execCmd.CombinedOutput()
	if err != nil {
		record.Status = 3 // 失败
		record.ErrorMessage = fmt.Sprintf("pt-archiver执行失败: %v, 输出: %s", err, string(output))
		finished := time.Now()
		record.FinishedTime = &finished
		h.archiveUsecase.UpdateRecord(ctx, record)
		return fmt.Errorf("pt-archiver执行失败: %w", err)
	}

	// 更新记录为成功
	record.Status = 2 // 成功
	finished := time.Now()
	record.FinishedTime = &finished
	// TODO: 解析输出获取归档数量
	h.log.Infof("pt-archiver输出: %s", string(output))

	if err := h.archiveUsecase.UpdateRecord(ctx, record); err != nil {
		h.log.Warnf("更新归档记录失败: record_id=%d, error=%v", record.ID, err)
	}

	h.log.Infof("数据归档执行成功: config_id=%d, work_id=%s", payload.ConfigID, payload.WorkID)
	return nil
}

// executeWithCKCheck 使用ckcheck方式执行ClickHouse SQL
func (h *TaskHandler) executeWithCKCheck(ctx context.Context, instance *DatabaseInstance, order *SQLOrder) error {
	if h.ckCheckConfig == nil || h.ckCheckConfig.TestHost == "" {
		return fmt.Errorf("未配置ClickHouse测试实例（ckcheck.test_host），无法审核执行")
	}

	testInstance := &DatabaseInstance{
		Host:     h.ckCheckConfig.TestHost,
		Port:     h.ckCheckConfig.TestPort,
		Username: h.ckCheckConfig.TestUsername,
		Password: h.ckCheckConfig.TestPassword,
	}

	results, err := h.ckCheckExecutor.AuditAndExecuteWithCKCheck(
		ctx,
		instance,
		testInstance,
		order.SQLContent,
		order.DatabaseName,
		false, // dryRun=false: 审核通过后执行生产
	)
	if err != nil {
		// 保存失败明细
		item := &SQLOrderItem{
			OrderID:      order.ID,
			Sequence:     1,
			SQLText:      order.SQLContent,
			Status:       2,
			ErrorMessage: err.Error(),
		}
		h.sqlOrderUsecase.CreateItem(ctx, item) //nolint:errcheck
		return err
	}

	hasError := false
	for i, result := range results {
		status := 1
		if result.ErrorLevel == 2 {
			status = 2
			hasError = true
		}
		item := &SQLOrderItem{
			OrderID:      order.ID,
			Sequence:     i + 1,
			SQLText:      result.SQL,
			Status:       status,
			ErrorMessage: result.ErrorMessage,
		}
		h.sqlOrderUsecase.CreateItem(ctx, item) //nolint:errcheck
	}

	if hasError {
		return fmt.Errorf("ClickHouse SQL执行存在错误，请查看执行明细")
	}
	return nil
}

// extractTableName 从SQL中提取表名（简单实现）
func extractTableName(sql string) string {
	// 这是一个简化实现，实际应该使用SQL解析器
	sql = strings.ToLower(strings.TrimSpace(sql))
	if strings.HasPrefix(sql, "alter table") {
		parts := strings.Fields(sql)
		if len(parts) >= 3 {
			tableName := parts[2]
			// 移除可能的反引号
			tableName = strings.Trim(tableName, "`")
			return tableName
		}
	}
	return ""
}
