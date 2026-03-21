package biz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ============================================================
// HBase SQL 解析层
// ============================================================

// HBaseQueryType 查询操作类型
type HBaseQueryType int

const (
	HBaseQueryGet      HBaseQueryType = iota // WHERE pk = 'val'
	HBaseQueryGetMulti                       // WHERE pk IN ('v1','v2')
	HBaseQueryScan                           // scan / range
)

// HBaseCondition 辅助列过滤条件（对应 SingleColumnValueFilter）
type HBaseCondition struct {
	Family    string
	Qualifier string
	Op        string // =, !=, >, <, >=, <=
	Value     string
}

// HBaseQuery 解析后的 HBase 查询
type HBaseQuery struct {
	Table      string
	Columns    []string       // 空=SELECT *, 否则是 cf:qualifier 列表
	QueryType  HBaseQueryType
	RowKey     string         // GET 单行
	RowKeys    []string       // GET 多行
	StartRow   string         // SCAN 起始 rowkey
	StopRow    string         // SCAN 终止 rowkey
	Conditions []HBaseCondition
	Limit      int
}

var (
	reSelect    = regexp.MustCompile(`(?i)SELECT\s+(.+?)\s+FROM\s+(\S+)`)
	reWhere     = regexp.MustCompile(`(?i)\s+WHERE\s+(.+?)(?:\s+LIMIT\s+\d+)?$`)
	reLimit     = regexp.MustCompile(`(?i)\s+LIMIT\s+(\d+)`)
	rePkEq      = regexp.MustCompile(`(?i)(?:pk|rowkey|row_key)\s*=\s*'([^']*)'`)
	rePkIn      = regexp.MustCompile(`(?i)(?:pk|rowkey|row_key)\s+IN\s*\(([^)]+)\)`)
	rePkRange   = regexp.MustCompile(`(?i)(?:pk|rowkey|row_key)\s*(>=|>)\s*'([^']*)'`)
	rePkRangeE  = regexp.MustCompile(`(?i)(?:pk|rowkey|row_key)\s*(<=|<)\s*'([^']*)'`)
	reColFilter = regexp.MustCompile(`(?i)(\w+:\w+|\w+)\s*(=|!=|>=|<=|>|<)\s*'([^']*)'`)
	reInVal     = regexp.MustCompile(`'([^']*)'`)
)

// ParseHBaseSQL 将简化版 SQL 解析为 HBaseQuery
//
// 支持的语法示例：
//   SELECT * FROM user_table WHERE rowkey = 'user_001'
//   SELECT cf1:name, cf1:age FROM user_table WHERE rowkey IN ('u1','u2','u3')
//   SELECT * FROM order_table WHERE rowkey >= 'ord_20240101' AND rowkey <= 'ord_20240201' LIMIT 100
//   SELECT * FROM logs WHERE rowkey >= 'log_2024' AND cf:level = 'ERROR' LIMIT 50
func ParseHBaseSQL(sql string) (*HBaseQuery, error) {
	q := &HBaseQuery{Limit: 1000}

	// 提取 LIMIT
	if m := reLimit.FindStringSubmatch(sql); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &q.Limit)
	}

	// 提取 SELECT ... FROM ...
	m := reSelect.FindStringSubmatch(sql)
	if len(m) < 3 {
		return nil, fmt.Errorf("无效的 HBase SQL 语法，缺少 SELECT ... FROM 子句")
	}
	colStr := strings.TrimSpace(m[1])
	q.Table = strings.TrimSpace(m[2])

	if colStr != "*" {
		for _, c := range strings.Split(colStr, ",") {
			q.Columns = append(q.Columns, strings.TrimSpace(c))
		}
	}

	// 提取 WHERE 子句
	whereSQL := sql
	if idx := strings.Index(strings.ToUpper(sql), " WHERE "); idx >= 0 {
		whereSQL = sql[idx+7:]
		// 去掉末尾 LIMIT
		if lm := reLimit.FindStringIndex(strings.ToUpper(whereSQL)); lm != nil {
			whereSQL = whereSQL[:lm[0]]
		}
		whereSQL = strings.TrimSpace(whereSQL)
	} else {
		// 无 WHERE 子句 → 全表扫描
		q.QueryType = HBaseQueryScan
		return q, nil
	}

	// pk = 'value'
	if m := rePkEq.FindStringSubmatch(whereSQL); len(m) > 1 {
		q.QueryType = HBaseQueryGet
		q.RowKey = m[1]
		return q, nil
	}

	// pk IN ('v1','v2')
	if m := rePkIn.FindStringSubmatch(whereSQL); len(m) > 1 {
		q.QueryType = HBaseQueryGetMulti
		for _, vm := range reInVal.FindAllStringSubmatch(m[1], -1) {
			q.RowKeys = append(q.RowKeys, vm[1])
		}
		return q, nil
	}

	// rowkey >= 'start' [ AND rowkey <= 'end' ] [ AND cf:col = 'val' ]
	q.QueryType = HBaseQueryScan

	if m := rePkRange.FindStringSubmatch(whereSQL); len(m) > 2 {
		q.StartRow = m[2]
	}
	if m := rePkRangeE.FindStringSubmatch(whereSQL); len(m) > 2 {
		q.StopRow = m[2]
	}

	// 辅助列过滤条件（排除 pk/rowkey 自身的条件）
	for _, cm := range reColFilter.FindAllStringSubmatch(whereSQL, -1) {
		col := cm[1]
		colLower := strings.ToLower(col)
		if colLower == "pk" || colLower == "rowkey" || colLower == "row_key" {
			continue
		}
		// 支持 cf:qualifier 和裸 qualifier 两种写法
		fam, qual := col, col
		if parts := strings.SplitN(col, ":", 2); len(parts) == 2 {
			fam, qual = parts[0], parts[1]
		} else {
			fam = col
		}
		q.Conditions = append(q.Conditions, HBaseCondition{
			Family:    fam,
			Qualifier: qual,
			Op:        cm[2],
			Value:     cm[3],
		})
	}

	return q, nil
}

