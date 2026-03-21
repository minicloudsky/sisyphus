package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
)

// toProtoSQLOrder 转换业务实体到 Protobuf 类型
func toProtoSQLOrder(order *biz.SQLOrder) *v1.SQLOrder {
	protoOrder := &v1.SQLOrder{
		Id:           order.ID,
		WorkId:       order.WorkID,
		InstanceId:   order.InstanceID,
		DatabaseName: order.DatabaseName,
		SqlContent:   order.SQLContent,
		OrderType:    int32(order.OrderType),
		Status:       int32(order.Status),
		RiskLevel:    int32(order.RiskLevel),
		NeedBackup:   order.NeedBackup,
		Remark:       order.Remark,
		SubmitterId:  order.SubmitterID,
		AuditorId:    order.AuditorID,
		ExecutorId:   order.ExecutorID,
		RejectReason: order.RejectReason,
		ExecuteMethod: int32(order.ExecuteMethod),
		CreatedAt:    order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    order.UpdatedAt.Format(time.RFC3339),
	}

	return protoOrder
}

// toProtoSQLOrderItem 转换工单明细
func toProtoSQLOrderItem(item *biz.SQLOrderItem) *v1.SQLOrderItem {
	return &v1.SQLOrderItem{
		Id:           item.ID,
		OrderId:      item.OrderID,
		Sequence:     int32(item.Sequence),
		SqlText:      item.SQLText,
		Status:       int32(item.Status),
		ErrorMessage: item.ErrorMessage,
		AffectedRows: int32(item.AffectedRows),
		ExecuteTime:  fmt.Sprintf("%.3fs", item.ExecuteTime),
	}
}

// toProtoSQLReviewRecord 转换审核记录
func toProtoSQLReviewRecord(record *biz.SQLReviewRecord) *v1.SQLReviewRecord {
	sequence := int32(0)
	if record.Sequence != "" {
		// 尝试解析 Sequence 字符串为整数
		if seq, err := strconv.ParseInt(record.Sequence, 10, 32); err == nil {
			sequence = int32(seq)
		}
	}
	return &v1.SQLReviewRecord{
		Id:           record.ID,
		OrderId:      record.OrderID,
		Sequence:     sequence,
		SqlText:      record.SQLText,
		ErrorLevel:   int32(record.ErrorLevel),
		ErrorMessage: record.ErrorMessage,
		StageStatus:  record.StageStatus,
		AffectedRows: int32(record.AffectedRows),
		ExecuteTime:  record.ExecuteTime,
		SqlHash:      record.SQLHash,
		BackupDbName: record.BackupDBName,
	}
}

// toProtoDatabaseInstance 转换数据库实例
func toProtoDatabaseInstance(instance *biz.DatabaseInstance) *v1.DatabaseInstance {
	return &v1.DatabaseInstance{
		Id:          instance.ID,
		Name:        instance.Name,
		DbType:      instance.DBType,
		Host:        instance.Host,
		Port:        int32(instance.Port),
		Username:    instance.Username,
		Password:    instance.Password, // 注意：实际使用时应该隐藏
		Env:         instance.Env,
		Region:      instance.Region,
		Description: instance.Description,
		IsEnabled:   instance.Status == 1,
		CreatedAt:   timestamppb.New(instance.CreatedAt),
		UpdatedAt:   timestamppb.New(instance.UpdatedAt),
	}
}

// toProtoPermission 转换权限
func toProtoPermission(permission *biz.Permission) *v1.Permission {
	return &v1.Permission{
		Id:          permission.ID,
		Name:        permission.Name,
		Code:        permission.Code,
		Resource:    permission.Resource,
		Action:      permission.Action,
		Description: permission.Description,
	}
}

// toProtoPermissionWithTimestamps 转换权限（包含时间戳）
func toProtoPermissionWithTimestamps(permission *biz.Permission) *v1.Permission {
	protoPerm := toProtoPermission(permission)
	// Permission 结构体没有 CreatedAt/UpdatedAt 字段，使用零值
	// 如果需要时间戳，需要在 biz.Permission 中添加这些字段
	return protoPerm
}

// toProtoRole 转换角色
func toProtoRole(role *biz.Role) *v1.Role {
	protoRole := &v1.Role{
		Id:          role.ID,
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
	}
	return protoRole
}

