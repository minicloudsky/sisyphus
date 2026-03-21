package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ProcessInfo 进程信息
type ProcessInfo struct {
	ID       string `json:"id"`
	User     string `json:"user"`
	Host     string `json:"host"`
	DB       string `json:"db"`
	Command  string `json:"command"`
	Time     int64  `json:"time"`
	State    string `json:"state"`
	Info     string `json:"info"`
	QueryID  string `json:"query_id,omitempty"` // ClickHouse query_id
	Progress string `json:"progress,omitempty"` // ClickHouse progress
}

// GetProcessList 获取进程列表（支持MySQL和ClickHouse）
func (uc *DatabaseInstanceUsecase) GetProcessList(ctx context.Context, instance *DatabaseInstance) ([]*ProcessInfo, error) {
	switch instance.DBType {
	case "mysql":
		return uc.getMySQLProcessList(ctx, instance)
	case "clickhouse":
		return uc.getClickHouseProcessList(ctx, instance)
	default:
		return nil, fmt.Errorf("数据库类型 %s 暂不支持进程列表", instance.DBType)
	}
}

// KillProcess 杀死进程/会话
func (uc *DatabaseInstanceUsecase) KillProcess(ctx context.Context, instance *DatabaseInstance, processID string) error {
	switch instance.DBType {
	case "mysql":
		return uc.killMySQLProcess(ctx, instance, processID)
	case "clickhouse":
		return uc.killClickHouseQuery(ctx, instance, processID)
	default:
		return fmt.Errorf("数据库类型 %s 暂不支持会话查杀", instance.DBType)
	}
}

func (uc *DatabaseInstanceUsecase) getMySQLProcessList(ctx context.Context, instance *DatabaseInstance) ([]*ProcessInfo, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/information_schema?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		instance.Username, instance.Password, instance.Host, instance.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 使用information_schema.processlist获取完整SQL信息
	rows, err := db.QueryContext(ctx, `
		SELECT
			CAST(ID AS CHAR) as id,
			USER as user,
			HOST as host,
			COALESCE(DB, '') as db,
			COMMAND as command,
			TIME as time,
			COALESCE(STATE, '') as state,
			COALESCE(INFO, '') as info
		FROM information_schema.PROCESSLIST
		WHERE COMMAND != 'Sleep'
		ORDER BY TIME DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询进程列表失败: %v", err)
	}
	defer rows.Close()

	var processes []*ProcessInfo
	for rows.Next() {
		p := &ProcessInfo{}
		if err := rows.Scan(&p.ID, &p.User, &p.Host, &p.DB, &p.Command, &p.Time, &p.State, &p.Info); err != nil {
			continue
		}
		processes = append(processes, p)
	}
	return processes, nil
}

func (uc *DatabaseInstanceUsecase) killMySQLProcess(ctx context.Context, instance *DatabaseInstance, processID string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		instance.Username, instance.Password, instance.Host, instance.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, fmt.Sprintf("KILL QUERY %s", processID))
	if err != nil {
		return fmt.Errorf("终止进程失败: %v", err)
	}
	return nil
}

func (uc *DatabaseInstanceUsecase) getClickHouseProcessList(ctx context.Context, instance *DatabaseInstance) ([]*ProcessInfo, error) {
	conn, err := ckConn(instance.Host, instance.Port, instance.Username, instance.Password, "system")
	if err != nil {
		return nil, fmt.Errorf("连接ClickHouse失败: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := conn.Query(ctx, `
		SELECT
			query_id,
			user,
			toString(query_start_time) as host,
			currentDatabase() as db,
			'Query' as command,
			toInt64(elapsed) as time,
			'' as state,
			query,
			query_id,
			toString(round(read_rows * 100.0 / if(total_rows_approx > 0, total_rows_approx, 1), 2)) || '%' as progress
		FROM system.processes
		ORDER BY elapsed DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询进程列表失败: %v", err)
	}
	defer rows.Close()

	var processes []*ProcessInfo
	for rows.Next() {
		p := &ProcessInfo{}
		if err := rows.Scan(&p.ID, &p.User, &p.Host, &p.DB, &p.Command, &p.Time, &p.State, &p.Info, &p.QueryID, &p.Progress); err != nil {
			continue
		}
		processes = append(processes, p)
	}
	return processes, nil
}

