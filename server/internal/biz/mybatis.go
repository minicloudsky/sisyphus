package biz

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-kratos/kratos/v2/log"
)

// MyBatisGenerator MyBatis代码生成器（本地实现）
type MyBatisGenerator struct {
	log *log.Helper
}

// NewMyBatisGenerator 创建MyBatis代码生成器
func NewMyBatisGenerator(apiURL string, logger log.Logger) *MyBatisGenerator {
	return &MyBatisGenerator{
		log: log.NewHelper(logger),
	}
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name       string
	Type       string
	Comment    string
	IsPrimary  bool
	IsAutoInc  bool
	IsNullable bool
}

// GenerateXML 从DDL生成MyBatis XML Mapper
func (g *MyBatisGenerator) GenerateXML(ddl string) (string, error) {
	tableName, columns, err := parseDDL(ddl)
	if err != nil {
		return "", fmt.Errorf("解析DDL失败: %w", err)
	}
	if tableName == "" || len(columns) == 0 {
		return "", fmt.Errorf("无法从DDL中提取表名或列信息")
	}

	entityName := toCamelCase(tableName)
	mapperName := entityName + "Mapper"
	namespace := "com.example.mapper." + mapperName

	// Build column list
	colNames := make([]string, 0, len(columns))
	for _, c := range columns {
		colNames = append(colNames, "`"+c.Name+"`")
	}
	columnList := strings.Join(colNames, ", ")

	// Build result map
	var resultMapItems strings.Builder
	for _, c := range columns {
		field := toCamelCase(c.Name)
		tag := "result"
		if c.IsPrimary {
			tag = "id"
		}
		resultMapItems.WriteString(fmt.Sprintf("        <%s column=\"%s\" property=\"%s\"/>\n", tag, c.Name, field))
	}

	// Build insert columns/values (exclude auto-increment primary key)
	var insertCols, insertVals []string
	for _, c := range columns {
		if c.IsPrimary && c.IsAutoInc {
			continue
		}
		insertCols = append(insertCols, "`"+c.Name+"`")
		insertVals = append(insertVals, "#{"+toCamelCase(c.Name)+"}")
	}

	// Build update set clause (exclude primary key)
	var updateSet []string
	for _, c := range columns {
		if c.IsPrimary {
			continue
		}
		updateSet = append(updateSet, fmt.Sprintf("`%s` = #{%s}", c.Name, toCamelCase(c.Name)))
	}

	// Find primary key column
	pkCol := "id"
	pkField := "id"
	for _, c := range columns {
		if c.IsPrimary {
			pkCol = c.Name
			pkField = toCamelCase(c.Name)
			break
		}
	}

	bt := "`"
	xmlLines := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"`,
		`    "http://mybatis.org/dtd/mybatis-3-mapper.dtd">`,
		``,
		fmt.Sprintf(`<mapper namespace="%s">`, namespace),
		``,
		`    <!-- 结果映射 -->`,
		fmt.Sprintf(`    <resultMap id="BaseResultMap" type="%s">`, entityName),
		resultMapItems.String() + `    </resultMap>`,
		``,
		`    <!-- 基础列 -->`,
		`    <sql id="Base_Column_List">`,
		`        ` + columnList,
		`    </sql>`,
		``,
		`    <!-- 查询所有 -->`,
		`    <select id="selectAll" resultMap="BaseResultMap">`,
		`        SELECT <include refid="Base_Column_List"/>`,
		fmt.Sprintf(`        FROM %s%s%s`, bt, tableName, bt),
		`    </select>`,
		``,
		`    <!-- 根据主键查询 -->`,
		`    <select id="selectById" parameterType="java.lang.Long" resultMap="BaseResultMap">`,
		`        SELECT <include refid="Base_Column_List"/>`,
		fmt.Sprintf(`        FROM %s%s%s`, bt, tableName, bt),
		fmt.Sprintf(`        WHERE %s%s%s = #{%s}`, bt, pkCol, bt, pkField),
		`    </select>`,
		``,
		`    <!-- 插入 -->`,
		fmt.Sprintf(`    <insert id="insert" parameterType="%s" useGeneratedKeys="true" keyProperty="%s">`, entityName, pkField),
		fmt.Sprintf(`        INSERT INTO %s%s%s (%s)`, bt, tableName, bt, strings.Join(insertCols, ", ")),
		fmt.Sprintf(`        VALUES (%s)`, strings.Join(insertVals, ", ")),
		`    </insert>`,
		``,
		`    <!-- 更新 -->`,
		fmt.Sprintf(`    <update id="updateById" parameterType="%s">`, entityName),
		fmt.Sprintf(`        UPDATE %s%s%s`, bt, tableName, bt),
		`        <set>`,
		`            ` + strings.Join(updateSet, ",\n            "),
		`        </set>`,
		fmt.Sprintf(`        WHERE %s%s%s = #{%s}`, bt, pkCol, bt, pkField),
		`    </update>`,
		``,
		`    <!-- 删除 -->`,
		`    <delete id="deleteById" parameterType="java.lang.Long">`,
		fmt.Sprintf(`        DELETE FROM %s%s%s`, bt, tableName, bt),
		fmt.Sprintf(`        WHERE %s%s%s = #{%s}`, bt, pkCol, bt, pkField),
		`    </delete>`,
		``,
		`</mapper>`,
	}
	xml := strings.Join(xmlLines, "\n")

	return xml, nil
}

