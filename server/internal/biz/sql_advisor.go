package biz

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// AdvisorSuggestion 优化建议
type AdvisorSuggestion struct {
	Level   string `json:"level"`   // info/warning/error
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// SQLAdvisor SQL优化建议引擎
type SQLAdvisor struct{}

func NewSQLAdvisor() *SQLAdvisor {
	return &SQLAdvisor{}
}

// GetAdvisorSuggestions 获取SQL优化建议（结合表结构信息）
func (a *SQLAdvisor) GetAdvisorSuggestions(
	ctx context.Context,
	dbType string,
	sqlText string,
	explainResult string,
	columns []map[string]string,
	indexes string,
) []*AdvisorSuggestion {
	switch strings.ToLower(dbType) {
	case "mysql":
		return a.getMySQLSuggestions(sqlText, explainResult, columns, indexes)
	case "clickhouse":
		return a.getClickHouseSuggestions(sqlText, explainResult, columns)
	default:
		return []*AdvisorSuggestion{{
			Level:   "info",
			Rule:    "unsupported",
			Message: fmt.Sprintf("暂不支持 %s 的自动优化建议", dbType),
		}}
	}
}

// getMySQLSuggestions MySQL优化建议
func (a *SQLAdvisor) getMySQLSuggestions(sqlText, explainResult string, columns []map[string]string, indexes string) []*AdvisorSuggestion {
	var suggestions []*AdvisorSuggestion
	lower := strings.ToLower(strings.TrimSpace(sqlText))

	// 1. SELECT * 检查
	if matched, _ := regexp.MatchString(`select\s+\*`, lower); matched {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "warning",
			Rule:    "no-select-star",
			Message: "避免使用 SELECT *",
			Detail:  "SELECT * 会读取所有列，建议明确指定需要的列，减少网络传输和内存消耗",
		})
	}

	// 2. 无WHERE条件的DML
	if (strings.HasPrefix(lower, "update") || strings.HasPrefix(lower, "delete")) &&
		!strings.Contains(lower, "where") {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "error",
			Rule:    "dml-without-where",
			Message: "UPDATE/DELETE 语句缺少 WHERE 条件",
			Detail:  "缺少 WHERE 条件将影响全表数据，请确认操作范围",
		})
	}

	// 3. LIMIT检查（无ORDER BY的LIMIT）
	if strings.Contains(lower, "limit") && !strings.Contains(lower, "order by") &&
		strings.HasPrefix(lower, "select") {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "info",
			Rule:    "limit-without-orderby",
			Message: "LIMIT 建议配合 ORDER BY 使用",
			Detail:  "无 ORDER BY 的 LIMIT 查询结果顺序不确定，在分页场景中可能导致数据重复或遗漏",
		})
	}

	// 4. LIKE前缀通配符
	if matched, _ := regexp.MatchString(`like\s+'%`, lower); matched {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "warning",
			Rule:    "like-leading-wildcard",
			Message: "LIKE 前缀通配符 '%' 无法使用索引",
			Detail:  "以 '%' 开头的 LIKE 查询会导致全表扫描，考虑使用全文索引或改用前缀匹配",
		})
	}

	// 5. OR条件索引问题
	if strings.Contains(lower, " or ") && strings.HasPrefix(lower, "select") {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "info",
			Rule:    "or-condition",
			Message: "OR 条件可能导致索引失效",
			Detail:  "多个 OR 条件时，MySQL 可能选择全表扫描，考虑改写为 UNION ALL 或使用 IN 条件",
		})
	}

	// 6. 分析EXPLAIN结果中的全表扫描
	if explainResult != "" && strings.Contains(explainResult, `"type":"ALL"`) {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "warning",
			Rule:    "full-table-scan",
			Message: "检测到全表扫描（type=ALL）",
			Detail:  "全表扫描会读取所有数据行，在大表上性能很差。请考虑添加合适的索引",
		})
	}

	// 7. 无索引列（基于表结构判断WHERE列是否有索引）
	if len(columns) > 0 && indexes != "" {
		indexedCols := extractIndexedColumns(indexes)
		whereColsPattern := regexp.MustCompile(`where\s+(\w+)\s*[=<>!]`)
		matches := whereColsPattern.FindAllStringSubmatch(lower, -1)
		for _, m := range matches {
			col := m[1]
			if !indexedCols[col] && col != "id" {
				suggestions = append(suggestions, &AdvisorSuggestion{
					Level:   "warning",
					Rule:    "missing-index",
					Message: fmt.Sprintf("WHERE 条件列 `%s` 缺少索引", col),
					Detail:  fmt.Sprintf("列 `%s` 未建立索引，可能导致全表扫描。建议执行: ALTER TABLE ... ADD INDEX idx_%s (%s)", col, col, col),
				})
			}
		}
	}

	// 8. 检查函数在索引列上
	funcOnIndexPattern := regexp.MustCompile(`where\s+\w+\s*\(\s*(\w+)\s*\)`)
	if funcOnIndexPattern.MatchString(lower) {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "warning",
			Rule:    "func-on-index-column",
			Message: "WHERE 条件中对列使用了函数，导致索引失效",
			Detail:  "将函数从索引列上移除，例如 WHERE DATE(created_at) = '2024-01-01' 改为 WHERE created_at >= '2024-01-01' AND created_at < '2024-01-02'",
		})
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "info",
			Rule:    "ok",
			Message: "SQL 基础检查通过，未发现明显问题",
		})
	}

	return suggestions
}

