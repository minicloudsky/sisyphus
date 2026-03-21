package biz

import (
	"context"
	"strings"

	"sisyphus/server/internal/pkg/sqlparser"
)

// SQLTypeChecker SQL类型检查器
type SQLTypeChecker struct{}

// NewSQLTypeChecker 创建SQL类型检查器
func NewSQLTypeChecker() *SQLTypeChecker {
	return &SQLTypeChecker{}
}

// GetSQLType 获取SQL类型
func (c *SQLTypeChecker) GetSQLType(ctx context.Context, sql string) ([]string, error) {
	sqls := []string{sql}
	results := sqlparser.ParseSqlType(sqls)
	if len(results) > 0 {
		return results[0].Type, nil
	}
	return []string{sqlparser.DdlTypeUnknown}, nil
}

// GetSQLTypes 批量获取SQL类型
func (c *SQLTypeChecker) GetSQLTypes(ctx context.Context, sqls []string) (map[string][]string, error) {
	results := sqlparser.ParseSqlType(sqls)
	resultMap := make(map[string][]string)
	for _, item := range results {
		resultMap[item.Sql] = item.Type
	}
	return resultMap, nil
}

// CanAutoExecute 判断SQL是否可以自助执行
// 根据SQL类型判断是否需要审核或可以直接执行
func (c *SQLTypeChecker) CanAutoExecute(ctx context.Context, sql string) (bool, []string, error) {
	types, err := c.GetSQLType(ctx, sql)
	if err != nil {
		return false, nil, err
	}

	// 危险操作，不能自助执行
	dangerousTypes := []string{
		sqlparser.DdlTypeDropDatabase,
		sqlparser.DdlTypeDropTable,
		sqlparser.DdlTypeTruncateTable,
		sqlparser.DdlTypeRenameTable,
		sqlparser.DdlTypeDropColumn,
		sqlparser.DdlTypeDropIndex,
		sqlparser.DdlTypePartitionTable,
	}

	// 检查是否包含危险操作
	for _, sqlType := range types {
		for _, dangerousType := range dangerousTypes {
			if strings.EqualFold(sqlType, dangerousType) {
				return false, types, nil
			}
		}
	}

	// 检查是否包含未知类型
	for _, sqlType := range types {
		if strings.EqualFold(sqlType, sqlparser.DdlTypeUnknown) {
			return false, types, nil
		}
	}

	// 其他类型可以自助执行（如：create table, add column, modify column, create index等）
	return true, types, nil
}

// IsDangerousSQL 判断是否为危险SQL
func (c *SQLTypeChecker) IsDangerousSQL(ctx context.Context, sql string) (bool, []string, error) {
	types, err := c.GetSQLType(ctx, sql)
	if err != nil {
		return false, nil, err
	}

	dangerousTypes := []string{
		sqlparser.DdlTypeDropDatabase,
		sqlparser.DdlTypeDropTable,
		sqlparser.DdlTypeTruncateTable,
		sqlparser.DdlTypeRenameTable,
		sqlparser.DdlTypeDropColumn,
		sqlparser.DdlTypeDropIndex,
	}

	for _, sqlType := range types {
		for _, dangerousType := range dangerousTypes {
			if strings.EqualFold(sqlType, dangerousType) {
				return true, types, nil
			}
		}
	}

	return false, types, nil
}
