package biz

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// GoInceptionClient goInception客户端
type GoInceptionClient struct {
	inceptionHost     string // goInception服务地址
	inceptionPort     int    // goInception服务端口
	inceptionUsername string // goInception服务用户名
	inceptionPassword string // goInception服务密码
	targetHost        string // 目标数据库地址
	targetPort        int    // 目标数据库端口
	targetUsername    string // 目标数据库用户名
	targetPassword    string // 目标数据库密码
}

// SetInceptionCredentials 设置goInception服务的用户名和密码
func (c *GoInceptionClient) SetInceptionCredentials(username, password string) {
	c.inceptionUsername = username
	c.inceptionPassword = password
}

// NewGoInceptionClient 创建goInception客户端
// inceptionHost: goInception服务地址
// inceptionPort: goInception服务端口
// targetHost: 目标数据库地址
// targetPort: 目标数据库端口
// targetUsername: 目标数据库用户名
// targetPassword: 目标数据库密码
func NewGoInceptionClient(inceptionHost string, inceptionPort int, targetHost string, targetPort int, targetUsername, targetPassword string) *GoInceptionClient {
	return &GoInceptionClient{
		inceptionHost:     inceptionHost,
		inceptionPort:     inceptionPort,
		inceptionUsername: "root", // goInception默认用户名
		inceptionPassword: "",     // goInception默认密码
		targetHost:        targetHost,
		targetPort:        targetPort,
		targetUsername:    targetUsername,
		targetPassword:    targetPassword,
	}
}

// generateStatements 生成goInception SQL语句
func (c *GoInceptionClient) generateStatements(sqlText, database, sqlType string, needBackup bool) string {
	// 移除SQL末尾的分号
	sqlText = strings.TrimSpace(sqlText)
	if strings.HasSuffix(sqlText, ";") || strings.HasSuffix(sqlText, "；") {
		sqlText = sqlText[:len(sqlText)-1]
		sqlText = strings.TrimSpace(sqlText)
	}

	var backupStr string
	if needBackup {
		backupStr = "--backup=1"
	} else {
		backupStr = "--backup=0"
	}

	// 构建goInception SQL，参考Python项目的generate_statements方法
	// 注意：Python代码中，SQL语句后面有分号和空格，然后是inception_magic_commit
	// 格式：inception_magic_start; use `db`; SQL语句; inception_magic_commit;
	// 仅在不需要备份时禁用远程备份，需要备份时让goInception连接backup_host写入回滚SQL
	var remoteBackupStr string
	if !needBackup {
		remoteBackupStr = "--enable-remote-backup=0;"
	}
	inceptionSQL := fmt.Sprintf(`/*--user=%s;--password=%s;--host=%s;--port=%d;%s;%s;--db=%s;%s*/ inception_magic_start; use `+"`%s`"+`; %s; inception_magic_commit;`,
		c.targetUsername,
		c.targetPassword,
		c.targetHost,
		c.targetPort,
		sqlType,
		backupStr,
		database,
		remoteBackupStr,
		database,
		sqlText)

	// 添加调试日志，输出完整的SQL格式
	fmt.Printf("[GoInception generateStatements] Generated SQL length: %d\n", len(inceptionSQL))
	fmt.Printf("[GoInception generateStatements] SQL ends with commit: %v\n", strings.HasSuffix(inceptionSQL, "inception_magic_commit;"))

	return inceptionSQL
}

