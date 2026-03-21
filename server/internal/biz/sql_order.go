package biz

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// SQLOrderType SQL工单类型
const (
	OrderTypeDDL    = 0 // DDL
	OrderTypeDML    = 1 // DML
	OrderTypeEXPORT = 2 // 导出
)

// SQLOrderStatus 工单状态
const (
	OrderStatusPending     = 0  // 待审核
	OrderStatusApproved    = 1  // 已通过
	OrderStatusRejected    = 2  // 已驳回
	OrderStatusExecuting   = 3  // 执行中
	OrderStatusSuccess     = 4  // 执行成功
	OrderStatusFailed      = 5  // 执行失败
	OrderStatusDoubleCheck = 10 // 等待复核
)

// ExecuteMethod 执行方式
const (
	ExecuteMethodNative      = 0 // 原生执行
	ExecuteMethodGhOst       = 1 // gh-ost
	ExecuteMethodGoInception = 2 // goInception
	ExecuteMethodCKCheck     = 3 // ckcheck（ClickHouse审核执行）
)

// SQLOrder SQL工单实体
type SQLOrder struct {
	ID            uint64
	WorkID        string
	InstanceID    uint64
	DatabaseName  string
	OrderType     int
	Status        int
	SQLContent    string
	Remark        string
	SubmitterID   uint64
	AuditorID     uint64
	ExecutorID    uint64
	RejectReason  string
	DelayMinutes  int
	NeedBackup    bool
	ExecuteMethod int
	RiskLevel     int
	ExecuteTime   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SQLOrderItem SQL工单明细
type SQLOrderItem struct {
	ID           uint64
	OrderID      uint64
	Sequence     int
	SQLText      string
	Status       int
	ExecuteType  string
	ExecuteTime  float64
	AffectedRows int
	ResultText   string
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SQLReviewRecord SQL审核记录
type SQLReviewRecord struct {
	ID           uint64
	OrderID      uint64
	ItemID       uint64
	Sequence     string
	Stage        string
	ErrorLevel   int
	StageStatus  string
	ErrorMessage string
	SQLText      string
	AffectedRows int
	BackupDBName string
	ExecuteTime  string
	SQLHash      string
	CreatedAt    time.Time
}

// SQLOrderRepo SQL工单仓储接口
type SQLOrderRepo interface {
	Create(ctx context.Context, order *SQLOrder) (*SQLOrder, error)
	Update(ctx context.Context, order *SQLOrder) error
	GetByID(ctx context.Context, id uint64) (*SQLOrder, error)
	GetByWorkID(ctx context.Context, workID string) (*SQLOrder, error)
	List(ctx context.Context, page, pageSize int, status *int, submitterID *uint64) ([]*SQLOrder, int64, error)

	CreateItem(ctx context.Context, item *SQLOrderItem) error
	UpdateItem(ctx context.Context, item *SQLOrderItem) error
	DeleteItem(ctx context.Context, itemID uint64) error
	GetItemsByOrderID(ctx context.Context, orderID uint64) ([]*SQLOrderItem, error)

	CreateReviewRecord(ctx context.Context, record *SQLReviewRecord) error
	GetReviewRecordsByOrderID(ctx context.Context, orderID uint64) ([]*SQLReviewRecord, error)
}

// SQLOrderUsecase SQL工单用例
type SQLOrderUsecase struct {
	repo           SQLOrderRepo
	sqlTypeChecker *SQLTypeChecker
}

// NewSQLOrderUsecase 创建SQL工单用例
func NewSQLOrderUsecase(repo SQLOrderRepo, sqlTypeChecker *SQLTypeChecker) *SQLOrderUsecase {
	return &SQLOrderUsecase{
		repo:           repo,
		sqlTypeChecker: sqlTypeChecker,
	}
}

// CreateOrder 创建工单
func (uc *SQLOrderUsecase) CreateOrder(ctx context.Context, order *SQLOrder, instanceName string) (*SQLOrder, error) {
	// 生成工单ID（包含类型前缀和实例名后缀）
	order.WorkID = generateSQLWorkID(order.OrderType, instanceName)
	order.Status = OrderStatusPending

	// 判断SQL类型和风险等级
	if order.OrderType == OrderTypeDDL {
		canAuto, types, err := uc.sqlTypeChecker.CanAutoExecute(ctx, order.SQLContent)
		if err == nil {
			// 如果包含危险操作，设置风险等级
			isDangerous, _, _ := uc.sqlTypeChecker.IsDangerousSQL(ctx, order.SQLContent)
			if isDangerous {
				order.RiskLevel = 1 // 危险
			}

			// 如果可以自助执行，可以考虑直接设置为已通过状态
			// 这里根据业务需求决定，暂时保持待审核状态
			_ = canAuto
			_ = types
		}
	}

	return uc.repo.Create(ctx, order)
}

// GetOrder 获取工单
func (uc *SQLOrderUsecase) GetOrder(ctx context.Context, id uint64) (*SQLOrder, error) {
	return uc.repo.GetByID(ctx, id)
}

// ListOrders 列出工单
func (uc *SQLOrderUsecase) ListOrders(ctx context.Context, page, pageSize int, status *int, submitterID *uint64) ([]*SQLOrder, int64, error) {
	return uc.repo.List(ctx, page, pageSize, status, submitterID)
}

// ApproveOrder 审核通过
func (uc *SQLOrderUsecase) ApproveOrder(ctx context.Context, orderID, auditorID uint64) error {
	order, err := uc.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 对于DDL工单，再次检查SQL类型，确保安全
	if order.OrderType == OrderTypeDDL {
		canAuto, _, err := uc.sqlTypeChecker.CanAutoExecute(ctx, order.SQLContent)
		if err == nil && !canAuto {
			// 如果包含危险操作，需要复核
			isDangerous, _, _ := uc.sqlTypeChecker.IsDangerousSQL(ctx, order.SQLContent)
			if isDangerous {
				order.Status = OrderStatusDoubleCheck // 需要复核
			} else {
				order.Status = OrderStatusApproved
			}
		} else {
			order.Status = OrderStatusApproved
		}
	} else {
		order.Status = OrderStatusApproved
	}

	order.AuditorID = auditorID
	return uc.repo.Update(ctx, order)
}

// CheckSQLType 检查SQL类型（用于前端展示）
func (uc *SQLOrderUsecase) CheckSQLType(ctx context.Context, sql string) ([]string, bool, error) {
	types, err := uc.sqlTypeChecker.GetSQLType(ctx, sql)
	if err != nil {
		return nil, false, err
	}

	canAuto, _, err := uc.sqlTypeChecker.CanAutoExecute(ctx, sql)
	if err != nil {
		return types, false, err
	}

	return types, canAuto, nil
}

// RejectOrder 驳回工单
func (uc *SQLOrderUsecase) RejectOrder(ctx context.Context, orderID, auditorID uint64, reason string) error {
	order, err := uc.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	order.Status = OrderStatusRejected
	order.AuditorID = auditorID
	order.RejectReason = reason
	return uc.repo.Update(ctx, order)
}

// UpdateOrder 更新工单
func (uc *SQLOrderUsecase) UpdateOrder(ctx context.Context, order *SQLOrder) error {
	return uc.repo.Update(ctx, order)
}

// generateSQLWorkID 生成SQL工单ID
// orderType: 0-DDL, 1-DML, 2-EXPORT
// instanceName: 实例名称
func generateSQLWorkID(orderType int, instanceName string) string {
	// 类型前缀
	prefix := ""
	switch orderType {
	case OrderTypeDDL:
		prefix = "DDL"
	case OrderTypeDML:
		prefix = "DML"
	case OrderTypeEXPORT:
		prefix = "EXPORT"
	default:
		prefix = "SQL"
	}
	
	// 时间戳（去掉随机字符串）
	timestamp := time.Now().Format("20060102150405")
	
	// 实例名处理：去除特殊字符，只保留字母数字，限制长度
	instanceSuffix := ""
	if instanceName != "" {
		// 只保留字母、数字、下划线、中文字符
		cleaned := ""
		for _, r := range instanceName {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
			   (r >= '0' && r <= '9') || r == '_' || r >= 0x4e00 && r <= 0x9fff {
				cleaned += string(r)
			}
		}
		// 限制长度，避免工单编号过长
		if len(cleaned) > 20 {
			cleaned = cleaned[:20]
		}
		if cleaned != "" {
			instanceSuffix = cleaned
		}
	}
	
	// 格式：前缀_时间戳_实例名
	if instanceSuffix != "" {
		return fmt.Sprintf("%s_%s_%s", prefix, timestamp, instanceSuffix)
	}
	return fmt.Sprintf("%s_%s", prefix, timestamp)
}

// generateWorkID 生成通用工单ID（用于归档、binlog等）
func generateWorkID() string {
	return time.Now().Format("20060102150405") + randomString(6)
}

func randomString(n int) string {
	const letters = "0123456789abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GetReviewRecords 获取工单审核记录
func (uc *SQLOrderUsecase) GetReviewRecords(ctx context.Context, orderID uint64) ([]*SQLReviewRecord, error) {
	return uc.repo.GetReviewRecordsByOrderID(ctx, orderID)
}

// CreateReviewRecord 创建审核记录
func (uc *SQLOrderUsecase) CreateReviewRecord(ctx context.Context, record *SQLReviewRecord) error {
	return uc.repo.CreateReviewRecord(ctx, record)
}

// GetOrderItems 获取工单明细
func (uc *SQLOrderUsecase) GetOrderItems(ctx context.Context, orderID uint64) ([]*SQLOrderItem, error) {
	return uc.repo.GetItemsByOrderID(ctx, orderID)
}

// CreateItem 创建工单明细
func (uc *SQLOrderUsecase) CreateItem(ctx context.Context, item *SQLOrderItem) error {
	return uc.repo.CreateItem(ctx, item)
}

// GetItemsByOrderID 获取工单明细列表（别名，供 task_handler 调用）
func (uc *SQLOrderUsecase) GetItemsByOrderID(ctx context.Context, orderID uint64) ([]*SQLOrderItem, error) {
	return uc.repo.GetItemsByOrderID(ctx, orderID)
}

// DeleteItem 删除工单明细
func (uc *SQLOrderUsecase) DeleteItem(ctx context.Context, itemID uint64) error {
	return uc.repo.DeleteItem(ctx, itemID)
}