// toProtoRoleWithTimestamps 转换角色（包含时间戳）
func toProtoRoleWithTimestamps(role *biz.Role) *v1.Role {
	protoRole := toProtoRole(role)
	// Role 结构体没有 CreatedAt/UpdatedAt 字段，使用零值
	// 如果需要时间戳，需要在 biz.Role 中添加这些字段
	return protoRole
}

// toBizSQLOrder 从 Protobuf 类型转换到业务实体（用于创建）
func toBizSQLOrder(req *v1.CreateOrderRequest, submitterID uint64) *biz.SQLOrder {
	return &biz.SQLOrder{
		InstanceID:   req.InstanceId,
		DatabaseName: req.DatabaseName,
		SQLContent:   req.SqlContent,
		OrderType:    int(req.OrderType),
		NeedBackup:   req.NeedBackup,
		Remark:       req.Remark,
		SubmitterID:  submitterID,
		ExecuteMethod: int(req.ExecuteMethod),
	}
}

// toBizDatabaseInstance 从 Protobuf 转换到业务实体
func toBizDatabaseInstance(req *v1.CreateInstanceRequest) *biz.DatabaseInstance {
	return &biz.DatabaseInstance{
		Name:        req.Name,
		DBType:      req.DbType,
		Host:        req.Host,
		Port:        int(req.Port),
		Username:    req.Username,
		Password:    req.Password,
		Env:         req.Env,
		Region:      req.Region,
		Description: req.Description,
		Status:      1,
	}
}

// toProtoBinlogOrder 转换Binlog回滚工单
func toProtoBinlogOrder(order *biz.BinlogRollbackOrder) *v1.BinlogRollbackOrder {
	return &v1.BinlogRollbackOrder{
		Id:          order.ID,
		WorkId:      order.WorkID,
		InstanceId:  order.InstanceID,
		DatabaseName: order.DatabaseName,
		StartTime:   timestamppb.New(order.StartTime),
		StopTime:    timestamppb.New(order.StopTime),
		BinlogFile:  order.BinlogFile,
		TableList:   order.TableList,
		Status:      int32(order.Status),
		SubmitterId: order.SubmitterID,
		CreatedAt:   timestamppb.New(order.CreatedAt),
		UpdatedAt:   timestamppb.New(order.UpdatedAt),
	}
}

// toProtoRollbackSQL 转换回滚SQL
func toProtoRollbackSQL(sql *biz.BinlogRollbackSQL) *v1.RollbackSQL {
	// 从 OriginalSQL 中提取 SQL 类型
	sqlType := "UNKNOWN"
	if strings.Contains(sql.OriginalSQL, "INSERT") {
		sqlType = "INSERT"
	} else if strings.Contains(sql.OriginalSQL, "UPDATE") {
		sqlType = "UPDATE"
	} else if strings.Contains(sql.OriginalSQL, "DELETE") {
		sqlType = "DELETE"
	}

	return &v1.RollbackSQL{
		Id:          sql.ID,
		OrderId:     sql.OrderID,
		TableName:   sql.TableName,
		SqlType:     sqlType,
		OriginalSql: sql.OriginalSQL,
		RollbackSql: sql.RollbackSQL,
		CreatedAt:   timestamppb.New(sql.CreatedAt),
	}
}

// toProtoRedisAnalysisTask 转换Redis分析任务
func toProtoRedisAnalysisTask(task *biz.RedisHotKeyTask) *v1.RedisAnalysisTask {
	protoTask := &v1.RedisAnalysisTask{
		Id:          task.ID,
		WorkId:      "", // RedisHotKeyTask 没有 WorkID，需要从 Result 中提取或生成
		InstanceId:  task.InstanceID,
		TaskType:    task.TaskType,
		Status:      int32(task.Status),
		Result:      "", // 需要将 map[string]interface{} 序列化为 JSON
		SubmitterId: task.SubmitterID,
		CreatedAt:   timestamppb.New(task.CreatedAt),
		UpdatedAt:   timestamppb.New(task.UpdatedAt),
	}
	
	// 序列化 Result
	if task.Result != nil {
		if resultJSON, err := json.Marshal(task.Result); err == nil {
			protoTask.Result = string(resultJSON)
		}
	}
	
	return protoTask
}

