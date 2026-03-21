package biz

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
)

// CKCheckExecutor ClickHouse SQL审核执行器（ckcheck逻辑原生实现）
type CKCheckExecutor struct{}

// NewCKCheckExecutor 创建CKCheck执行器
func NewCKCheckExecutor() *CKCheckExecutor {
	return &CKCheckExecutor{}
}

// CKCheckResult ClickHouse审核结果（与goInception格式兼容）
type CKCheckResult struct {
	OrderID      int    `json:"order_id"`
	Stage        string `json:"stage"`
	ErrorLevel   int    `json:"error_level"` // 0-info, 1-warning, 2-error
	StageStatus  string `json:"stage_status"`
	ErrorMessage string `json:"error_message"`
	SQL          string `json:"sql"`
	AffectedRows int    `json:"affected_rows"`
	Sequence     string `json:"sequence"`
	ExecuteTime  string `json:"execute_time"`
}

// ckConn 创建ClickHouse连接（DSN：host:port）
func ckConn(host string, port int, username, password, database string) (clickhouse.Conn, error) {
	if database == "" {
		database = "default"
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", host, port)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 10 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// ckTableRegexp 从SQL提取表名的正则
var ckTableRegexp = regexp.MustCompile(`(?i)\b(?:FROM|JOIN|INTO|UPDATE|TABLE)\s+[` + "`" + `"]?(\w+)[` + "`" + `"]?(?:\s|,|$|\()`)

// extractCKTableNames 从SQL语句中提取涉及的表名列表
func extractCKTableNames(sql string) []string {
	matches := ckTableRegexp.FindAllStringSubmatch(sql, -1)
	seen := map[string]bool{}
	var tables []string
	for _, m := range matches {
		name := strings.ToLower(m[1])
		// 过滤ClickHouse系统表名和关键字
		skipList := []string{"dual", "values", "select", "where", "and", "or", "not", "in", "is", "null", "true", "false"}
		skip := false
		for _, kw := range skipList {
			if name == kw {
				skip = true
				break
			}
		}
		if !skip && !seen[name] {
			seen[name] = true
			tables = append(tables, m[1])
		}
	}
	return tables
}

// generateTempDBPrefix 生成随机临时数据库前缀
func generateTempDBPrefix() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return "sisyphus_ck_tmp_" + string(b)
}

// AuditAndExecuteWithCKCheck 使用ckcheck方式审核并执行ClickHouse SQL
// prodInstance: 生产实例（用于获取建表语句）
// testInstance: 测试实例（用于创建临时表+试跑SQL）
// sqlContent: 要执行的SQL（可包含多条，分号分隔）
// database: 目标数据库名
// dryRun: true=仅审核不执行生产
func (e *CKCheckExecutor) AuditAndExecuteWithCKCheck(
	ctx context.Context,
	prodInstance *DatabaseInstance,
	testInstance *DatabaseInstance,
	sqlContent string,
	database string,
	dryRun bool,
) ([]*CKCheckResult, error) {
	// 分割SQL
	sqls := splitCKSQLs(sqlContent)
	if len(sqls) == 0 {
		return nil, fmt.Errorf("没有可审核的SQL")
	}

	// 连接生产实例（只读：获取建表语句）
	prodConn, err := ckConn(prodInstance.Host, prodInstance.Port, prodInstance.Username, prodInstance.Password, database)
	if err != nil {
		return nil, fmt.Errorf("连接生产ClickHouse实例失败: %v", err)
	}
	defer prodConn.Close()

	// 连接测试实例
	testConn, err := ckConn(testInstance.Host, testInstance.Port, testInstance.Username, testInstance.Password, "")
	if err != nil {
		return nil, fmt.Errorf("连接测试ClickHouse实例失败: %v", err)
	}
	defer testConn.Close()

	// 创建临时数据库
	tmpDB := generateTempDBPrefix()
	if err := testConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", tmpDB)); err != nil {
		return nil, fmt.Errorf("创建临时数据库失败: %v", err)
	}
	// defer删除临时数据库
	defer func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		testConn.Exec(cleanCtx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", tmpDB)) //nolint:errcheck
	}()

	// 收集所有涉及的表名并在测试实例上创建镜像表
	allTables := map[string]bool{}
	for _, sql := range sqls {
		for _, t := range extractCKTableNames(sql) {
			allTables[t] = true
		}
	}

	// 从生产获取SHOW CREATE TABLE，在测试库中创建
	for tableName := range allTables {
		createSQL, err := getCKCreateTable(ctx, prodConn, database, tableName)
		if err != nil {
			// 表不存在或获取失败：跳过，后续执行时会报错
			continue
		}
		// 替换数据库名和引擎（ReplicatedMergeTree -> MergeTree）
		adaptedSQL := adaptCKCreateTableForTest(createSQL, database, tmpDB)
		if err := testConn.Exec(ctx, adaptedSQL); err != nil {
			// 建表失败也继续，让SQL执行时报错更直观
			continue
		}
	}

	// 逐条执行SQL并收集结果
	var results []*CKCheckResult
	for i, sql := range sqls {
		result := &CKCheckResult{
			OrderID:  i + 1,
			Stage:    "Execute",
			Sequence: fmt.Sprintf("%d", i+1),
			SQL:      sql,
		}

		// 替换SQL中的数据库名为临时库
		testSQL := replaceCKDatabase(sql, database, tmpDB)

		startTime := time.Now()
		execErr := testConn.Exec(ctx, testSQL)
		elapsed := time.Since(startTime)
		result.ExecuteTime = fmt.Sprintf("%.3fs", elapsed.Seconds())

		if execErr != nil {
			result.ErrorLevel = 2
			result.StageStatus = "Execute Failed"
			result.ErrorMessage = convertCKError(execErr.Error())
		} else {
			result.ErrorLevel = 0
			result.StageStatus = "Execute Successfully"
		}

		results = append(results, result)
	}

	// 如果是dry run，到此结束
	if dryRun {
		return results, nil
	}

	// 检查是否有错误
	for _, r := range results {
		if r.ErrorLevel == 2 {
			return results, fmt.Errorf("SQL审核发现错误，停止执行生产")
		}
	}

	// 执行生产实例
	prodDB, err := ckConn(prodInstance.Host, prodInstance.Port, prodInstance.Username, prodInstance.Password, database)
	if err != nil {
		return results, fmt.Errorf("连接生产实例执行失败: %v", err)
	}
	defer prodDB.Close()

	for i, sql := range sqls {
		startTime := time.Now()
		execErr := prodDB.Exec(ctx, sql)
		elapsed := time.Since(startTime)
		results[i].ExecuteTime = fmt.Sprintf("%.3fs", elapsed.Seconds())
		if execErr != nil {
			results[i].ErrorLevel = 2
			results[i].StageStatus = "Execute Failed"
			results[i].ErrorMessage = convertCKError(execErr.Error())
		}
	}

	return results, nil
}

