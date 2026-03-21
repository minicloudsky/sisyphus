package biz

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

// QueryExecutor 查询执行器
type QueryExecutor struct {
	log *log.Helper
}

// NewQueryExecutor 创建查询执行器
func NewQueryExecutor(logger log.Logger) *QueryExecutor {
	return &QueryExecutor{
		log: log.NewHelper(logger),
	}
}

// QueryResult 查询结果
type QueryResult struct {
	Columns []string
	Rows    [][]interface{}
	Count   int
	Time    time.Duration
}

// ExecuteMySQLQuery 执行MySQL查询
func (e *QueryExecutor) ExecuteMySQLQuery(ctx context.Context, instance *InstanceInfo, sqlText, database string, limit int) (*QueryResult, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		instance.Username, instance.Password, instance.Host, instance.Port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 设置超时
	db.SetConnMaxLifetime(time.Second * 30)

	// 添加LIMIT限制
	querySQL := sqlText
	if limit > 0 && !strings.Contains(strings.ToUpper(querySQL), "LIMIT") {
		querySQL = fmt.Sprintf("%s LIMIT %d", querySQL, limit)
	}

	start := time.Now()
	rows, err := db.QueryContext(ctx, querySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 读取数据
	var resultRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		resultRows = append(resultRows, values)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("MySQL查询迭代错误: %w", err)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    resultRows,
		Count:   len(resultRows),
		Time:    time.Since(start),
	}, nil
}

// ExecuteRedisQuery 执行Redis查询
func (e *QueryExecutor) ExecuteRedisQuery(ctx context.Context, instance *InstanceInfo, command string) (interface{}, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", instance.Host, instance.Port),
		Password: instance.Password,
		DB:       0,
	})
	defer rdb.Close()

	// 解析Redis命令
	parts := strings.Fields(strings.TrimSpace(command))
	if len(parts) == 0 {
		return nil, fmt.Errorf("无效的Redis命令")
	}

	cmd := strings.ToUpper(parts[0])
	args := parts[1:]

	// 只读命令白名单，禁止写命令
	readOnlyCmds := map[string]bool{
		"GET": true, "MGET": true, "KEYS": true, "SCAN": true,
		"HGET": true, "HGETALL": true, "HMGET": true, "HKEYS": true, "HVALS": true, "HLEN": true,
		"LRANGE": true, "LLEN": true, "LINDEX": true,
		"SMEMBERS": true, "SCARD": true, "SISMEMBER": true, "SUNION": true, "SINTER": true,
		"ZRANGE": true, "ZREVRANGE": true, "ZSCORE": true, "ZRANK": true, "ZCARD": true, "ZRANGEBYSCORE": true,
		"TTL": true, "PTTL": true, "TYPE": true, "EXISTS": true, "STRLEN": true,
		"GETRANGE": true, "GETBIT": true, "BITCOUNT": true,
		"DBSIZE": true, "INFO": true, "PING": true,
	}
	if !readOnlyCmds[cmd] {
		return nil, fmt.Errorf("仅支持只读命令，不允许执行: %s", cmd)
	}

	switch cmd {
	case "GET":
		if len(args) != 1 {
			return nil, fmt.Errorf("GET命令需要一个参数")
		}
		return rdb.Get(ctx, args[0]).Result()
	case "MGET":
		return rdb.MGet(ctx, args...).Result()
	case "KEYS":
		if len(args) != 1 {
			return nil, fmt.Errorf("KEYS命令需要一个参数")
		}
		return rdb.Keys(ctx, args[0]).Result()
	case "TTL":
		if len(args) != 1 {
			return nil, fmt.Errorf("TTL命令需要一个参数")
		}
		return rdb.TTL(ctx, args[0]).Result()
	case "TYPE":
		if len(args) != 1 {
			return nil, fmt.Errorf("TYPE命令需要一个参数")
		}
		return rdb.Type(ctx, args[0]).Result()
	case "EXISTS":
		return rdb.Exists(ctx, args...).Result()
	case "HGET":
		if len(args) != 2 {
			return nil, fmt.Errorf("HGET命令需要两个参数")
		}
		return rdb.HGet(ctx, args[0], args[1]).Result()
	case "HGETALL":
		if len(args) != 1 {
			return nil, fmt.Errorf("HGETALL命令需要一个参数")
		}
		return rdb.HGetAll(ctx, args[0]).Result()
	case "SMEMBERS":
		if len(args) != 1 {
			return nil, fmt.Errorf("SMEMBERS命令需要一个参数")
		}
		return rdb.SMembers(ctx, args[0]).Result()
	case "LRANGE":
		if len(args) != 3 {
			return nil, fmt.Errorf("LRANGE命令需要三个参数")
		}
		start, _ := strconv.Atoi(args[1])
		stop, _ := strconv.Atoi(args[2])
		return rdb.LRange(ctx, args[0], int64(start), int64(stop)).Result()
	case "ZRANGE":
		if len(args) < 3 {
			return nil, fmt.Errorf("ZRANGE命令参数不足")
		}
		start, _ := strconv.Atoi(args[1])
		stop, _ := strconv.Atoi(args[2])
		return rdb.ZRange(ctx, args[0], int64(start), int64(stop)).Result()
	case "DBSIZE":
		return rdb.DBSize(ctx).Result()
	case "INFO":
		if len(args) > 0 {
			return rdb.Info(ctx, args[0]).Result()
		}
		return rdb.Info(ctx).Result()
	case "PING":
		return rdb.Ping(ctx).Result()
	default:
		return nil, fmt.Errorf("不支持的Redis命令: %s", cmd)
	}
}

// ExecuteClickHouseQuery 执行ClickHouse查询
func (e *QueryExecutor) ExecuteClickHouseQuery(ctx context.Context, instance *InstanceInfo, sqlText, database string) (*QueryResult, error) {
	dsn := fmt.Sprintf("clickhouse://%s:%d/%s", instance.Host, instance.Port, database)

	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析ClickHouse DSN失败: %w", err)
	}
	opts.Auth.Username = instance.Username
	opts.Auth.Password = instance.Password
	opts.DialTimeout = 10 * time.Second

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("连接ClickHouse失败: %w", err)
	}
	defer conn.Close()

	start := time.Now()
	rows, err := conn.Query(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("执行ClickHouse查询失败: %w", err)
	}
	defer rows.Close()

	// 获取列信息
	columnTypes := rows.ColumnTypes()
	columns := make([]string, len(columnTypes))
	for i, ct := range columnTypes {
		columns[i] = ct.Name()
	}

	// 读取数据
	var resultRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("读取ClickHouse行数据失败: %w", err)
		}
		resultRows = append(resultRows, values)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ClickHouse查询迭代错误: %w", err)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    resultRows,
		Count:   len(resultRows),
		Time:    time.Since(start),
	}, nil
}