func (uc *DatabaseInstanceUsecase) killClickHouseQuery(ctx context.Context, instance *DatabaseInstance, queryID string) error {
	conn, err := ckConn(instance.Host, instance.Port, instance.Username, instance.Password, "system")
	if err != nil {
		return fmt.Errorf("连接ClickHouse失败: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = conn.Exec(ctx, fmt.Sprintf("KILL QUERY WHERE query_id = '%s'", queryID))
	if err != nil {
		return fmt.Errorf("终止查询失败: %v", err)
	}
	return nil
}

// GetTableColumns 获取表列结构信息（用于优化建议和查询辅助）
func (uc *DatabaseInstanceUsecase) GetTableColumns(ctx context.Context, instance *DatabaseInstance, database, table string) ([]map[string]string, error) {
	switch instance.DBType {
	case "mysql":
		return uc.getMySQLTableColumns(ctx, instance, database, table)
	case "clickhouse":
		return uc.getClickHouseTableColumns(ctx, instance, database, table)
	default:
		return nil, fmt.Errorf("数据库类型 %s 暂不支持获取表列信息", instance.DBType)
	}
}

func (uc *DatabaseInstanceUsecase) getMySQLTableColumns(ctx context.Context, instance *DatabaseInstance, database, table string) ([]map[string]string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		instance.Username, instance.Password, instance.Host, instance.Port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT, EXTRA, COLUMN_COMMENT
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'
		ORDER BY ORDINAL_POSITION
	`, database, table))
	if err != nil {
		return nil, fmt.Errorf("查询列信息失败: %v", err)
	}
	defer rows.Close()

	var columns []map[string]string
	for rows.Next() {
		var name, colType, nullable, key, defaultVal, extra, comment sql.NullString
		if err := rows.Scan(&name, &colType, &nullable, &key, &defaultVal, &extra, &comment); err != nil {
			continue
		}
		columns = append(columns, map[string]string{
			"name":    name.String,
			"type":    colType.String,
			"null":    nullable.String,
			"key":     key.String,
			"default": defaultVal.String,
			"extra":   extra.String,
			"comment": comment.String,
		})
	}
	return columns, nil
}

func (uc *DatabaseInstanceUsecase) getClickHouseTableColumns(ctx context.Context, instance *DatabaseInstance, database, table string) ([]map[string]string, error) {
	conn, err := ckConn(instance.Host, instance.Port, instance.Username, instance.Password, database)
	if err != nil {
		return nil, fmt.Errorf("连接ClickHouse失败: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := conn.Query(ctx, fmt.Sprintf(`
		SELECT name, type, default_kind, default_expression, comment
		FROM system.columns
		WHERE database = '%s' AND table = '%s'
		ORDER BY position
	`, database, table))
	if err != nil {
		return nil, fmt.Errorf("查询列信息失败: %v", err)
	}
	defer rows.Close()

	var columns []map[string]string
	for rows.Next() {
		var name, colType, defaultKind, defaultExpr, comment string
		if err := rows.Scan(&name, &colType, &defaultKind, &defaultExpr, &comment); err != nil {
			continue
		}
		columns = append(columns, map[string]string{
			"name":    name,
			"type":    colType,
			"default": defaultKind + " " + defaultExpr,
			"comment": comment,
		})
	}
	return columns, nil
}

// GetTableIndexes 获取MySQL表索引信息
func (uc *DatabaseInstanceUsecase) GetTableIndexes(ctx context.Context, instance *DatabaseInstance, database, table string) (string, error) {
	if instance.DBType != "mysql" {
		return "", nil // ClickHouse不需要单独索引
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		instance.Username, instance.Password, instance.Host, instance.Port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, fmt.Sprintf("SHOW INDEX FROM `%s`", table))
	if err != nil {
		return "", fmt.Errorf("查询索引失败: %v", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var indexes []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			if b, ok := vals[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = vals[i]
			}
		}
		indexes = append(indexes, row)
	}
	data, err := json.Marshal(indexes)
	if err != nil {
		return "", fmt.Errorf("序列化失败: %v", err)
	}
	return string(data), nil
}
