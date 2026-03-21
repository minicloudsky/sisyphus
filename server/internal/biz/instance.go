package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

// DatabaseInstanceUsecase 数据库实例用例
type DatabaseInstanceUsecase struct {
	repo DatabaseInstanceRepo
}

// NewDatabaseInstanceUsecase 创建数据库实例用例
func NewDatabaseInstanceUsecase(repo DatabaseInstanceRepo) *DatabaseInstanceUsecase {
	return &DatabaseInstanceUsecase{repo: repo}
}

// DatabaseInstance 数据库实例实体
type DatabaseInstance struct {
	ID          uint64
	Name        string
	DBType      string // mysql, redis, clickhouse等
	Host        string
	Port        int
	Username    string
	Password    string // 加密存储
	Env         string // dev, test, prod
	Region      string
	Status      int // 1-正常, 0-下线
	Owner       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DatabaseInstanceRepo 数据库实例仓储接口
type DatabaseInstanceRepo interface {
	Create(ctx context.Context, instance *DatabaseInstance) (*DatabaseInstance, error)
	Update(ctx context.Context, instance *DatabaseInstance) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*DatabaseInstance, error)
	// restrictInstanceIDs 为 true 时仅返回 allowedInstanceIDs 中的实例；allowed 为空则返回空列表
	List(ctx context.Context, page, pageSize int, dbType *string, env *string, restrictInstanceIDs bool, allowedInstanceIDs []uint64) ([]*DatabaseInstance, int64, error)
}

// CreateInstance 创建实例
func (uc *DatabaseInstanceUsecase) CreateInstance(ctx context.Context, instance *DatabaseInstance) (*DatabaseInstance, error) {
	return uc.repo.Create(ctx, instance)
}

// UpdateInstance 更新实例
func (uc *DatabaseInstanceUsecase) UpdateInstance(ctx context.Context, instance *DatabaseInstance) error {
	_, err := uc.repo.GetByID(ctx, instance.ID)
	if err != nil {
		return fmt.Errorf("实例不存在")
	}

	return uc.repo.Update(ctx, instance)
}

// DeleteInstance 删除实例
func (uc *DatabaseInstanceUsecase) DeleteInstance(ctx context.Context, id uint64) error {
	_, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("实例不存在")
	}

	return uc.repo.Delete(ctx, id)
}

// TestConnection 测试连接
func (uc *DatabaseInstanceUsecase) TestConnection(ctx context.Context, instance *DatabaseInstance) error {
	switch instance.DBType {
	case "mysql":
		return uc.testMySQLConnection(ctx, instance)
	case "redis":
		return uc.testRedisConnection(ctx, instance)
	case "hbase":
		return uc.testHBaseConnection(ctx, instance)
	default:
		return fmt.Errorf("不支持的数据库类型: %s", instance.DBType)
	}
}

// testMySQLConnection 测试MySQL连接
func (uc *DatabaseInstanceUsecase) testMySQLConnection(ctx context.Context, instance *DatabaseInstance) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		instance.Username, instance.Password, instance.Host, instance.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping失败: %v", err)
	}

	return nil
}

// testRedisConnection 测试Redis连接
func (uc *DatabaseInstanceUsecase) testRedisConnection(ctx context.Context, instance *DatabaseInstance) error {
	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", instance.Host, instance.Port),
		Password: instance.Password,
		DB:       0, // 默认使用DB 0
	})
	defer rdb.Close()

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 测试连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis连接失败: %v", err)
	}

	return nil
}

// testHBaseConnection 测试HBase REST连接
func (uc *DatabaseInstanceUsecase) testHBaseConnection(ctx context.Context, instance *DatabaseInstance) error {
	url := fmt.Sprintf("http://%s:%d/version/cluster", instance.Host, instance.Port)
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("构建HBase REST请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	if instance.Username != "" {
		req.SetBasicAuth(instance.Username, instance.Password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HBase REST连接失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HBase REST连接失败，HTTP状态码: %d", resp.StatusCode)
	}
	return nil
}

// GetInstanceByID 根据ID获取实例
func (uc *DatabaseInstanceUsecase) GetInstanceByID(ctx context.Context, id uint64) (*DatabaseInstance, error) {
	return uc.repo.GetByID(ctx, id)
}

// ListInstances 分页列出实例；restrictInstanceIDs 为 true 时仅返回 allowedInstanceIDs 中的实例（空切片则结果为空）
func (uc *DatabaseInstanceUsecase) ListInstances(ctx context.Context, page, pageSize int, dbType *string, env *string, restrictInstanceIDs bool, allowedInstanceIDs []uint64) ([]*DatabaseInstance, int64, error) {
	return uc.repo.List(ctx, page, pageSize, dbType, env, restrictInstanceIDs, allowedInstanceIDs)
}

// ListDatabases 获取数据库列表
func (uc *DatabaseInstanceUsecase) ListDatabases(ctx context.Context, instance *DatabaseInstance) ([]string, error) {
	switch instance.DBType {
	case "mysql":
		return uc.listMySQLDatabases(ctx, instance)
	case "clickhouse":
		return uc.listClickHouseDatabases(ctx, instance)
	default:
		return nil, fmt.Errorf("数据库类型 %s 暂不支持列出数据库", instance.DBType)
	}
}

func (uc *DatabaseInstanceUsecase) listMySQLDatabases(ctx context.Context, instance *DatabaseInstance) ([]string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		instance.Username, instance.Password, instance.Host, instance.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			continue
		}
		if dbName != "information_schema" && dbName != "performance_schema" && dbName != "mysql" && dbName != "sys" {
			databases = append(databases, dbName)
		}
	}
	return databases, nil
}

