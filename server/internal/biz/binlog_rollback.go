package biz

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BinlogRollbackUsecase Binlog回滚用例
type BinlogRollbackUsecase struct {
	repo BinlogRollbackRepo
}

// NewBinlogRollbackUsecase 创建Binlog回滚用例
func NewBinlogRollbackUsecase(repo BinlogRollbackRepo) *BinlogRollbackUsecase {
	return &BinlogRollbackUsecase{repo: repo}
}

// BinlogRollbackOrder Binlog回滚工单
type BinlogRollbackOrder struct {
	ID             uint64
	WorkID         string
	InstanceID     uint64
	DatabaseName   string
	StartTime      time.Time
	StopTime       time.Time
	BinlogFile     string
	TableList      []string
	Status         int // 0-解析中, 1-解析成功, 2-解析失败
	RollbackSQLCount int
	SubmitterID    uint64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// BinlogRollbackSQL Binlog回滚SQL
type BinlogRollbackSQL struct {
	ID          uint64
	OrderID     uint64
	TableName   string
	RollbackSQL string
	OriginalSQL string
	CreatedAt   time.Time
}

// BinlogRollbackRepo Binlog回滚仓储接口
type BinlogRollbackRepo interface {
	CreateOrder(ctx context.Context, order *BinlogRollbackOrder) (*BinlogRollbackOrder, error)
	UpdateOrder(ctx context.Context, order *BinlogRollbackOrder) error
	GetOrderByID(ctx context.Context, id uint64) (*BinlogRollbackOrder, error)
	GetOrderByWorkID(ctx context.Context, workID string) (*BinlogRollbackOrder, error)
	ListOrders(ctx context.Context, page, pageSize int) ([]*BinlogRollbackOrder, int64, error)
	
	CreateRollbackSQL(ctx context.Context, sql *BinlogRollbackSQL) error
	GetRollbackSQLsByOrderID(ctx context.Context, orderID uint64) ([]*BinlogRollbackSQL, error)
}

// CreateRollbackOrder 创建回滚工单
func (uc *BinlogRollbackUsecase) CreateRollbackOrder(ctx context.Context, order *BinlogRollbackOrder) (*BinlogRollbackOrder, error) {
	order.WorkID = generateWorkID()
	order.Status = 0 // 解析中
	return uc.repo.CreateOrder(ctx, order)
}

// GetOrderByID 获取回滚工单
func (uc *BinlogRollbackUsecase) GetOrderByID(ctx context.Context, id uint64) (*BinlogRollbackOrder, error) {
	return uc.repo.GetOrderByID(ctx, id)
}

// ListOrders 列出回滚工单
func (uc *BinlogRollbackUsecase) ListOrders(ctx context.Context, page, pageSize int) ([]*BinlogRollbackOrder, int64, error) {
	return uc.repo.ListOrders(ctx, page, pageSize)
}

// GetRollbackSQLsByOrderID 获取回滚SQL列表
func (uc *BinlogRollbackUsecase) GetRollbackSQLsByOrderID(ctx context.Context, orderID uint64) ([]*BinlogRollbackSQL, error) {
	return uc.repo.GetRollbackSQLsByOrderID(ctx, orderID)
}

// UpdateOrder 更新回滚工单
func (uc *BinlogRollbackUsecase) UpdateOrder(ctx context.Context, order *BinlogRollbackOrder) error {
	return uc.repo.UpdateOrder(ctx, order)
}

// ParseBinlog 解析Binlog
func (uc *BinlogRollbackUsecase) ParseBinlog(ctx context.Context, orderID uint64, instance *InstanceInfo, binlogFile string, startTime, stopTime time.Time, tableList []string) error {
	// 使用mysqlbinlog解析binlog
	// 支持时间范围解析
	cmdArgs := []string{
		"--no-defaults",
		"-R",
		"--base64-output=decode-rows",
		"-v",
		"-h", instance.Host,
		"-u", instance.Username,
		"-p" + instance.Password,
	}

	// 添加时间范围
	if !startTime.IsZero() {
		cmdArgs = append(cmdArgs, "--start-datetime", startTime.Format("2006-01-02 15:04:05"))
	}
	if !stopTime.IsZero() {
		cmdArgs = append(cmdArgs, "--stop-datetime", stopTime.Format("2006-01-02 15:04:05"))
	}

	// 添加数据库和表过滤
	if len(tableList) > 0 {
		for _, table := range tableList {
			cmdArgs = append(cmdArgs, "-d", table)
		}
	}

	cmdArgs = append(cmdArgs, binlogFile)

	cmd := exec.CommandContext(ctx, "mysqlbinlog", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("binlog解析失败: %v, output: %s", err, string(output))
	}

	// 解析输出，生成回滚SQL
	rollbackSQLs := uc.parseBinlogOutput(string(output), tableList)
	
	// 保存回滚SQL
	for _, sql := range rollbackSQLs {
		rollbackSQL := &BinlogRollbackSQL{
			OrderID:     orderID,
			TableName:   sql.TableName,
			RollbackSQL: sql.RollbackSQL,
			OriginalSQL: sql.OriginalSQL,
		}
		if err := uc.repo.CreateRollbackSQL(ctx, rollbackSQL); err != nil {
			return err
		}
	}

	// 更新工单状态
	order, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	order.Status = 1 // 解析成功
	order.RollbackSQLCount = len(rollbackSQLs)
	return uc.repo.UpdateOrder(ctx, order)
}

// parseBinlogOutput 解析binlog输出，生成回滚SQL
func (uc *BinlogRollbackUsecase) parseBinlogOutput(output string, tableList []string) []*BinlogRollbackSQL {
	var rollbackSQLs []*BinlogRollbackSQL
	lines := strings.Split(output, "\n")
	
	var currentTable string
	var currentSQL strings.Builder
	var currentType string // INSERT, UPDATE, DELETE
	var inStatement bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// 识别表名
		if strings.HasPrefix(line, "### Table map:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				currentTable = strings.Trim(parts[2], "`")
				// 如果指定了表列表，只处理指定的表
				if len(tableList) > 0 {
					found := false
					for _, t := range tableList {
						if strings.Contains(currentTable, t) {
							found = true
							break
						}
					}
					if !found {
						currentTable = ""
						continue
					}
				}
			}
		}
		
		// 识别SQL类型
		if strings.HasPrefix(line, "### INSERT INTO") {
			currentType = "INSERT"
			inStatement = true
			currentSQL.Reset()
			currentSQL.WriteString(line)
		} else if strings.HasPrefix(line, "### UPDATE") {
			currentType = "UPDATE"
			inStatement = true
			currentSQL.Reset()
			currentSQL.WriteString(line)
		} else if strings.HasPrefix(line, "### DELETE FROM") {
			currentType = "DELETE"
			inStatement = true
			currentSQL.Reset()
			currentSQL.WriteString(line)
		} else if inStatement {
			currentSQL.WriteString("\n")
			currentSQL.WriteString(line)
			
			// 检查是否结束
			if strings.HasPrefix(line, "### @") || strings.HasPrefix(line, "# at") {
				// 生成回滚SQL
				if currentTable != "" && currentSQL.Len() > 0 {
					rollbackSQL := uc.generateRollbackSQL(currentType, currentTable, currentSQL.String())
					if rollbackSQL != nil {
						rollbackSQLs = append(rollbackSQLs, rollbackSQL)
					}
				}
				inStatement = false
				currentTable = ""
			}
		}
	}
	
	return rollbackSQLs
}

// generateRollbackSQL 生成回滚SQL
func (uc *BinlogRollbackUsecase) generateRollbackSQL(sqlType, tableName, originalSQL string) *BinlogRollbackSQL {
	rollbackSQL := &BinlogRollbackSQL{
		TableName:   tableName,
		OriginalSQL: originalSQL,
	}
	
	switch sqlType {
	case "INSERT":
		// INSERT -> DELETE
		// 从INSERT语句中提取WHERE条件（基于插入的值）
		rollbackSQL.RollbackSQL = uc.insertToDelete(originalSQL, tableName)
	case "UPDATE":
		// UPDATE -> UPDATE（恢复旧值）
		rollbackSQL.RollbackSQL = uc.updateToRollback(originalSQL, tableName)
	case "DELETE":
		// DELETE -> INSERT
		rollbackSQL.RollbackSQL = uc.deleteToInsert(originalSQL, tableName)
	}
	
	return rollbackSQL
}

// extractFieldValue extracts the raw value from a binlog verbose line like:
//
//	###   @1=100 /* INT meta=0 ... */
//	###   @2='John' /* VARSTRING... */
//	###   @3=NULL
//
// It returns the value portion (e.g. "100", "'John'", "NULL").
func extractFieldValue(line string) string {
	// Strip leading "###" and whitespace
	line = strings.TrimPrefix(line, "###")
	line = strings.TrimSpace(line)

	// Find the '=' separating @N from its value
	eqIdx := strings.Index(line, "=")
	if eqIdx < 0 {
		return ""
	}
	value := strings.TrimSpace(line[eqIdx+1:])

	// Strip trailing /* comment */ if present
	if commentIdx := strings.Index(value, " /*"); commentIdx >= 0 {
		value = strings.TrimSpace(value[:commentIdx])
	}
	return value
}

// parseFieldValues parses a binlog verbose SQL block and returns the ordered
// list of raw values found in the named section ("SET" or "WHERE").
func parseFieldValues(sqlBlock, section string) []string {
	lines := strings.Split(sqlBlock, "\n")
	var values []string
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Section header: "### SET" or "### WHERE"
		if trimmed == "### "+section {
			inSection = true
			continue
		}

		// Any other "### SET" or "### WHERE" ends the current section
		if trimmed == "### SET" || trimmed == "### WHERE" {
			inSection = false
			continue
		}

		if inSection {
			// Field lines look like: "###   @N=value /* type... */"
			if strings.HasPrefix(trimmed, "### @") || strings.HasPrefix(trimmed, "###   @") {
				values = append(values, extractFieldValue(trimmed))
			}
		}
	}
	return values
}

// insertToDelete 将INSERT转换为DELETE
// Parses the ### SET section of the binlog verbose INSERT block and generates
// a DELETE ... WHERE clause using @N positional references.
func (uc *BinlogRollbackUsecase) insertToDelete(sql, tableName string) string {
	values := parseFieldValues(sql, "SET")
	if len(values) == 0 {
		return fmt.Sprintf("DELETE FROM `%s` WHERE /* could not parse binlog SET values */", tableName)
	}

	conditions := make([]string, len(values))
	for i, v := range values {
		conditions[i] = fmt.Sprintf("@%d=%s", i+1, v)
	}
	return fmt.Sprintf("DELETE FROM `%s` WHERE %s;", tableName, strings.Join(conditions, " AND "))
}

// updateToRollback 将UPDATE转换为回滚UPDATE
// Swaps the WHERE (old values) and SET (new values) sections so that the
// rollback UPDATE restores the original row.
func (uc *BinlogRollbackUsecase) updateToRollback(sql, tableName string) string {
	whereValues := parseFieldValues(sql, "WHERE")
	setValues := parseFieldValues(sql, "SET")

	if len(whereValues) == 0 || len(setValues) == 0 {
		return fmt.Sprintf("UPDATE `%s` SET /* could not parse binlog SET/WHERE values */", tableName)
	}

	// The rollback SET uses the old (WHERE) values; rollback WHERE uses new (SET) values.
	setClauses := make([]string, len(whereValues))
	for i, v := range whereValues {
		setClauses[i] = fmt.Sprintf("@%d=%s", i+1, v)
	}

	whereClauses := make([]string, len(setValues))
	for i, v := range setValues {
		whereClauses[i] = fmt.Sprintf("@%d=%s", i+1, v)
	}

	return fmt.Sprintf("UPDATE `%s` SET %s WHERE %s;",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))
}

// deleteToInsert 将DELETE转换为INSERT
// Parses the ### WHERE section of the binlog verbose DELETE block and generates
// an INSERT INTO ... VALUES (...) statement.
func (uc *BinlogRollbackUsecase) deleteToInsert(sql, tableName string) string {
	values := parseFieldValues(sql, "WHERE")
	if len(values) == 0 {
		return fmt.Sprintf("INSERT INTO `%s` VALUES (/* could not parse binlog WHERE values */);", tableName)
	}
	return fmt.Sprintf("INSERT INTO `%s` VALUES (%s);", tableName, strings.Join(values, ", "))
}