// ============================================================
// HBase REST 客户端
// ============================================================

// hbaseRow HBase REST API 返回的单行数据
type hbaseRow struct {
	Key  string      `json:"key"`
	Cell []hbaseCell `json:"Cell"`
}

// hbaseCell HBase REST API 返回的单个 cell
type hbaseCell struct {
	Column    string `json:"column"`
	Timestamp int64  `json:"timestamp"`
	Value     string `json:"$"`
}

// hbaseRESTResponse HBase REST API 顶层响应
type hbaseRESTResponse struct {
	Row []hbaseRow `json:"Row"`
}

// HBaseRESTExecutor 通过 HBase REST API 执行查询
type HBaseRESTExecutor struct {
	client *http.Client
}

// NewHBaseRESTExecutor 创建执行器
func NewHBaseRESTExecutor() *HBaseRESTExecutor {
	return &HBaseRESTExecutor{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *HBaseRESTExecutor) baseURL(instance *InstanceInfo) string {
	return fmt.Sprintf("http://%s:%d", instance.Host, instance.Port)
}

func (e *HBaseRESTExecutor) doGet(ctx context.Context, url, basicUser, basicPass string) ([]hbaseRow, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if basicUser != "" {
		req.SetBasicAuth(basicUser, basicPass)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HBase REST 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // 行不存在
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HBase REST 响应错误 %d: %s", resp.StatusCode, string(body))
	}

	var result hbaseRESTResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 HBase REST 响应失败: %w", err)
	}
	return result.Row, nil
}

// scannerConfig HBase REST scanner 配置（XML）
type scannerConfig struct {
	XMLName  xml.Name `xml:"Scanner"`
	Batch    int      `xml:"batch,attr"`
	StartRow string   `xml:"startRow,omitempty"`
	EndRow   string   `xml:"endRow,omitempty"`
	Filter   string   `xml:"filter,omitempty"`
}