func (uc *DatabaseInstanceUsecase) listClickHouseDatabases(ctx context.Context, instance *DatabaseInstance) ([]string, error) {
	conn, err := ckConn(instance.Host, instance.Port, instance.Username, instance.Password, "system")
	if err != nil {
		return nil, fmt.Errorf("连接ClickHouse失败: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := conn.Query(ctx, "SELECT name FROM system.databases WHERE name NOT IN ('system','information_schema','INFORMATION_SCHEMA') ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			continue
		}
		databases = append(databases, dbName)
	}
	return databases, nil
}

// ListTables 获取表列表
func (uc *DatabaseInstanceUsecase) ListTables(ctx context.Context, instance *DatabaseInstance, database string) ([]string, error) {
	switch instance.DBType {
	case "mysql":
		return uc.listMySQLTables(ctx, instance, database)
	case "clickhouse":
		return uc.listClickHouseTables(ctx, instance, database)
	default:
		return nil, fmt.Errorf("数据库类型 %s 暂不支持列出表", instance.DBType)
	}
}

func (uc *DatabaseInstanceUsecase) listMySQLTables(ctx context.Context, instance *DatabaseInstance, database string) ([]string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		instance.Username, instance.Password, instance.Host, instance.Port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

func (uc *DatabaseInstanceUsecase) listClickHouseTables(ctx context.Context, instance *DatabaseInstance, database string) ([]string, error) {
	conn, err := ckConn(instance.Host, instance.Port, instance.Username, instance.Password, database)
	if err != nil {
		return nil, fmt.Errorf("连接ClickHouse失败: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := conn.Query(ctx, fmt.Sprintf("SELECT name FROM system.tables WHERE database = '%s' ORDER BY name", database))
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

// GetTableDDL 获取表DDL
func (uc *DatabaseInstanceUsecase) GetTableDDL(ctx context.Context, instance *DatabaseInstance, database, table string) (string, error) {
	switch instance.DBType {
	case "mysql":
		return uc.getMySQLTableDDL(ctx, instance, database, table)
	case "clickhouse":
		return uc.getClickHouseTableDDL(ctx, instance, database, table)
	default:
		return "", fmt.Errorf("数据库类型 %s 暂不支持获取DDL", instance.DBType)
	}
}

func (uc *DatabaseInstanceUsecase) getMySQLTableDDL(ctx context.Context, instance *DatabaseInstance, database, table string) (string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local&timeout=5s",
		instance.Username, instance.Password, instance.Host, instance.Port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := db.Conn(ctx)
	if err != nil {
		return "", fmt.Errorf("获取连接失败: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return "", fmt.Errorf("设置字符集失败: %v", err)
	}

	var tableName, ddlBytes []byte
	err = conn.QueryRowContext(ctx, fmt.Sprintf("SHOW CREATE TABLE `%s`", table)).Scan(&tableName, &ddlBytes)
	if err != nil {
		return "", fmt.Errorf("获取DDL失败: %v", err)
	}
	return string(ddlBytes), nil
}

func (uc *DatabaseInstanceUsecase) getClickHouseTableDDL(ctx context.Context, instance *DatabaseInstance, database, table string) (string, error) {
	conn, err := ckConn(instance.Host, instance.Port, instance.Username, instance.Password, database)
	if err != nil {
		return "", fmt.Errorf("连接ClickHouse失败: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := conn.Query(ctx, fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", database, table))
	if err != nil {
		return "", fmt.Errorf("获取DDL失败: %v", err)
	}
	defer rows.Close()

	if rows.Next() {
		var ddl string
		if err := rows.Scan(&ddl); err != nil {
			return "", fmt.Errorf("读取DDL失败: %v", err)
		}
		return ddl, nil
	}
	return "", fmt.Errorf("表不存在: %s.%s", database, table)
}

// GetExplain 获取执行计划
func (uc *DatabaseInstanceUsecase) GetExplain(ctx context.Context, instance *DatabaseInstance, database, sqlText string) (string, error) {
	switch instance.DBType {
	case "mysql":
		return uc.getMySQLExplain(ctx, instance, database, sqlText)
	case "clickhouse":
		return uc.getClickHouseExplain(ctx, instance, database, sqlText)
	default:
		return "", fmt.Errorf("数据库类型 %s 暂不支持执行计划", instance.DBType)
	}
}

func (uc *DatabaseInstanceUsecase) getMySQLExplain(ctx context.Context, instance *DatabaseInstance, database, sqlText string) (string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
		instance.Username, instance.Password, instance.Host, instance.Port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	explainSQL := fmt.Sprintf("EXPLAIN %s", sqlText)
	rows, err := db.QueryContext(ctx, explainSQL)
	if err != nil {
		return "", fmt.Errorf("获取执行计划失败: %v", err)
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	jsonData, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("序列化失败: %v", err)
	}
	return string(jsonData), nil
}

func (uc *DatabaseInstanceUsecase) getClickHouseExplain(ctx context.Context, instance *DatabaseInstance, database, sqlText string) (string, error) {
	conn, err := ckConn(instance.Host, instance.Port, instance.Username, instance.Password, database)
	if err != nil {
		return "", fmt.Errorf("连接ClickHouse失败: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// ClickHouse EXPLAIN支持多种模式，默认返回PLAN
	rows, err := conn.Query(ctx, fmt.Sprintf("EXPLAIN %s", sqlText))
	if err != nil {
		return "", fmt.Errorf("获取执行计划失败: %v", err)
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			continue
		}
		lines = append(lines, line)
	}

	jsonData, err := json.Marshal(map[string]interface{}{"plan": lines})
	if err != nil {
		return "", fmt.Errorf("序列化失败: %v", err)
	}
	return string(jsonData), nil
}
