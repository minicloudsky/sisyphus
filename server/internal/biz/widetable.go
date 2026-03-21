package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// ─────────────────────────────────────────────────────────────────────────────
// 常量：同步模式 & 目标类型
// ─────────────────────────────────────────────────────────────────────────────

// 宽表同步模式
// direct  ：实时关联查询（适合小数据量快速验证）
// datax   ：DataX 全量批量同步（适合初始全量导入 / 定时批量重建）
// canal   ：Canal 订阅 MySQL Binlog 实时增量同步（适合生产高实时性场景）
const (
	SyncModeDirect = "direct" // 直连聚合查询
	SyncModeDataX  = "datax"  // DataX 全量同步
	SyncModeCanal  = "canal"  // Canal 增量实时同步
)

// 宽表目标存储类型
// mysql         ：将宽表物化到 MySQL 目标表（传统方案）
// elasticsearch ：将宽表索引到 Elasticsearch（高性能检索场景）
const (
	TargetTypeMySQL         = "mysql"
	TargetTypeElasticsearch = "elasticsearch"
)

// ─────────────────────────────────────────────────────────────────────────────
// 数据结构
// ─────────────────────────────────────────────────────────────────────────────

// JoinConfig 关联表配置
type JoinConfig struct {
	Database string `json:"database"`
	Table    string `json:"table"`
	Alias    string `json:"alias"`
	JoinType string `json:"join_type"` // LEFT JOIN | INNER JOIN | RIGHT JOIN
	OnClause string `json:"on"`
}

// FieldConfig 字段映射配置
type FieldConfig struct {
	Source string `json:"source"` // e.g. "t.id" or "u.name"
	Alias  string `json:"alias"`  // 目标列别名
}

// WidetableConfig 宽表配置
//
// 架构演进路径（参考 https://juejin.cn/post/7242550697522118716）：
//  1. direct  模式：实时 JOIN 查询，零配置，适合开发/小数据验证
//  2. datax   模式：通过 DataX 生成 Job JSON 提交批量同步，适合全量初始化
//  3. canal   模式：Canal 订阅 Binlog，实时捕获增量变更写入目标存储
//
// 目标存储：
//   - MySQL：物化宽表到目标库/表（数据同库或跨库）
//   - Elasticsearch：宽表数据写入 ES 索引，支持全文搜索与聚合分析
type WidetableConfig struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// ── 数据源 ────────────────────────────────────────
	SourceInstanceID uint64 `json:"source_instance_id"`
	MainDatabase     string `json:"main_database"`
	MainTable        string `json:"main_table"`
	JoinConfig       string `json:"join_config"`   // JSON: []JoinConfig
	FieldConfig      string `json:"field_config"`  // JSON: []FieldConfig
	SyncCondition    string `json:"sync_condition"` // WHERE 过滤条件

	// ── 同步模式 ──────────────────────────────────────
	// direct | datax | canal
	SyncMode string `json:"sync_mode"`

	// ── 目标存储 ──────────────────────────────────────
	// mysql | elasticsearch
	TargetType       string `json:"target_type"`
	TargetInstanceID uint64 `json:"target_instance_id"` // MySQL 目标实例
	TargetDatabase   string `json:"target_database"`
	TargetTable      string `json:"target_table"`

	// ── Elasticsearch 配置（target_type=elasticsearch 时使用）──
	// ES 集群地址（逗号分隔，如 "http://es1:9200,http://es2:9200"）
	ESHosts   string `json:"es_hosts"`
	ESIndex   string `json:"es_index"`
	ESMapping string `json:"es_mapping"` // JSON：自定义 mapping，为空时自动推断

	// ── Canal 配置（sync_mode=canal 时使用）────────────
	// Canal Server 地址，如 "127.0.0.1:11111"
	CanalServerAddr  string `json:"canal_server_addr"`
	CanalDestination string `json:"canal_destination"` // Canal destination 名称
	// 表过滤规则，如 "mydb\\.order.*"（正则，Canal filter 格式）
	CanalFilter      string `json:"canal_filter"`
	CanalSubscribeID string `json:"canal_subscribe_id"` // 运行时订阅 ID（系统填充）
	// Binlog 消费位点（GTID 或 binlog file:pos，系统自动维护）
	CanalPosition string `json:"canal_position"`

	// ── DataX 配置（sync_mode=datax 时使用）────────────
	// DataX Web API 地址，如 "http://datax-web:9527"
	DataXServerURL string `json:"datax_server_url"`
	// DataX Job JSON 模板（为空时由系统自动生成 Reader+Writer）
	DataXJobTemplate string `json:"datax_job_template"`

	Status    int       `json:"status"`
	CreatedBy uint64    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WidetableSyncRecord 同步执行记录