// toProtoArchiveConfig 转换归档配置
func toProtoArchiveConfig(config *biz.ArchiveConfig) *v1.ArchiveConfig {
	protoConfig := &v1.ArchiveConfig{
		Id:              config.ID,
		InstanceId:      config.InstanceID,
		SourceDatabase:  config.SourceDatabase,
		SourceTable:     config.SourceTable,
		DestDatabase:    config.DestDatabase,
		DestTable:       config.DestTable,
		ArchiveField:    config.ArchiveField,
		RetentionDays:   int32(config.RetentionDays),
		ArchiveCondition: config.ArchiveCondition,
		IsEnabled:       config.IsEnabled,
		CronExpression:  config.CronExpression,
		Owner:           config.Owner,
		CreatedAt:       timestamppb.New(config.CreatedAt),
		UpdatedAt:       timestamppb.New(config.UpdatedAt),
	}
	
	if config.DestInstanceID != nil {
		protoConfig.DestInstanceId = *config.DestInstanceID
	}
	
	return protoConfig
}

// toProtoArchiveRecord 转换归档记录
func toProtoArchiveRecord(record *biz.ArchiveRecord) *v1.ArchiveRecord {
	protoRecord := &v1.ArchiveRecord{
		Id:          record.ID,
		ConfigId:    record.ConfigID,
		WorkId:      record.WorkID,
		Status:      int32(record.Status),
		Message:     record.ErrorMessage,
		ArchivedRows: int64(record.ArchiveCount),
		CreatedAt:   timestamppb.New(record.CreatedAt),
	}
	
	if record.ExecuteTime != nil {
		protoRecord.StartTime = timestamppb.New(*record.ExecuteTime)
	}
	if record.FinishedTime != nil {
		protoRecord.EndTime = timestamppb.New(*record.FinishedTime)
	}
	
	return protoRecord
}

// toProtoGhOstProgress 转换gh-ost进度
func toProtoGhOstProgress(progress *biz.GhOstProgress, workID, tableName string) *v1.GhOstProgress {
	// 解析进度百分比
	percent := 0.0
	if progress.Percent > 0 {
		percent = float64(progress.Percent)
	}
	
	// 解析预计剩余时间（秒）
	estimatedSeconds := 0.0
	// 从 Consuming 字符串中提取时间（格式可能为 "1h2m3s"）
	// 这里简化处理，实际需要解析字符串
	
	return &v1.GhOstProgress{
		WorkId:                  workID,
		TableName:               tableName,
		TotalRows:               0, // 需要从日志中解析
		CopiedRows:              0, // 需要从日志中解析
		ProgressPercent:          percent,
		EstimatedSecondsRemaining: estimatedSeconds,
		Status:                  progress.Status,
		Message:                 progress.Backlog,
	}
}

// toProtoGhOstParams 转换gh-ost参数
func toProtoGhOstParams(params *biz.GhOstParams, workID string) *v1.GhOstParams {
	return &v1.GhOstParams{
		WorkId:        workID,
		ChunkSize:     int32(params.ChunkSize),
		MaxLagMillis:  int32(params.MaxLagMillis),
		MaxLoad:       int32(params.ThreadsRunning), // ThreadsRunning 用于 max-load
		CriticalLoad:  int32(params.ThreadsRunning), // ThreadsRunning 用于 critical-load
		NiceRatio:     params.NiceRatio,
	}
}

// toBizArchiveConfig 从 Protobuf 转换到业务实体
func toBizArchiveConfig(req *v1.CreateArchiveConfigRequest, owner string) *biz.ArchiveConfig {
	config := &biz.ArchiveConfig{
		InstanceID:      req.InstanceId,
		SourceDatabase:  req.SourceDatabase,
		SourceTable:     req.SourceTable,
		DestDatabase:    req.DestDatabase,
		DestTable:       req.DestTable,
		ArchiveField:    req.ArchiveField,
		RetentionDays:   int(req.RetentionDays),
		ArchiveCondition: req.ArchiveCondition,
		IsEnabled:       req.IsEnabled,
		CronExpression:  req.CronExpression,
		Owner:           owner,
	}
	
	if req.DestInstanceId > 0 {
		destInstanceID := req.DestInstanceId
		config.DestInstanceID = &destInstanceID
	}
	
	return config
}