// buildFilter 将 HBaseCondition 列表转为 HBase JSON filter 字符串
func buildFilter(conds []HBaseCondition) string {
	if len(conds) == 0 {
		return ""
	}

	// 操作符映射
	opMap := map[string]string{
		"=":  "EQUAL",
		"!=": "NOT_EQUAL",
		">":  "GREATER",
		">=": "GREATER_OR_EQUAL",
		"<":  "LESS",
		"<=": "LESS_OR_EQUAL",
	}

	type filterJSON struct {
		Type       string `json:"type"`
		Op         string `json:"op"`
		Family     string `json:"family"`
		Qualifier  string `json:"qualifier"`
		Comparator struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"comparator"`
		FilterIfMissing bool `json:"filterIfMissing"`
	}

	if len(conds) == 1 {
		c := conds[0]
		f := filterJSON{
			Type:      "SingleColumnValueFilter",
			Op:        opMap[c.Op],
			Family:    base64.StdEncoding.EncodeToString([]byte(c.Family)),
			Qualifier: base64.StdEncoding.EncodeToString([]byte(c.Qualifier)),
			FilterIfMissing: false,
		}
		f.Comparator.Type = "BinaryComparator"
		f.Comparator.Value = base64.StdEncoding.EncodeToString([]byte(c.Value))
		b, _ := json.Marshal(f)
		return string(b)
	}

	// 多条件用 FilterList
	type listJSON struct {
		Type    string        `json:"type"`
		Op      string        `json:"op"`
		Filters []interface{} `json:"filters"`
	}
	list := listJSON{Type: "FilterList", Op: "MUST_PASS_ALL"}
	for _, c := range conds {
		f := filterJSON{
			Type:      "SingleColumnValueFilter",
			Op:        opMap[c.Op],
			Family:    base64.StdEncoding.EncodeToString([]byte(c.Family)),
			Qualifier: base64.StdEncoding.EncodeToString([]byte(c.Qualifier)),
			FilterIfMissing: false,
		}
		f.Comparator.Type = "BinaryComparator"
		f.Comparator.Value = base64.StdEncoding.EncodeToString([]byte(c.Value))
		list.Filters = append(list.Filters, f)
	}
	b, _ := json.Marshal(list)
	return string(b)
}

func (e *HBaseRESTExecutor) scanRows(ctx context.Context, instance *InstanceInfo, q *HBaseQuery) ([]hbaseRow, error) {
	base := e.baseURL(instance)

	cfg := scannerConfig{Batch: q.Limit}
	if q.StartRow != "" {
		cfg.StartRow = base64.StdEncoding.EncodeToString([]byte(q.StartRow))
	}
	if q.StopRow != "" {
		cfg.EndRow = base64.StdEncoding.EncodeToString([]byte(q.StopRow))
	}
	cfg.Filter = buildFilter(q.Conditions)

	xmlBody, err := xml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("序列化 scanner 配置失败: %w", err)
	}

	// 1. 创建 scanner
	scanURL := fmt.Sprintf("%s/%s/scanner", base, q.Table)
	scanReq, err := http.NewRequestWithContext(ctx, http.MethodPut, scanURL, strings.NewReader(string(xmlBody)))
	if err != nil {
		return nil, err
	}
	scanReq.Header.Set("Content-Type", "text/xml")
	scanReq.Header.Set("Accept", "text/plain")
	if instance.Username != "" {
		scanReq.SetBasicAuth(instance.Username, instance.Password)
	}

	scanResp, err := e.client.Do(scanReq)
	if err != nil {
		return nil, fmt.Errorf("创建 HBase scanner 失败: %w", err)
	}
	defer scanResp.Body.Close()
	if scanResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(scanResp.Body)
		return nil, fmt.Errorf("创建 HBase scanner 失败 %d: %s", scanResp.StatusCode, string(body))
	}

	// scanner URL 在 Location header 中
	scannerURL := scanResp.Header.Get("Location")
	if scannerURL == "" {
		return nil, fmt.Errorf("HBase REST 未返回 scanner Location")
	}
	defer func() {
		delReq, _ := http.NewRequestWithContext(ctx, http.MethodDelete, scannerURL, nil)
		if instance.Username != "" {
			delReq.SetBasicAuth(instance.Username, instance.Password)
		}
		e.client.Do(delReq) //nolint:errcheck
	}()

	// 2. 读取数据（分批次直到 204 No Content）
	var allRows []hbaseRow
	for len(allRows) < q.Limit {
		getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, scannerURL, nil)
		if err != nil {
			return nil, err
		}
		getReq.Header.Set("Accept", "application/json")
		if instance.Username != "" {
			getReq.SetBasicAuth(instance.Username, instance.Password)
		}

		getResp, err := e.client.Do(getReq)
		if err != nil {
			return nil, fmt.Errorf("读取 HBase scanner 数据失败: %w", err)
		}

		if getResp.StatusCode == http.StatusNoContent {
			getResp.Body.Close()
			break // 数据已读完
		}
		if getResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(getResp.Body)
			getResp.Body.Close()
			return nil, fmt.Errorf("读取 HBase scanner 数据错误 %d: %s", getResp.StatusCode, string(body))
		}

		var batch hbaseRESTResponse
		if err := json.NewDecoder(getResp.Body).Decode(&batch); err != nil {
			getResp.Body.Close()
			return nil, fmt.Errorf("解析 scanner 批次数据失败: %w", err)
		}
		getResp.Body.Close()

		allRows = append(allRows, batch.Row...)
		if len(batch.Row) == 0 {
			break
		}
	}

	return allRows, nil
}

// ============================================================
// 结果转换
// ============================================================

func decodeBase64(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s // 若不是 base64 则原样返回
	}
	return string(b)
}

// rowsToQueryResult 将 hbaseRow 列表转换为通用 QueryResult
func rowsToQueryResult(rows []hbaseRow, selectCols []string) *QueryResult {
	if len(rows) == 0 {
		return &QueryResult{
			Columns: []string{"rowkey"},
			Rows:    nil,
			Count:   0,
		}
	}

	// 收集所有出现的 column 名（family:qualifier），保持顺序
	colSet := make(map[string]struct{})
	var colOrder []string
	for _, row := range rows {
		for _, cell := range row.Cell {
			col := decodeBase64(cell.Column)
			if _, exists := colSet[col]; !exists {
				colSet[col] = struct{}{}
				colOrder = append(colOrder, col)
			}
		}
	}

	// 如果指定了 SELECT 列，则按指定顺序过滤
	var finalCols []string
	if len(selectCols) > 0 {
		for _, c := range selectCols {
			finalCols = append(finalCols, c)
		}
	} else {
		finalCols = colOrder
	}

	columns := append([]string{"rowkey"}, finalCols...)
	colIdx := make(map[string]int, len(finalCols))
	for i, c := range finalCols {
		colIdx[c] = i + 1 // +1 因为第0列是 rowkey
	}

	var resultRows [][]interface{}
	for _, row := range rows {
		record := make([]interface{}, len(columns))
		record[0] = decodeBase64(row.Key)
		for _, cell := range row.Cell {
			col := decodeBase64(cell.Column)
			if idx, ok := colIdx[col]; ok {
				record[idx] = decodeBase64(cell.Value)
			}
		}
		resultRows = append(resultRows, record)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    resultRows,
		Count:   len(resultRows),
	}
}

// ============================================================
// 主执行入口
// ============================================================

// ExecuteHBaseQuery 解析 SQL 并通过 HBase REST API 执行查询
func (e *QueryExecutor) ExecuteHBaseQuery(ctx context.Context, instance *InstanceInfo, sqlText string, limit int) (*QueryResult, error) {
	if limit <= 0 {
		limit = 1000
	}

	q, err := ParseHBaseSQL(sqlText)
	if err != nil {
		return nil, err
	}
	if limit < q.Limit {
		q.Limit = limit
	}

	restExec := NewHBaseRESTExecutor()
	start := time.Now()

	var rows []hbaseRow

	switch q.QueryType {
	case HBaseQueryGet:
		url := fmt.Sprintf("%s/%s/%s", restExec.baseURL(instance), q.Table, q.RowKey)
		rows, err = restExec.doGet(ctx, url, instance.Username, instance.Password)
		if err != nil {
			return nil, err
		}

	case HBaseQueryGetMulti:
		// 并发获取多个 rowkey
		for _, rk := range q.RowKeys {
			url := fmt.Sprintf("%s/%s/%s", restExec.baseURL(instance), q.Table, rk)
			r, e := restExec.doGet(ctx, url, instance.Username, instance.Password)
			if e != nil {
				return nil, e
			}
			rows = append(rows, r...)
		}

	case HBaseQueryScan:
		rows, err = restExec.scanRows(ctx, instance, q)
		if err != nil {
			return nil, err
		}
	}

	result := rowsToQueryResult(rows, q.Columns)
	result.Time = time.Since(start)
	return result, nil
}