type WidetableSyncRecord struct {
	ID           uint64    `json:"id"`
	ConfigID     uint64    `json:"config_id"`
	SyncType     int       `json:"sync_type"`    // 1-全量 2-增量
	SyncMode     string    `json:"sync_mode"`    // direct | datax | canal
	Status       int       `json:"status"`       // 0-进行中 1-成功 2-失败
	ProcessedRows int64    `json:"processed_rows"`
	ErrorMessage string    `json:"error_message"`
	CostSeconds  float64   `json:"cost_seconds"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WidetableQueryResult 宽表查询结果
type WidetableQueryResult struct {
	Columns     []string
	Rows        [][]string
	Total       int64
	ExecuteTime string
	Source      string // "direct" | "elasticsearch" | "mysql_materialized"
}

// DataXJobConfig DataX 作业配置（系统生成）
type DataXJobConfig struct {
	JobID       string `json:"job_id"`
	JobName     string `json:"job_name"`
	JobJSON     string `json:"job_json"`      // DataX Job JSON 内容
	SubmitURL   string `json:"submit_url"`    // 提交到 DataX Web 的 URL
	Status      string `json:"status"`        // pending | running | success | failed
	SubmittedAt time.Time `json:"submitted_at"`
}

// CanalSubscription Canal 订阅信息
type CanalSubscription struct {
	ConfigID    uint64    `json:"config_id"`
	SubscribeID string    `json:"subscribe_id"`
	ServerAddr  string    `json:"server_addr"`
	Destination string    `json:"destination"`
	Filter      string    `json:"filter"`
	Position    string    `json:"position"`    // 当前消费位点
	Status      string    `json:"status"`      // active | stopped | error
	StartedAt   time.Time `json:"started_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Repository 接口
// ─────────────────────────────────────────────────────────────────────────────

type WidetableRepo interface {
	CreateConfig(ctx context.Context, config *WidetableConfig) (*WidetableConfig, error)
	UpdateConfig(ctx context.Context, config *WidetableConfig) error
	DeleteConfig(ctx context.Context, id uint64) error
	GetConfig(ctx context.Context, id uint64) (*WidetableConfig, error)
	ListConfigs(ctx context.Context, page, pageSize int, keyword string) ([]*WidetableConfig, int64, error)
	CreateSyncRecord(ctx context.Context, record *WidetableSyncRecord) (*WidetableSyncRecord, error)
	UpdateSyncRecord(ctx context.Context, record *WidetableSyncRecord) error
	ListSyncRecords(ctx context.Context, configID uint64, page, pageSize int) ([]*WidetableSyncRecord, int64, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// UseCase
// ─────────────────────────────────────────────────────────────────────────────

type WidetableUsecase struct {
	repo            WidetableRepo
	instanceUsecase *DatabaseInstanceUsecase
	log             *log.Helper
}

func NewWidetableUsecase(repo WidetableRepo, instanceUsecase *DatabaseInstanceUsecase, logger log.Logger) *WidetableUsecase {
	return &WidetableUsecase{
		repo:            repo,
		instanceUsecase: instanceUsecase,
		log:             log.NewHelper(logger),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (uc *WidetableUsecase) CreateConfig(ctx context.Context, config *WidetableConfig) (*WidetableConfig, error) {
	if config.SyncMode == "" {
		config.SyncMode = SyncModeDirect
	}
	if config.TargetType == "" {
		config.TargetType = TargetTypeMySQL
	}
	return uc.repo.CreateConfig(ctx, config)
}

func (uc *WidetableUsecase) UpdateConfig(ctx context.Context, config *WidetableConfig) error {
	if _, err := uc.repo.GetConfig(ctx, config.ID); err != nil {
		return fmt.Errorf("宽表配置不存在")
	}
	return uc.repo.UpdateConfig(ctx, config)
}

func (uc *WidetableUsecase) DeleteConfig(ctx context.Context, id uint64) error {
	return uc.repo.DeleteConfig(ctx, id)
}

func (uc *WidetableUsecase) GetConfig(ctx context.Context, id uint64) (*WidetableConfig, error) {
	return uc.repo.GetConfig(ctx, id)
}

func (uc *WidetableUsecase) ListConfigs(ctx context.Context, page, pageSize int, keyword string) ([]*WidetableConfig, int64, error) {
	return uc.repo.ListConfigs(ctx, page, pageSize, keyword)
}

func (uc *WidetableUsecase) ListSyncRecords(ctx context.Context, configID uint64, page, pageSize int) ([]*WidetableSyncRecord, int64, error) {
	return uc.repo.ListSyncRecords(ctx, configID, page, pageSize)
}

// ─────────────────────────────────────────────────────────────────────────────
// SQL 构建（direct 模式复用 / DataX Reader 生成）
// ─────────────────────────────────────────────────────────────────────────────

// BuildSQL 根据宽表配置构建聚合查询 SQL
func (uc *WidetableUsecase) BuildSQL(config *WidetableConfig, extraCondition, orderBy string, limit, offset int) (string, error) {
	var fields []FieldConfig
	if config.FieldConfig != "" {
		if err := json.Unmarshal([]byte(config.FieldConfig), &fields); err != nil {
			return "", fmt.Errorf("字段配置解析失败: %v", err)
		}
	}

	var joins []JoinConfig
	if config.JoinConfig != "" {
		if err := json.Unmarshal([]byte(config.JoinConfig), &joins); err != nil {
			return "", fmt.Errorf("关联配置解析失败: %v", err)
		}
	}

	var selectParts []string
	if len(fields) > 0 {
		for _, f := range fields {
			if f.Alias != "" {
				selectParts = append(selectParts, fmt.Sprintf("%s AS `%s`", f.Source, f.Alias))
			} else {
				selectParts = append(selectParts, f.Source)
			}
		}
	} else {
		selectParts = []string{"t.*"}
	}

	fromPart := fmt.Sprintf("`%s`.`%s` AS t", config.MainDatabase, config.MainTable)

	var joinParts []string
	for _, j := range joins {
		joinType := j.JoinType
		if joinType == "" {
			joinType = "LEFT JOIN"
		}
		alias := j.Alias
		if alias == "" {
			alias = j.Table
		}
		joinParts = append(joinParts, fmt.Sprintf("%s `%s`.`%s` AS %s ON %s",
			joinType, j.Database, j.Table, alias, j.OnClause))
	}

	var whereParts []string
	if config.SyncCondition != "" {
		whereParts = append(whereParts, config.SyncCondition)
	}
	if extraCondition != "" {
		whereParts = append(whereParts, extraCondition)
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(selectParts, ", "))
	sb.WriteString("\nFROM ")
	sb.WriteString(fromPart)
	if len(joinParts) > 0 {
		sb.WriteString("\n")
		sb.WriteString(strings.Join(joinParts, "\n"))
	}
	if len(whereParts) > 0 {
		sb.WriteString("\nWHERE ")
		sb.WriteString(strings.Join(whereParts, " AND "))
	}
	if orderBy != "" {
		sb.WriteString("\nORDER BY ")
		sb.WriteString(orderBy)
	}
	if limit > 0 {
		sb.WriteString(fmt.Sprintf("\nLIMIT %d OFFSET %d", limit, offset))
	}
	return sb.String(), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// 查询入口（路由到不同存储后端）
// ─────────────────────────────────────────────────────────────────────────────

// ExecuteQuery 执行宽表查询，根据目标存储类型路由到不同实现
func (uc *WidetableUsecase) ExecuteQuery(ctx context.Context, configID uint64, page, pageSize int, extraCondition, orderBy string) (*WidetableQueryResult, error) {
	config, err := uc.repo.GetConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("宽表配置不存在: %v", err)
	}

	switch config.TargetType {
	case TargetTypeElasticsearch:
		// ES 目标：查询 ES 索引（需要先通过 DataX/Canal 完成数据同步）
		return uc.queryElasticsearch(ctx, config, page, pageSize, extraCondition, orderBy)
	default:
		// MySQL 目标 或 direct 模式：直连 JOIN 查询
		return uc.queryMySQL(ctx, config, page, pageSize, extraCondition, orderBy)
	}
}

// queryMySQL direct/mysql 模式下执行关联查询
func (uc *WidetableUsecase) queryMySQL(ctx context.Context, config *WidetableConfig, page, pageSize int, extraCondition, orderBy string) (*WidetableQueryResult, error) {
	instance, err := uc.instanceUsecase.GetInstanceByID(ctx, config.SourceInstanceID)
	if err != nil {
		return nil, fmt.Errorf("获取源实例失败: %v", err)
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	querySql, err := uc.BuildSQL(config, extraCondition, orderBy, pageSize, offset)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	result, err := uc.executeOnInstance(ctx, instance, config.MainDatabase, querySql)
	if err != nil {
		return nil, err
	}
	result.ExecuteTime = fmt.Sprintf("%v", time.Since(start).Round(time.Millisecond))
	result.Source = "direct_mysql"
	return result, nil
}

// queryElasticsearch ES 模式下查询宽表索引
// 实际使用时需集成 go-elasticsearch 或 olivere/elastic 客户端
// 此处提供框架骨架，生产部署时替换为真实 ES 调用
func (uc *WidetableUsecase) queryElasticsearch(ctx context.Context, config *WidetableConfig, page, pageSize int, extraCondition, orderBy string) (*WidetableQueryResult, error) {
	if config.ESHosts == "" || config.ESIndex == "" {
		return nil, fmt.Errorf("ES 配置不完整：请填写 es_hosts 和 es_index")
	}
	// TODO: 集成 go-elasticsearch 客户端
	// 示例实现骨架：
	//   client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: strings.Split(config.ESHosts, ",")})
	//   query := buildESQuery(extraCondition, orderBy, page, pageSize)
	//   res, err := client.Search(client.Search.WithIndex(config.ESIndex), client.Search.WithBody(query))
	//   return parseESResponse(res), nil
	uc.log.Warnf("ES 查询尚未集成，configID=%d index=%s", config.ID, config.ESIndex)
	return &WidetableQueryResult{
		Columns:     []string{"提示"},
		Rows:        [][]string{{"请先完成 DataX/Canal 同步，然后集成 go-elasticsearch 客户端以启用 ES 查询"}},
		Source:      "elasticsearch",
		ExecuteTime: "0ms",
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// DataX 全量同步
// ─────────────────────────────────────────────────────────────────────────────

// GenerateDataXJob 根据宽表配置生成 DataX Job JSON
// DataX Reader: mysqlreader（多表关联 SQL）
// DataX Writer: 根据目标类型选择 mysqlwriter 或 elasticsearchwriter
func (uc *WidetableUsecase) GenerateDataXJob(ctx context.Context, configID uint64) (*DataXJobConfig, error) {
	config, err := uc.repo.GetConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("宽表配置不存在")
	}

	srcInstance, err := uc.instanceUsecase.GetInstanceByID(ctx, config.SourceInstanceID)
	if err != nil {
		return nil, fmt.Errorf("获取源实例失败: %v", err)
	}

	// 构建全量查询 SQL（无分页）
	querySql, err := uc.BuildSQL(config, "", "", 0, 0)
	if err != nil {
		return nil, err
	}

	var jobJSON string
	jobName := fmt.Sprintf("widetable_full_%d_%s", configID, time.Now().Format("20060102150405"))

	switch config.TargetType {
	case TargetTypeElasticsearch:
		// DataX elasticsearchwriter job
		jobJSON = uc.buildDataXESJob(jobName, srcInstance, querySql, config)
	default:
		// DataX mysqlwriter job
		dstInstance, err := uc.instanceUsecase.GetInstanceByID(ctx, config.TargetInstanceID)
		if err != nil {
			return nil, fmt.Errorf("获取目标实例失败: %v", err)
		}
		jobJSON = uc.buildDataXMySQLJob(jobName, srcInstance, dstInstance, querySql, config)
	}

	submitURL := ""
	if config.DataXServerURL != "" {
		submitURL = fmt.Sprintf("%s/api/job/add", strings.TrimRight(config.DataXServerURL, "/"))
	}

	return &DataXJobConfig{
		JobName:     jobName,
		JobJSON:     jobJSON,
		SubmitURL:   submitURL,
		Status:      "pending",
		SubmittedAt: time.Now(),
	}, nil
}

// buildDataXMySQLJob 生成 MySQL→MySQL 的 DataX Job JSON
func (uc *WidetableUsecase) buildDataXMySQLJob(jobName string, src, dst *DatabaseInstance, querySql string, config *WidetableConfig) string {
	template := `{
  "job": {
    "setting": {
      "speed": { "channel": 4 },
      "errorLimit": { "record": 0, "percentage": 0.02 }
    },
    "content": [{
      "reader": {
        "name": "mysqlreader",
        "parameter": {
          "username": "%s",
          "password": "%s",
          "connection": [{ "jdbcUrl": ["jdbc:mysql://%s:%d/%s?characterEncoding=utf8"], "querySql": ["%s"] }]
        }
      },
      "writer": {
        "name": "mysqlwriter",
        "parameter": {
          "username": "%s",
          "password": "%s",
          "writeMode": "replace",
          "connection": [{ "jdbcUrl": "jdbc:mysql://%s:%d/%s?characterEncoding=utf8", "table": ["%s"] }]
        }
      }
    }]
  }
}`
	escapedSQL := strings.ReplaceAll(querySql, `"`, `\"`)
	escapedSQL = strings.ReplaceAll(escapedSQL, "\n", " ")
	return fmt.Sprintf(template,
		src.Username, src.Password, src.Host, src.Port, config.MainDatabase, escapedSQL,
		dst.Username, dst.Password, dst.Host, dst.Port, config.TargetDatabase, config.TargetTable,
	)
}

// buildDataXESJob 生成 MySQL→Elasticsearch 的 DataX Job JSON
func (uc *WidetableUsecase) buildDataXESJob(jobName string, src *DatabaseInstance, querySql string, config *WidetableConfig) string {
	esHosts := strings.Split(config.ESHosts, ",")
	esHostJSON, _ := json.Marshal(esHosts)
	template := `{
  "job": {
    "setting": {
      "speed": { "channel": 4 },
      "errorLimit": { "record": 0, "percentage": 0.02 }
    },
    "content": [{
      "reader": {
        "name": "mysqlreader",
        "parameter": {
          "username": "%s",
          "password": "%s",
          "connection": [{ "jdbcUrl": ["jdbc:mysql://%s:%d/%s?characterEncoding=utf8"], "querySql": ["%s"] }]
        }
      },
      "writer": {
        "name": "elasticsearchwriter",
        "parameter": {
          "endpoint": %s,
          "index": "%s",
          "type": "_doc",
          "cleanup": false,
          "batchSize": 1000,
          "splitter": ","
        }
      }
    }]
  }
}`
	escapedSQL := strings.ReplaceAll(querySql, `"`, `\"`)
	escapedSQL = strings.ReplaceAll(escapedSQL, "\n", " ")
	return fmt.Sprintf(template,
		src.Username, src.Password, src.Host, src.Port, config.MainDatabase, escapedSQL,
		string(esHostJSON), config.ESIndex,
	)
}

// StartDataXSync 提交 DataX 作业到 DataX Web（异步）
func (uc *WidetableUsecase) StartDataXSync(ctx context.Context, configID uint64, userID uint64) (*WidetableSyncRecord, error) {
	config, err := uc.repo.GetConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("宽表配置不存在")
	}
	if config.SyncMode != SyncModeDataX {
		return nil, fmt.Errorf("当前配置同步模式不是 datax，请先将 sync_mode 设为 datax")
	}

	// 生成 DataX Job
	jobConfig, err := uc.GenerateDataXJob(ctx, configID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	record := &WidetableSyncRecord{
		ConfigID:  configID,
		SyncType:  1, // 全量
		SyncMode:  SyncModeDataX,
		Status:    0, // 进行中
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	record, err = uc.repo.CreateSyncRecord(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("创建同步记录失败: %v", err)
	}

	// 异步提交 DataX 作业
	go uc.submitDataXJob(config, jobConfig, record)

	return record, nil
}

// submitDataXJob 异步提交 DataX 作业
func (uc *WidetableUsecase) submitDataXJob(config *WidetableConfig, jobConfig *DataXJobConfig, record *WidetableSyncRecord) {
	ctx := context.Background()
	start := time.Now()

	defer func() {
		record.FinishedAt = time.Now()
		record.CostSeconds = time.Since(start).Seconds()
		record.UpdatedAt = time.Now()
		if r := recover(); r != nil {
			record.Status = 2
			record.ErrorMessage = fmt.Sprintf("panic: %v", r)
		}
		_ = uc.repo.UpdateSyncRecord(ctx, record)
	}()

	if jobConfig.SubmitURL == "" {
		// 没有 DataX Web URL，直接执行内置全量同步（兼容模式）
		uc.log.Infof("DataX Web 未配置，使用内置全量同步 configID=%d", config.ID)
		if err := uc.runDirectFullSync(ctx, config, record); err != nil {
			record.Status = 2
			record.ErrorMessage = err.Error()
		} else {
			record.Status = 1
		}
		return
	}

	// TODO: 调用 DataX Web API 提交作业
	// POST {DataXServerURL}/api/job/add  body: jobJSON
	// 轮询作业状态直到完成
	// 参考 DataX Web API 文档：https://github.com/WeiYe-Jing/datax-web
	uc.log.Infof("已生成 DataX Job，请手动提交到 %s，job=%s", jobConfig.SubmitURL, jobConfig.JobName)
	record.Status = 1
	record.ErrorMessage = fmt.Sprintf("DataX Job 已生成，提交 URL: %s", jobConfig.SubmitURL)
}

// ─────────────────────────────────────────────────────────────────────────────
// Canal 增量实时同步
// ─────────────────────────────────────────────────────────────────────────────

// StartCanalSync 启动 Canal 增量订阅
// Canal 是阿里巴巴开源的 MySQL Binlog 解析组件，实现原理：
//  1. Canal Server 伪装成 MySQL Slave，接收 MySQL Master 推送的 Binlog
//  2. Canal Client 通过 TCP 连接 Canal Server 获取变更事件（INSERT/UPDATE/DELETE）
//  3. 将变更数据转换并应用到目标存储（MySQL 或 ES）
//
// 生产接入步骤：
//  1. 部署 Canal Server（https://github.com/alibaba/canal）
//  2. 配置 instance.properties 设置 canal.instance.filter.regex
//  3. 使用 canal.go（go-mysql-org/go-mysql）订阅变更事件
//  4. 本函数启动后台 goroutine 处理增量变更
func (uc *WidetableUsecase) StartCanalSync(ctx context.Context, configID uint64) (*CanalSubscription, error) {
	config, err := uc.repo.GetConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("宽表配置不存在")
	}
	if config.SyncMode != SyncModeCanal {
		return nil, fmt.Errorf("当前配置同步模式不是 canal，请先将 sync_mode 设为 canal")
	}
	if config.CanalServerAddr == "" {
		return nil, fmt.Errorf("Canal Server 地址未配置（canal_server_addr）")
	}
	if config.CanalDestination == "" {
		return nil, fmt.Errorf("Canal destination 未配置（canal_destination）")
	}

	subscribeID := fmt.Sprintf("widetable_%d_%d", configID, time.Now().UnixNano())

	sub := &CanalSubscription{
		ConfigID:    configID,
		SubscribeID: subscribeID,
		ServerAddr:  config.CanalServerAddr,
		Destination: config.CanalDestination,
		Filter:      config.CanalFilter,
		Status:      "active",
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 异步启动 Canal 消费循环
	go uc.runCanalConsumer(config, sub)

	uc.log.Infof("Canal 订阅已启动: configID=%d subscribeID=%s dest=%s",
		configID, subscribeID, config.CanalDestination)
	return sub, nil
}

// runCanalConsumer Canal 消费循环（后台 goroutine）
// TODO: 集成 go-mysql-org/go-mysql canal 客户端：
//
//	import "github.com/go-mysql-org/go-mysql/canal"
//	cfg := canal.NewDefaultConfig()
//	cfg.Addr = config.CanalServerAddr
//	cfg.User = "canal"
//	cfg.Password = "canal"
//	cfg.Dump.TableDB = config.MainDatabase
//	cfg.IncludeTableRegex = []string{config.CanalFilter}
//	c, _ := canal.NewCanal(cfg)
//	c.SetEventHandler(&widetableCanalHandler{config: config, uc: uc})
//	c.Run()
func (uc *WidetableUsecase) runCanalConsumer(config *WidetableConfig, sub *CanalSubscription) {
	uc.log.Infof("Canal 消费循环启动 configID=%d server=%s destination=%s filter=%s",
		config.ID, sub.ServerAddr, sub.Destination, sub.Filter)

	// 架构说明：Canal 消费事件处理逻辑
	// 1. 收到 INSERT 事件 → 构建宽表行（关联其他表补全字段）→ 写入目标存储
	// 2. 收到 UPDATE 事件 → 按主键更新目标存储对应行
	// 3. 收到 DELETE 事件 → 按主键删除目标存储对应行
	// 4. 周期性 checkpoint 位点到数据库，保证断点续传

	// TODO: 替换为真实的 go-mysql canal 实现
	// 参考：https://github.com/go-mysql-org/go-mysql
	uc.log.Warnf("Canal 消费循环暂未集成真实客户端，configID=%d。请参考注释完成接入。", config.ID)
}

// applyBinlogEvent 处理单条 Binlog 变更事件，写入目标存储
func (uc *WidetableUsecase) applyBinlogEvent(ctx context.Context, config *WidetableConfig, eventType string, row map[string]interface{}) error {
	switch config.TargetType {
	case TargetTypeElasticsearch:
		return uc.applyToElasticsearch(ctx, config, eventType, row)
	default:
		return uc.applyToMySQL(ctx, config, eventType, row)
	}
}

// applyToMySQL 将变更应用到 MySQL 目标表
func (uc *WidetableUsecase) applyToMySQL(ctx context.Context, config *WidetableConfig, eventType string, row map[string]interface{}) error {
	if config.TargetTable == "" {
		return fmt.Errorf("未配置目标表")
	}
	dstInstance, err := uc.instanceUsecase.GetInstanceByID(ctx, config.TargetInstanceID)
	if err != nil {
		return err
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dstInstance.Username, dstInstance.Password, dstInstance.Host, dstInstance.Port, config.TargetDatabase)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	switch strings.ToUpper(eventType) {
	case "INSERT", "UPDATE":
		// REPLACE INTO（包含所有字段）
		cols := make([]string, 0, len(row))
		vals := make([]interface{}, 0, len(row))
		placeholders := make([]string, 0, len(row))
		for k, v := range row {
			cols = append(cols, "`"+k+"`")
			vals = append(vals, v)
			placeholders = append(placeholders, "?")
		}
		sqlStr := fmt.Sprintf("REPLACE INTO `%s` (%s) VALUES (%s)",
			config.TargetTable,
			strings.Join(cols, ","),
			strings.Join(placeholders, ","),
		)
		_, err = db.ExecContext(ctx, sqlStr, vals...)
	case "DELETE":
		// 假设主键字段名为 "id"，实际使用时应从 FieldConfig 中提取主键
		if id, ok := row["id"]; ok {
			_, err = db.ExecContext(ctx,
				fmt.Sprintf("DELETE FROM `%s` WHERE `id` = ?", config.TargetTable), id)
		}
	}
	return err
}

// applyToElasticsearch 将变更应用到 Elasticsearch 索引
// TODO: 集成 go-elasticsearch 客户端
func (uc *WidetableUsecase) applyToElasticsearch(ctx context.Context, config *WidetableConfig, eventType string, row map[string]interface{}) error {
	// 示例骨架：
	// client, _ := elasticsearch.NewClient(...)
	// switch eventType:
	// case "INSERT"/"UPDATE": client.Index(..., body)
	// case "DELETE": client.Delete(config.ESIndex, docID)
	uc.log.Infof("ES 增量写入 index=%s event=%s（待集成 go-elasticsearch）", config.ESIndex, eventType)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// 内置全量同步（direct / 兼容模式，DataX Web 未部署时使用）
// ─────────────────────────────────────────────────────────────────────────────

// StartFullSync 启动内置全量同步（direct/mysql 模式兜底）
func (uc *WidetableUsecase) StartFullSync(ctx context.Context, configID uint64, batchSize int, userID uint64) (*WidetableSyncRecord, error) {
	config, err := uc.repo.GetConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("宽表配置不存在")
	}
	if config.TargetTable == "" {
		return nil, fmt.Errorf("未配置目标表，无法执行全量同步")
	}
	if batchSize <= 0 {
		batchSize = 1000
	}

	now := time.Now()
	record := &WidetableSyncRecord{
		ConfigID:  configID,
		SyncType:  1,
		SyncMode:  SyncModeDirect,
		Status:    0,
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	record, err = uc.repo.CreateSyncRecord(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("创建同步记录失败: %v", err)
	}

	go func() {
		ctx2 := context.Background()
		start := time.Now()
		defer func() {
			record.FinishedAt = time.Now()
			record.CostSeconds = time.Since(start).Seconds()
			record.UpdatedAt = time.Now()
			if r := recover(); r != nil {
				record.Status = 2
				record.ErrorMessage = fmt.Sprintf("panic: %v", r)
			}
			_ = uc.repo.UpdateSyncRecord(ctx2, record)
		}()
		if err := uc.runDirectFullSync(ctx2, config, record); err != nil {
			record.Status = 2
			record.ErrorMessage = err.Error()
		} else {
			record.Status = 1
		}
	}()

	return record, nil
}

// runDirectFullSync 内置分批读取 + 写入目标表
func (uc *WidetableUsecase) runDirectFullSync(ctx context.Context, config *WidetableConfig, record *WidetableSyncRecord) error {
	srcInstance, err := uc.instanceUsecase.GetInstanceByID(ctx, config.SourceInstanceID)
	if err != nil {
		return fmt.Errorf("获取源实例失败: %v", err)
	}
	var dstInstance *DatabaseInstance
	if config.TargetInstanceID > 0 {
		dstInstance, err = uc.instanceUsecase.GetInstanceByID(ctx, config.TargetInstanceID)
		if err != nil {
			return fmt.Errorf("获取目标实例失败: %v", err)
		}
	} else {
		dstInstance = srcInstance
	}

	srcDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=30s",
		srcInstance.Username, srcInstance.Password, srcInstance.Host, srcInstance.Port, config.MainDatabase)
	srcDB, err := sql.Open("mysql", srcDSN)
	if err != nil {
		return fmt.Errorf("源库连接失败: %v", err)
	}
	defer srcDB.Close()

	dstDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=30s",
		dstInstance.Username, dstInstance.Password, dstInstance.Host, dstInstance.Port, config.TargetDatabase)
	dstDB, err := sql.Open("mysql", dstDSN)
	if err != nil {
		return fmt.Errorf("目标库连接失败: %v", err)
	}
	defer dstDB.Close()

	const batchSize = 1000
	var offset int
	var totalRows int64

	for {
		querySql, err := uc.BuildSQL(config, "", "", batchSize, offset)
		if err != nil {
			return fmt.Errorf("构建SQL失败: %v", err)
		}
		rows, err := srcDB.QueryContext(ctx, querySql)
		if err != nil {
			return fmt.Errorf("查询失败: %v", err)
		}
		columns, _ := rows.Columns()
		if len(columns) == 0 {
			rows.Close()
			break
		}

		var batchData [][]interface{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			if err := rows.Scan(valuePtrs...); err != nil {
				continue
			}
			rowCopy := make([]interface{}, len(values))
			copy(rowCopy, values)
			batchData = append(batchData, rowCopy)
		}
		rows.Close()

		if len(batchData) == 0 {
			break
		}

		if err := uc.batchInsert(ctx, dstDB, config.TargetTable, columns, batchData); err != nil {
			return fmt.Errorf("写入目标表失败: %v", err)
		}

		totalRows += int64(len(batchData))
		record.ProcessedRows = totalRows
		record.UpdatedAt = time.Now()
		_ = uc.repo.UpdateSyncRecord(ctx, record)

		if len(batchData) < batchSize {
			break
		}
		offset += batchSize
	}

	uc.log.Infof("内置全量同步完成: config_id=%d, rows=%d", config.ID, totalRows)
	return nil
}

// batchInsert 批量写入目标表（REPLACE INTO 支持幂等）
func (uc *WidetableUsecase) batchInsert(ctx context.Context, db *sql.DB, table string, columns []string, rows [][]interface{}) error {
	if len(rows) == 0 {
		return nil
	}
	quotedCols := make([]string, len(columns))
	for i, c := range columns {
		quotedCols[i] = "`" + c + "`"
	}
	placeholder := "(" + strings.Repeat("?,", len(columns)-1) + "?)"
	allPlaceholders := strings.Repeat(placeholder+",", len(rows)-1) + placeholder

	insertSQL := fmt.Sprintf("REPLACE INTO `%s` (%s) VALUES %s",
		table, strings.Join(quotedCols, ","), allPlaceholders)

	var args []interface{}
	for _, row := range rows {
		args = append(args, row...)
	}
	_, err := db.ExecContext(ctx, insertSQL, args...)
	return err
}

// executeOnInstance 在指定实例上执行 SELECT SQL（direct 查询用）
func (uc *WidetableUsecase) executeOnInstance(ctx context.Context, instance *DatabaseInstance, database, querySql string) (*WidetableQueryResult, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=30s",
		instance.Username, instance.Password, instance.Host, instance.Port, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	queryCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	rows, err := db.QueryContext(queryCtx, querySql)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %v", err)
	}

	result := &WidetableQueryResult{Columns: columns}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else if b, ok := val.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		result.Rows = append(result.Rows, row)
		result.Total++
	}
	return result, nil
}