// parseDDL 解析 CREATE TABLE DDL，返回表名和列信息
func parseDDL(ddl string) (string, []ColumnInfo, error) {
	ddl = strings.TrimSpace(ddl)

	// 提取表名
	tableRe := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?[` + "`" + `"']?(\w+)[` + "`" + `"']?`)
	tableMatch := tableRe.FindStringSubmatch(ddl)
	if len(tableMatch) < 2 {
		return "", nil, fmt.Errorf("未找到 CREATE TABLE 语句")
	}
	tableName := tableMatch[1]

	// 提取列定义区域（括号内）
	start := strings.Index(ddl, "(")
	end := strings.LastIndex(ddl, ")")
	if start < 0 || end < 0 || end <= start {
		return tableName, nil, fmt.Errorf("括号不完整")
	}
	body := ddl[start+1 : end]

	// 提取主键
	pkRe := regexp.MustCompile(`(?i)PRIMARY\s+KEY\s*\([` + "`" + `"']?(\w+)[` + "`" + `"']?\)`)
	pkMatch := pkRe.FindStringSubmatch(body)
	pkCol := ""
	if len(pkMatch) >= 2 {
		pkCol = pkMatch[1]
	}

	// 逐行解析列定义
	lines := splitColumnLines(body)
	var columns []ColumnInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lineLower := strings.ToLower(line)

		// 跳过约束行
		if strings.HasPrefix(lineLower, "primary key") ||
			strings.HasPrefix(lineLower, "unique") ||
			strings.HasPrefix(lineLower, "key ") ||
			strings.HasPrefix(lineLower, "index ") ||
			strings.HasPrefix(lineLower, "constraint") {
			continue
		}

		col, ok := parseColumnLine(line)
		if !ok {
			continue
		}
		if pkCol != "" && strings.EqualFold(col.Name, pkCol) {
			col.IsPrimary = true
		}
		columns = append(columns, col)
	}

	return tableName, columns, nil
}

// splitColumnLines 按逗号分割列定义行（处理括号嵌套）
func splitColumnLines(body string) []string {
	var lines []string
	depth := 0
	var cur strings.Builder
	for _, ch := range body {
		switch ch {
		case '(':
			depth++
			cur.WriteRune(ch)
		case ')':
			depth--
			cur.WriteRune(ch)
		case ',':
			if depth == 0 {
				lines = append(lines, cur.String())
				cur.Reset()
			} else {
				cur.WriteRune(ch)
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		lines = append(lines, s)
	}
	return lines
}

// parseColumnLine 解析单行列定义
func parseColumnLine(line string) (ColumnInfo, bool) {
	// 去掉反引号
	line = strings.ReplaceAll(line, "`", "")
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return ColumnInfo{}, false
	}

	col := ColumnInfo{Name: parts[0]}
	col.Type = parts[1]

	lineLower := strings.ToLower(line)
	if strings.Contains(lineLower, "primary key") {
		col.IsPrimary = true
	}
	if strings.Contains(lineLower, "auto_increment") {
		col.IsAutoInc = true
	}
	if strings.Contains(lineLower, "not null") {
		col.IsNullable = false
	} else {
		col.IsNullable = true
	}

	// 提取 COMMENT
	commentRe := regexp.MustCompile(`(?i)COMMENT\s+'([^']*)'`)
	if m := commentRe.FindStringSubmatch(line); len(m) >= 2 {
		col.Comment = m[1]
	}

	return col, true
}

// toCamelCase 将 snake_case 转换为 CamelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			b.WriteString(p)
		} else {
			runes := []rune(p)
			runes[0] = unicode.ToUpper(runes[0])
			b.WriteString(string(runes))
		}
	}
	return b.String()
}
