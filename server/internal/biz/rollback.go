package biz

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// SQLRollback SQL回滚记录
type SQLRollback struct {
	ID           uint64
	OrderID      uint64
	OrderItemID  *uint64
	BackupDBName string
	OpidTime     string
	TableName    string
	RollbackSQL  string
	CreatedAt    time.Time
}

// RollbackRepo 回滚记录仓储接口
type RollbackRepo interface {
	Create(ctx context.Context, rollback *SQLRollback) error
	GetByOrderID(ctx context.Context, orderID uint64) ([]*SQLRollback, error)
	GetByOrderItemID(ctx context.Context, orderItemID uint64) ([]*SQLRollback, error)
}

// RollbackUsecase 回滚业务逻辑
type RollbackUsecase struct {
	repo         RollbackRepo
	inceptionCfg *InceptionConfig
	log          *log.Helper
}

// NewRollbackUsecase 创建回滚业务逻辑
func NewRollbackUsecase(repo RollbackRepo, inceptionCfg *InceptionConfig, logger log.Logger) *RollbackUsecase {
	return &RollbackUsecase{
		repo:         repo,
		inceptionCfg: inceptionCfg,
		log:          log.NewHelper(logger),
	}
}

// GenerateAndSaveRollbackSQL 生成并保存回滚SQL
func (uc *RollbackUsecase) GenerateAndSaveRollbackSQL(ctx context.Context, orderID uint64, orderItemID *uint64, backupDBName, opidTime string) error {
	if backupDBName == "" || opidTime == "" {
		// 如果没有备份信息，不生成回滚SQL
		return nil
	}

	// 从goInception备份库查询表名
	tableName, err := uc.getTableNameFromBackup(ctx, backupDBName, opidTime)
	if err != nil {
		uc.log.Warnf("获取表名失败: %v", err)
		return err
	}

	// 从goInception备份库查询回滚SQL
	rollbackSQLs, err := uc.getRollbackSQLFromBackup(ctx, backupDBName, tableName, opidTime)
	if err != nil {
		uc.log.Warnf("获取回滚SQL失败: %v", err)
		return err
	}

	// 保存每条回滚SQL
	for _, sql := range rollbackSQLs {
		rollback := &SQLRollback{
			OrderID:      orderID,
			OrderItemID:  orderItemID,
			BackupDBName: backupDBName,
			OpidTime:     opidTime,
			TableName:    tableName,
			RollbackSQL:  sql,
		}
		if err := uc.repo.Create(ctx, rollback); err != nil {
			uc.log.Errorf("保存回滚SQL失败: %v", err)
			return err
		}
	}

	return nil
}

// getTableNameFromBackup 从备份库获取表名
func (uc *RollbackUsecase) getTableNameFromBackup(ctx context.Context, backupDBName, opidTime string) (string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4",
		uc.inceptionCfg.BackupUsername,
		uc.inceptionCfg.BackupPassword,
		uc.inceptionCfg.BackupHost,
		uc.inceptionCfg.BackupPort,
		backupDBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var tableName string
	query := `SELECT tablename FROM $_$Inception_backup_information$_$ WHERE opid_time = ? LIMIT 1`
	err = db.QueryRowContext(ctx, query, opidTime).Scan(&tableName)
	if err != nil {
		return "", fmt.Errorf("查询表名失败: %w", err)
	}

	return tableName, nil
}

// getRollbackSQLFromBackup 从备份库获取回滚SQL
func (uc *RollbackUsecase) getRollbackSQLFromBackup(ctx context.Context, backupDBName, tableName, opidTime string) ([]string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4",
		uc.inceptionCfg.BackupUsername,
		uc.inceptionCfg.BackupPassword,
		uc.inceptionCfg.BackupHost,
		uc.inceptionCfg.BackupPort)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 构建查询SQL：SELECT rollback_statement FROM backup_db.table WHERE opid_time = ?
	fullTableName := fmt.Sprintf("`%s`.`%s`", backupDBName, tableName)
	query := fmt.Sprintf("SELECT rollback_statement FROM %s WHERE opid_time = ?", fullTableName)

	rows, err := db.QueryContext(ctx, query, opidTime)
	if err != nil {
		return nil, fmt.Errorf("查询回滚SQL失败: %w", err)
	}
	defer rows.Close()

	var rollbackSQLs []string
	for rows.Next() {
		var sql string
		if err := rows.Scan(&sql); err != nil {
			return nil, fmt.Errorf("扫描回滚SQL失败: %w", err)
		}
		if sql != "" {
			rollbackSQLs = append(rollbackSQLs, sql)
		}
	}

	return rollbackSQLs, nil
}

// GetRollbackSQLs 获取工单的回滚SQL列表
func (uc *RollbackUsecase) GetRollbackSQLs(ctx context.Context, orderID uint64) ([]*SQLRollback, error) {
	return uc.repo.GetByOrderID(ctx, orderID)
}

// GetRollbackSQLsByItem 获取工单明细的回滚SQL列表
func (uc *RollbackUsecase) GetRollbackSQLsByItem(ctx context.Context, orderItemID uint64) ([]*SQLRollback, error) {
	return uc.repo.GetByOrderItemID(ctx, orderItemID)
}
