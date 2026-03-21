package biz

import "strings"

// GoInceptionStageStatus goInception状态常量
type GoInceptionStageStatus struct {
	Text  string // 中文说明
	Color string // 颜色标识（用于前端显示）
}

// GetGoInceptionStatusText 获取goInception状态的中文说明
// 根据goInception返回的stage_status值，转换为用户可读的中文状态
func GetGoInceptionStatusText(stageStatus string) GoInceptionStageStatus {
	// 转换为小写进行不区分大小写的匹配
	statusLower := strings.ToLower(stageStatus)
	
	// goInception常见状态值映射（不区分大小写）
	statusMap := map[string]GoInceptionStageStatus{
		// 审核相关状态
		"audit completed":     {Text: "审核完成", Color: "success"},
		"checked":              {Text: "已审核", Color: "success"},
		"checking":             {Text: "审核中", Color: "processing"},
		
		// 执行相关状态
		"execute successfully": {Text: "执行成功", Color: "success"},
		"executed":             {Text: "已执行", Color: "success"},
		"executing":             {Text: "执行中", Color: "processing"},
		
		// 备份相关状态
		"backup":               {Text: "备份中", Color: "processing"},
		"backuped":             {Text: "已备份", Color: "success"},
		
		// 回滚相关状态
		"rollback":             {Text: "回滚中", Color: "warning"},
		"rollbacked":           {Text: "已回滚", Color: "default"},
		
		// 错误相关状态
		"failed":               {Text: "执行失败", Color: "error"},
		"error":                {Text: "执行错误", Color: "error"},
		"execute failed":       {Text: "执行失败", Color: "error"},
		
		// 其他状态
		"pending":              {Text: "待处理", Color: "default"},
		"waiting":              {Text: "等待中", Color: "default"},
	}
	
	// 如果找到映射，返回对应的中文说明
	if mapped, ok := statusMap[statusLower]; ok {
		return mapped
	}
	
	// 如果状态为空或未找到映射，返回默认值
	if stageStatus == "" {
		return GoInceptionStageStatus{Text: "未知状态", Color: "default"}
	}
	
	// 返回原始状态（可能包含其他未定义的状态值）
	return GoInceptionStageStatus{Text: stageStatus, Color: "default"}
}
