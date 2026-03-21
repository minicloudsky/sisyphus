package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"

	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
	asynqPkg "sisyphus/server/internal/pkg/asynq"
	"sisyphus/server/internal/middleware"
)

// BinlogService Binlog回滚服务
type BinlogService struct {
	binlogUsecase   *biz.BinlogRollbackUsecase
	instanceUsecase *biz.DatabaseInstanceUsecase
	asynqClient     *asynqPkg.Client
	log             *log.Helper
}

// NewBinlogService 创建Binlog回滚服务
func NewBinlogService(
	binlogUsecase *biz.BinlogRollbackUsecase,
	instanceUsecase *biz.DatabaseInstanceUsecase,
	asynqClient *asynqPkg.Client,
	logger log.Logger,
) *BinlogService {
	return &BinlogService{
		binlogUsecase:   binlogUsecase,
		instanceUsecase: instanceUsecase,
		asynqClient:     asynqClient,
		log:             log.NewHelper(logger),
	}
}

// ========== Protobuf 接口实现 ==========

// ParseBinlog 解析Binlog（实现 Protobuf 接口）
func (s *BinlogService) ParseBinlog(ctx context.Context, req *v1.ParseBinlogRequest) (*v1.ParseBinlogResponse, error) {
	claims, ok := middleware.FromContext(ctx)
	if !ok {
		return &v1.ParseBinlogResponse{Code: 401, Message: "未认证"}, nil
	}

	startTime := req.StartTime.AsTime()
	stopTime := req.StopTime.AsTime()

	// 时间范围不能超过 24 小时
	if stopTime.Sub(startTime) > 24*time.Hour {
		return &v1.ParseBinlogResponse{Code: 400, Message: "时间范围不能超过 24 小时"}, nil
	}
	if stopTime.Before(startTime) {
		return &v1.ParseBinlogResponse{Code: 400, Message: "结束时间不能早于开始时间"}, nil
	}

	order := &biz.BinlogRollbackOrder{
		InstanceID:   req.InstanceId,
		DatabaseName: req.DatabaseName,
		StartTime:    startTime,
		StopTime:     stopTime,
		BinlogFile:   req.BinlogFile,
		TableList:    req.TableList,
		SubmitterID:  claims.UserID,
	}

	order, err := s.binlogUsecase.CreateRollbackOrder(ctx, order)
	if err != nil {
		return &v1.ParseBinlogResponse{Code: 500, Message: fmt.Sprintf("创建回滚工单失败: %v", err)}, nil
	}

	// 使用Asynq入队任务
	payload := biz.BinlogParsePayload{
		OrderID:    order.ID,
		InstanceID: req.InstanceId,
		Database:   req.DatabaseName,
		Tables:     req.TableList,
		StartTime:  startTime.Format("2006-01-02 15:04:05"),
		EndTime:    stopTime.Format("2006-01-02 15:04:05"),
		CreatedBy:  claims.UserID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return &v1.ParseBinlogResponse{Code: 500, Message: fmt.Sprintf("序列化任务负载失败: %v", err)}, nil
	}

	opts := []asynq.Option{
		asynq.Queue("default"),
		asynq.TaskID(fmt.Sprintf("binlog:%d", order.ID)),
	}

	taskInfo, err := s.asynqClient.Enqueue(asynqPkg.TaskTypeBinlogParse, payloadBytes, opts...)
	if err != nil {
		return &v1.ParseBinlogResponse{Code: 500, Message: fmt.Sprintf("入队任务失败: %v", err)}, nil
	}

	return &v1.ParseBinlogResponse{
		Code: 200, Message: "success",
		Data: &v1.ParseBinlogData{
			OrderId: order.ID,
			WorkId:  order.WorkID,
			Message: fmt.Sprintf("Binlog解析任务已发起，任务ID: %s", taskInfo.ID),
		},
	}, nil
}


// ListBinlogOrders 获取回滚工单列表（实现 Protobuf 接口）
func (s *BinlogService) ListBinlogOrders(ctx context.Context, req *v1.ListBinlogOrdersRequest) (*v1.ListBinlogOrdersResponse, error) {
	pageReq := req.GetPageRequest()
	page := int(pageReq.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(pageReq.GetPageSize())
	if pageSize < 1 {
		pageSize = 10
	}

	orders, total, err := s.binlogUsecase.ListOrders(ctx, page, pageSize)
	if err != nil {
		return &v1.ListBinlogOrdersResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	protoOrders := make([]*v1.BinlogRollbackOrder, len(orders))
	for i, order := range orders {
		protoOrders[i] = toProtoBinlogOrder(order)
	}

	return &v1.ListBinlogOrdersResponse{
		Code: 200, Message: "success",
		List: protoOrders,
		PageResponse: &v1.PageResponse{Page: int32(page), PageSize: int32(pageSize), Total: total},
	}, nil
}

// GetBinlogOrder 获取回滚工单详情（实现 Protobuf 接口）
func (s *BinlogService) GetBinlogOrder(ctx context.Context, req *v1.GetBinlogOrderRequest) (*v1.GetBinlogOrderResponse, error) {
	order, err := s.binlogUsecase.GetOrderByID(ctx, req.Id)
	if err != nil {
		return &v1.GetBinlogOrderResponse{Code: 404, Message: "工单不存在"}, nil
	}

	return &v1.GetBinlogOrderResponse{
		Code: 200, Message: "success",
		Data: toProtoBinlogOrder(order),
	}, nil
}

// GetRollbackSQLs 获取回滚SQL列表（实现 Protobuf 接口）
func (s *BinlogService) GetRollbackSQLs(ctx context.Context, req *v1.GetRollbackSQLsRequest) (*v1.GetRollbackSQLsResponse, error) {
	sqls, err := s.binlogUsecase.GetRollbackSQLsByOrderID(ctx, req.Id)
	if err != nil {
		return &v1.GetRollbackSQLsResponse{Code: 500, Message: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	protoSQLs := make([]*v1.RollbackSQL, len(sqls))
	for i, sql := range sqls {
		protoSQLs[i] = toProtoRollbackSQL(sql)
	}

	return &v1.GetRollbackSQLsResponse{Code: 200, Message: "success", List: protoSQLs}, nil
}
