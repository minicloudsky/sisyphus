package sqlparser

import (
	"strings"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/parser/test_driver"
)

// DDL类型常量
const (
	// 数据库操作
	DdlTypeCreateDatabase = "create database"
	DdlTypeDropDatabase   = "drop database"

	// 表操作
	DdlTypeCreateTable  = "create table"
	DdlTypeTruncateTable = "truncate table"
	DdlTypeDropTable    = "drop table"
	DdlTypeRenameTable  = "rename table"

	// 列操作
	DdlTypeAddColumn    = "add column"
	DdlTypeModifyColumn = "modify column"
	DdlTypeChangeColumn = "change column"
	DdlTypeAlterColumn  = "alter column"
	DdlTypeDropColumn   = "drop column"

	// 索引操作
	DdlTypeAddIndex    = "add index"
	DdlTypeCreateIndex = "create index"
	DdlTypeDropIndex   = "drop index"

	// 分区表
	DdlTypePartitionTable = "partition"

	// 未知类型
	DdlTypeUnknown = "unknown"
)

// SqlTypeItem SQL类型项
type SqlTypeItem struct {
	Sql  string
	Type []string
}

// ParseSqlType 解析SQL类型
func ParseSqlType(sqls []string) []SqlTypeItem {
	sqlTypeMap := make(map[string][]string, 0)
	for _, sql := range sqls {
		sqlTypeMap[sql] = make([]string, 0)
	}

	for _, originSql := range sqls {
		p := parser.New()
		formattedSql := formatSQL(originSql)
		
		stmts, _, err := p.Parse(formattedSql, "", "")
		if err != nil {
			if len(sqlTypeMap[originSql]) == 0 {
				sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeUnknown)
			}
			continue
		}

		for _, stmt := range stmts {
			parseStatement(stmt, originSql, sqlTypeMap)
		}
	}

	// 转换为结果
	var sqlTypes []SqlTypeItem
	for sql, types := range sqlTypeMap {
		uniqTypes := removeDuplicateStr(types)
		// 当有多个sql类型匹配上时候，去掉未知的类型，按照已有的类型进行判断
		if len(uniqTypes) >= 2 && findStr(uniqTypes, DdlTypeUnknown) {
			uniqTypes = removeStr(uniqTypes, DdlTypeUnknown)
		}
		sqlTypes = append(sqlTypes, SqlTypeItem{
			Sql:  sql,
			Type: uniqTypes,
		})
	}
	return sqlTypes
}

// formatSQL 格式化SQL
func formatSQL(sql string) string {
	formattedSql := strings.ToLower(sql)
	formattedSql = strings.ReplaceAll(formattedSql, "\n", " ")
	formattedSql = strings.ReplaceAll(formattedSql, "  ", " ")
	formattedSql = strings.ReplaceAll(formattedSql, ", ", ",")
	formattedSql = strings.ReplaceAll(formattedSql, " ,", ",")
	formattedSql = strings.ReplaceAll(formattedSql, " , ", ",")
	formattedSql = strings.ReplaceAll(formattedSql, " =", "=")
	formattedSql = strings.ReplaceAll(formattedSql, "= ", "=")
	formattedSql = strings.ReplaceAll(formattedSql, "algorithm=inplace,lock=none,", "")
	formattedSql = strings.ReplaceAll(formattedSql, ",algorithm=inplace,lock=none", "")
	formattedSql = strings.ReplaceAll(formattedSql, ",algorithm=instant,lock=default", "")
	formattedSql = strings.ReplaceAll(formattedSql, "algorithm=instant,lock=default,", "")
	return formattedSql
}

// parseStatement 解析语句
func parseStatement(stmt ast.StmtNode, originSql string, sqlTypeMap map[string][]string) {
	switch stmt.(type) {
	case *ast.DropDatabaseStmt:
		sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeDropDatabase)
	case *ast.DropTableStmt:
		sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeDropTable)
	case *ast.RenameTableStmt:
		sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeRenameTable)
	case *ast.TruncateTableStmt:
		sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeTruncateTable)
	case *ast.DropIndexStmt:
		sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeDropIndex)
	case *ast.AlterTableStmt:
		parseAlterTableStmt(stmt, originSql, sqlTypeMap)
	case *ast.CreateIndexStmt:
		sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeCreateIndex)
	case *ast.CreateTableStmt:
		sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeCreateTable)
		createTableStmt, ok := stmt.(*ast.CreateTableStmt)
		if ok && createTableStmt.Partition != nil && 
			createTableStmt.Partition.PartitionMethod.Num > 0 && 
			len(createTableStmt.Partition.Definitions) > 0 {
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypePartitionTable)
		}
	case *ast.CreateDatabaseStmt:
		sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeCreateDatabase)
	default:
		if len(sqlTypeMap[originSql]) == 0 {
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeUnknown)
		}
	}
}

// parseAlterTableStmt 解析ALTER TABLE语句
func parseAlterTableStmt(stmt ast.StmtNode, originSql string, sqlTypeMap map[string][]string) {
	alterStmt, ok := stmt.(*ast.AlterTableStmt)
	if !ok {
		return
	}

	for _, spec := range alterStmt.Specs {
		switch spec.Tp {
		case ast.AlterTableRenameTable:
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeRenameTable)
		case ast.AlterTableDropIndex, ast.AlterTableDropForeignKey, ast.AlterTableDropPrimaryKey:
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeDropIndex)
		case ast.AlterTableDropColumn:
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeDropColumn)
		case ast.AlterTableChangeColumn:
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeChangeColumn)
		case ast.AlterTableModifyColumn:
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeModifyColumn)
		case ast.AlterTableAlterColumn:
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeAlterColumn)
		case ast.AlterTableAddColumns:
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeAddColumn)
		case ast.AlterTableAddConstraint:
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeAddIndex)
		case ast.AlterTablePartition, ast.AlterTableExchangePartition,
			ast.AlterTableReorganizePartition, ast.AlterTableCheckPartitions,
			ast.AlterTableAddPartitions, ast.AlterTableOptimizePartition,
			ast.AlterTableCoalescePartitions, ast.AlterTableRemovePartitioning,
			ast.AlterTableTruncatePartition, ast.AlterTableRepairPartition,
			ast.AlterTableDropFirstPartition, ast.AlterTableAddLastPartition,
			ast.AlterTableReorganizeFirstPartition, ast.AlterTableReorganizeLastPartition,
			ast.AlterTableDropPartition:
			sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypePartitionTable)
		default:
			if len(sqlTypeMap[originSql]) == 0 {
				sqlTypeMap[originSql] = append(sqlTypeMap[originSql], DdlTypeUnknown)
			}
		}
	}
}

// 工具函数
func removeDuplicateStr(strs []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, str := range strs {
		if !keys[str] {
			keys[str] = true
			result = append(result, str)
		}
	}
	return result
}

func findStr(strs []string, target string) bool {
	for _, str := range strs {
		if str == target {
			return true
		}
	}
	return false
}

func removeStr(strs []string, target string) []string {
	var result []string
	for _, str := range strs {
		if str != target {
			result = append(result, str)
		}
	}
	return result
}