// getClickHouseSuggestions ClickHouse优化建议
func (a *SQLAdvisor) getClickHouseSuggestions(sqlText, explainResult string, columns []map[string]string) []*AdvisorSuggestion {
	var suggestions []*AdvisorSuggestion
	lower := strings.ToLower(strings.TrimSpace(sqlText))

	// 1. 建议使用PREWHERE代替WHERE（对于MergeTree引擎）
	if strings.Contains(lower, "where") && !strings.Contains(lower, "prewhere") {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "info",
			Rule:    "use-prewhere",
			Message: "考虑将部分 WHERE 条件改为 PREWHERE",
			Detail:  "PREWHERE 在读取全列数据之前先过滤行，对于高选择性的过滤条件可显著减少 I/O。MergeTree 系列引擎支持 PREWHERE，通常用于低基数列的过滤",
		})
	}

	// 2. SELECT * 检查
	if matched, _ := regexp.MatchString(`select\s+\*`, lower); matched {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "warning",
			Rule:    "no-select-star",
			Message: "避免使用 SELECT *",
			Detail:  "ClickHouse 列式存储中，SELECT * 会读取所有列数据，严重影响性能。请只查询需要的列",
		})
	}

	// 3. 无分区键过滤
	if strings.HasPrefix(lower, "select") && strings.Contains(lower, "where") {
		hasDateFilter := strings.Contains(lower, "todate") ||
			strings.Contains(lower, "date_trunc") ||
			regexp.MustCompile(`\d{4}-\d{2}-\d{2}`).MatchString(lower)
		if !hasDateFilter {
			suggestions = append(suggestions, &AdvisorSuggestion{
				Level:   "info",
				Rule:    "partition-pruning",
				Message: "建议在 WHERE 条件中包含分区键（通常是日期列）",
				Detail:  "ClickHouse 按分区存储数据，在 WHERE 中包含分区键可以跳过无关分区，大幅减少扫描量",
			})
		}
	}

	// 4. JOIN优化建议
	if strings.Contains(lower, "join") {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "info",
			Rule:    "join-optimization",
			Message: "JOIN 操作建议：小表放右侧，考虑使用字典替代维度表 JOIN",
			Detail:  "ClickHouse 的 JOIN 默认将右表全部加载到内存，建议：(1)小表放右侧；(2)高频 JOIN 考虑使用字典(DICT)替代；(3)分布式表 JOIN 考虑 GLOBAL JOIN 避免数据放大",
		})
	}

	// 5. COUNT(DISTINCT) 性能
	if strings.Contains(lower, "count(distinct") {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "info",
			Rule:    "count-distinct",
			Message: "COUNT(DISTINCT) 性能较低，考虑使用 uniq() 或 uniqCombined()",
			Detail:  "ClickHouse 的 uniq() 函数使用近似算法，速度比精确的 COUNT(DISTINCT) 快得多，适合大数据量场景",
		})
	}

	// 6. GROUP BY高基数列
	if matched, _ := regexp.MatchString(`group by\s+\w+`, lower); matched {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "info",
			Rule:    "group-by-memory",
			Message: "GROUP BY 高基数列注意内存使用",
			Detail:  "在高基数列上的 GROUP BY 可能消耗大量内存。考虑使用 max_memory_usage 限制，或预先过滤数据范围",
		})
	}

	// 7. 建议采样查询
	if strings.HasPrefix(lower, "select") && !strings.Contains(lower, "sample") {
		rowCount := 0
		for _, col := range columns {
			if col["type"] != "" {
				rowCount++
			}
		}
		if rowCount > 10 {
			suggestions = append(suggestions, &AdvisorSuggestion{
				Level:   "info",
				Rule:    "sampling",
				Message: "对于探索性查询，考虑使用 SAMPLE 子句",
				Detail:  "例如 SELECT * FROM table SAMPLE 0.1 可以只扫描10%的数据，快速验证查询逻辑",
			})
		}
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, &AdvisorSuggestion{
			Level:   "info",
			Rule:    "ok",
			Message: "SQL 基础检查通过，未发现明显优化空间",
		})
	}

	return suggestions
}

// extractIndexedColumns 从SHOW INDEX结果中提取有索引的列名
func extractIndexedColumns(indexesJSON string) map[string]bool {
	indexed := map[string]bool{}
	// 简单解析：找Column_name字段
	re := regexp.MustCompile(`"Column_name":"([^"]+)"`)
	matches := re.FindAllStringSubmatch(indexesJSON, -1)
	for _, m := range matches {
		indexed[strings.ToLower(m[1])] = true
	}
	return indexed
}
