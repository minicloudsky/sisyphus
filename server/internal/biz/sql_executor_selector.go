package biz

import (
	"context"
	"strings"

	"sisyphus/server/internal/pkg/sqlparser"
)

// SQLExecutorSelector SQL执行器选择器
type SQLExecutorSelector struct {
	typeChecker *SQLTypeChecker
}

// NewSQLExecutorSelector 创建SQL执行器选择器
func NewSQLExecutorSelector() *SQLExecutorSelector {
	return &SQLExecutorSelector{
		typeChecker: NewSQLTypeChecker(),
	}
}

// SelectExecuteMethod 根据SQL类型选择执行方式
// 返回执行方式：ExecuteMethodGhOst, ExecuteMethodGoInception, ExecuteMethodNative
func (s *SQLExecutorSelector) SelectExecuteMethod(ctx context.Context, sql string) (int, error) {
	types, err := s.typeChecker.GetSQLType(ctx, sql)
	if err != nil {
		// 如果解析失败，默认使用goInception
		return ExecuteMethodGoInception, err
	}

	// gh-ost执行的DDL类型（参考Python项目的DDL_TYPE_GHOST_EXECUTE）
	ghostExecuteTypes := []string{
		sqlparser.DdlTypePartitionTable,
		sqlparser.DdlTypeAddIndex,
		sqlparser.DdlTypeCreateIndex,
		sqlparser.DdlTypeAddColumn,
		sqlparser.DdlTypeModifyColumn,
		sqlparser.DdlTypeChangeColumn,
		sqlparser.DdlTypeAlterColumn,
		sqlparser.DdlTypeDropColumn,
		sqlparser.DdlTypeDropIndex,
		sqlparser.DdlTypeUnknown,
	}

	// goInception执行的DDL类型（参考Python项目的DDL_TYPE_GOINCEPTION_EXECUTE）
	goInceptionExecuteTypes := []string{
		sqlparser.DdlTypeCreateTable,
		sqlparser.DdlTypeTruncateTable,
		sqlparser.DdlTypeDropTable,
		sqlparser.DdlTypeRenameTable,
	}

	// 检查是否匹配gh-ost执行类型
	for _, sqlType := range types {
		for _, ghostType := range ghostExecuteTypes {
			if strings.EqualFold(sqlType, ghostType) {
				return ExecuteMethodGhOst, nil
			}
		}
	}

	// 检查是否匹配goInception执行类型
	for _, sqlType := range types {
		for _, inceptionType := range goInceptionExecuteTypes {
			if strings.EqualFold(sqlType, inceptionType) {
				return ExecuteMethodGoInception, nil
			}
		}
	}

	// 默认使用goInception（包括DML语句）
	return ExecuteMethodGoInception, nil
}

// ShouldUseGhOst 判断是否应该使用gh-ost
func (s *SQLExecutorSelector) ShouldUseGhOst(ctx context.Context, sql string) (bool, error) {
	method, err := s.SelectExecuteMethod(ctx, sql)
	if err != nil {
		return false, err
	}
	return method == ExecuteMethodGhOst, nil
}

// ShouldUseGoInception 判断是否应该使用goInception
func (s *SQLExecutorSelector) ShouldUseGoInception(ctx context.Context, sql string) (bool, error) {
	method, err := s.SelectExecuteMethod(ctx, sql)
	if err != nil {
		return false, err
	}
	return method == ExecuteMethodGoInception, nil
}