// ReviewSQL 审核SQL
func (c *GoInceptionClient) ReviewSQL(ctx context.Context, sqlText, database string) ([]*SQLReviewResult, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4", c.inceptionUsername, c.inceptionPassword, c.inceptionHost, c.inceptionPort)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 设置超时
	db.SetConnMaxLifetime(time.Second * 30)

	// 构建审核SQL（--check=1表示只审核不执行）
	reviewSQL := c.generateStatements(sqlText, database, "--check=1", false)

	rows, err := db.QueryContext(ctx, reviewSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SQLReviewResult
	columns, _ := rows.Columns()
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

	// 保存原始SQL，用于fallback
	originalSQL := sqlText
	
	// 调试：打印所有列名和原始SQL
	fmt.Printf("[GoInception] DEBUG: Columns returned: %v\n", columns)
	fmt.Printf("[GoInception] DEBUG: Original SQL: %s\n", originalSQL)

	for rows.Next() {
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		result := &SQLReviewResult{}
		for i, col := range columns {
			val := values[i]
			var strVal string
			if b, ok := val.([]byte); ok {
				strVal = string(b)
			} else if val != nil {
				strVal = fmt.Sprintf("%v", val)
			}

			// 调试：打印所有字段的值（用于排查）
			colLower := strings.ToLower(col)
			fmt.Printf("[GoInception] DEBUG: col=%s (lower=%s), value=%s\n", col, colLower, strVal)

			// 使用不区分大小写的列名匹配
			switch colLower {
			case "order_id", "id":
				result.OrderID = strVal
			case "stage":
				result.Stage = strVal
			case "error_level", "errlevel":
				if strVal == "0" {
					result.ErrorLevel = 0
				} else if strVal == "1" {
					result.ErrorLevel = 1
				} else {
					result.ErrorLevel = 2
				}
			case "stage_status", "stagestatus":
				result.StageStatus = strVal
			case "error_message", "errormessage":
				result.ErrorMessage = strVal
			case "sqltext", "sql":
				result.SQLText = strVal
			case "affected_rows", "affectedrows":
				result.AffectedRows = parseInt(strVal)
			case "backup_dbname", "backupdbname":
				result.BackupDBName = strVal
			case "execute_time", "executetime":
				result.ExecuteTime = strVal
			case "sqlhash", "sqlsha1":
				result.SQLHash = strVal
			case "sequence":
				result.Sequence = strVal
			}
		}
		
		// 调试：打印解析后的结果
		fmt.Printf("[GoInception] DEBUG: Parsed result - SQLText=%s, StageStatus=%s, ErrorLevel=%d\n", 
			result.SQLText, result.StageStatus, result.ErrorLevel)
		
		// 如果SQL为空，使用原始SQL作为fallback
		if result.SQLText == "" {
			fmt.Printf("[GoInception] DEBUG: SQLText is empty, using original SQL as fallback\n")
			result.SQLText = originalSQL
		}
		
		results = append(results, result)
	}

	return results, nil
}

// ExecuteSQL 执行SQL
func (c *GoInceptionClient) ExecuteSQL(ctx context.Context, sqlText, database string, needBackup bool) ([]*SQLReviewResult, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4", c.inceptionUsername, c.inceptionPassword, c.inceptionHost, c.inceptionPort)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	db.SetConnMaxLifetime(time.Second * 300) // 执行可能较慢，设置5分钟超时

	// 构建执行SQL（--execute=1表示执行，--check=0表示不审核）
	executeSQL := c.generateStatements(sqlText, database, "--execute=1", needBackup)
	
	// 添加调试日志
	fmt.Printf("[GoInception ExecuteSQL] SQL: %s\n", executeSQL)

	// goInception执行SQL应该使用Query，因为它返回结果集
	// 注意：即使使用--execute=1，goInception也会返回执行结果
	// 重要：必须使用Query而不是Exec，因为goInception返回结果集
	// 同时必须完全读取所有结果，否则事务可能不会提交
	rows, err := db.QueryContext(ctx, executeSQL)
	if err != nil {
		fmt.Printf("[GoInception ExecuteSQL] Query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	
	// 重要：goInception需要读取所有结果，否则事务可能不会提交
	// 确保rows被完全读取，这是goInception事务提交的关键

	var results []*SQLReviewResult
	columns, _ := rows.Columns()
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

		for rows.Next() {
			for i := range columns {
				valuePtrs[i] = &values[i]
			}
			if err := rows.Scan(valuePtrs...); err != nil {
				fmt.Printf("[GoInception ExecuteSQL] Scan error: %v\n", err)
				return nil, err
			}

			result := &SQLReviewResult{}
			for i, col := range columns {
				val := values[i]
				var strVal string
				if b, ok := val.([]byte); ok {
					strVal = string(b)
				} else if val != nil {
					strVal = fmt.Sprintf("%v", val)
				}

				switch col {
				case "order_id":
					result.OrderID = strVal
				case "stage":
					result.Stage = strVal
				case "error_level":
					if strVal == "0" {
						result.ErrorLevel = 0
					} else if strVal == "1" {
						result.ErrorLevel = 1
					} else {
						result.ErrorLevel = 2
					}
				case "stage_status":
					result.StageStatus = strVal
				case "error_message":
					result.ErrorMessage = strVal
				case "sqltext", "sql": // 支持两种列名
					result.SQLText = strVal
				case "affected_rows":
					result.AffectedRows = parseInt(strVal)
				case "backup_dbname":
					result.BackupDBName = strVal
				case "execute_time":
					result.ExecuteTime = strVal
				case "sqlhash", "sqlsha1":
					result.SQLHash = strVal
				case "sequence":
					result.Sequence = strVal
				}
			}
			// 添加调试日志
			fmt.Printf("[GoInception ExecuteSQL] Result[%d]: order_id=%s, stage=%s, error_level=%d, stage_status=%s, affected_rows=%d, error_message=%s, sqltext=%s\n",
				len(results), result.OrderID, result.Stage, result.ErrorLevel, result.StageStatus, result.AffectedRows, result.ErrorMessage, result.SQLText)
			results = append(results, result)
		}

	return results, nil
}

// SQLReviewResult SQL审核结果
type SQLReviewResult struct {
	OrderID      string
	Stage        string
	ErrorLevel   int // 0-正常, 1-警告, 2-错误
	StageStatus  string
	ErrorMessage string
	SQLText      string
	AffectedRows int
	BackupDBName string
	ExecuteTime  string
	SQLHash      string
	Sequence     string // opid_time，用于从备份库查找回滚SQL
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// SplitSQL 拆分SQL语句
func SplitSQL(sqlText string) []string {
	// 简单的SQL拆分，按分号分割
	parts := strings.Split(sqlText, ";")
	var results []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && !strings.HasPrefix(strings.ToLower(part), "inception") {
			results = append(results, part+";")
		}
	}
	return results
}