// getCKCreateTable 获取ClickHouse表的CREATE TABLE语句
func getCKCreateTable(ctx context.Context, conn clickhouse.Conn, database, table string) (string, error) {
	rows, err := conn.Query(ctx, fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", database, table))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var createSQL string
		if err := rows.Scan(&createSQL); err != nil {
			return "", err
		}
		return createSQL, nil
	}
	return "", fmt.Errorf("表不存在: %s.%s", database, table)
}

// adaptCKCreateTableForTest 将生产CREATE TABLE适配为测试库使用
// 替换数据库名，移除Replicated引擎参数，避免需要ZooKeeper
func adaptCKCreateTableForTest(createSQL, prodDB, testDB string) string {
	// 替换数据库名
	result := strings.ReplaceAll(createSQL, "`"+prodDB+"`.", "`"+testDB+"`.")
	result = strings.ReplaceAll(result, prodDB+".", testDB+".")

	// Replicated引擎降级：ReplicatedMergeTree -> MergeTree（无需ZooKeeper）
	replicatedEngines := []string{
		"ReplicatedMergeTree", "ReplicatedReplacingMergeTree", "ReplicatedSummingMergeTree",
		"ReplicatedAggregatingMergeTree", "ReplicatedCollapsingMergeTree",
		"ReplicatedVersionedCollapsingMergeTree", "ReplicatedGraphiteMergeTree",
	}
	for _, eng := range replicatedEngines {
		// 替换引擎名并移除第一个括号内的ZooKeeper路径参数（两个字符串参数）
		re := regexp.MustCompile(`(?i)` + eng + `\s*\('[^']*',\s*'[^']*'\s*\)`)
		plainEng := strings.TrimPrefix(eng, "Replicated")
		result = re.ReplaceAllString(result, plainEng+"()")
		// 没有参数的情况
		result = strings.ReplaceAll(result, eng+"()", plainEng+"()")
	}

	return result
}

// replaceCKDatabase 将SQL中的数据库引用替换为临时数据库
func replaceCKDatabase(sql, prodDB, testDB string) string {
	result := strings.ReplaceAll(sql, "`"+prodDB+"`.", "`"+testDB+"`.")
	result = strings.ReplaceAll(result, prodDB+".", testDB+".")
	return result
}

// splitCKSQLs 按分号分割SQL，跳过空语句和注释
func splitCKSQLs(content string) []string {
	parts := strings.Split(content, ";")
	var sqls []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "--") || strings.HasPrefix(lower, "/*") {
			continue
		}
		sqls = append(sqls, trimmed)
	}
	return sqls
}

// convertCKError 将ClickHouse错误码转换为中文说明
func convertCKError(errMsg string) string {
	if strings.Contains(errMsg, "code: 60") || strings.Contains(errMsg, "UNKNOWN_TABLE") {
		return "表不存在: " + errMsg
	}
	if strings.Contains(errMsg, "code: 81") || strings.Contains(errMsg, "DATABASE_NOT_FOUND") {
		return "数据库不存在: " + errMsg
	}
	if strings.Contains(errMsg, "code: 62") || strings.Contains(errMsg, "SYNTAX_ERROR") {
		return "SQL语法错误: " + errMsg
	}
	if strings.Contains(errMsg, "code: 57") || strings.Contains(errMsg, "TABLE_ALREADY_EXISTS") {
		return "表已存在: " + errMsg
	}
	if strings.Contains(errMsg, "code: 56") || strings.Contains(errMsg, "TABLE_DOESNT_EXIST") {
		return "表不存在: " + errMsg
	}
	if strings.Contains(errMsg, "code: 82") || strings.Contains(errMsg, "NOT_IMPLEMENTED") {
		return "操作不支持: " + errMsg
	}
	if strings.Contains(errMsg, "code: 84") || strings.Contains(errMsg, "CANNOT_INSERT_NULL_IN_ORDINARY_COLUMN") {
		return "不能向非NULL字段插入NULL: " + errMsg
	}
	return errMsg
}
