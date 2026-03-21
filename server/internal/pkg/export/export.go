package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportData 导出数据
type ExportData struct {
	Columns []string
	Rows    [][]interface{}
}

// ExportToExcel 导出为Excel
func ExportToExcel(data *ExportData, filePath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Sheet1"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}
	f.SetActiveSheet(index)

	// 写入表头
	for i, col := range data.Columns {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, col)
	}

	// 写入数据
	for rowIdx, row := range data.Rows {
		for colIdx, val := range row {
			cell := fmt.Sprintf("%c%d", 'A'+colIdx, rowIdx+2)
			// 将值转换为字符串，避免大数值科学计数法
			strVal := fmt.Sprintf("%v", val)
			if strVal == "<nil>" {
				strVal = ""
			}
			f.SetCellValue(sheetName, cell, strVal)
		}
	}

	return f.SaveAs(filePath)
}

// ExportToCSV 导出为CSV
func ExportToCSV(data *ExportData, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	if err := writer.Write(data.Columns); err != nil {
		return err
	}

	// 写入数据
	for _, row := range data.Rows {
		record := make([]string, len(row))
		for i, val := range row {
			record[i] = fmt.Sprintf("%v", val)
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportToJSON 导出为JSON
func ExportToJSON(data *ExportData, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 转换为JSON格式
	records := make([]map[string]interface{}, len(data.Rows))
	for i, row := range data.Rows {
		record := make(map[string]interface{})
		for j, col := range data.Columns {
			if j < len(row) {
				record[col] = row[j]
			}
		}
		records[i] = record
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

// GenerateExportFileName 生成导出文件名
func GenerateExportFileName(format string) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("export_%s.%s", timestamp, format)
}

// GetExportDir 获取导出目录
func GetExportDir() string {
	// 默认导出目录
	exportDir := "./exports"
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		// 如果创建失败，使用临时目录
		exportDir = os.TempDir()
	}
	return exportDir
}

// GetExportFilePath 获取导出文件完整路径
func GetExportFilePath(format string) string {
	dir := GetExportDir()
	filename := GenerateExportFileName(format)
	return filepath.Join(dir, filename)
}
