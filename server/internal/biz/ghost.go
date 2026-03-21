package biz

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// GhOstManager gh-ost管理器
type GhOstManager struct {
	socketDir string
	logDir    string
	panicDir  string
	log       *log.Helper
}

// NewGhOstManager 创建gh-ost管理器
func NewGhOstManager(socketDir, logDir, panicDir string, logger log.Logger) *GhOstManager {
	return &GhOstManager{
		socketDir: strings.TrimSuffix(socketDir, "/"),
		logDir:    strings.TrimSuffix(logDir, "/"),
		panicDir:  strings.TrimSuffix(panicDir, "/"),
		log:       log.NewHelper(logger),
	}
}

// GetSocketPath 获取socket文件路径
func (m *GhOstManager) GetSocketPath(tableName, workID string) string {
	return fmt.Sprintf("%s/%s_%s.sock", m.socketDir, tableName, workID)
}

// GetLogPath 获取日志文件路径
func (m *GhOstManager) GetLogPath(tableName, workID string) string {
	return fmt.Sprintf("%s/%s_%s.log", m.logDir, tableName, workID)
}

// GetPanicPath 获取panic文件路径
func (m *GhOstManager) GetPanicPath(tableName, workID string) string {
	return fmt.Sprintf("%s/%s_%s.panic.flag", m.panicDir, tableName, workID)
}

// GhOstProgress gh-ost执行进度
type GhOstProgress struct {
	Percent   int    // 进度百分比 0-100
	Consuming string // 预计剩余时间
	Backlog   string // Backlog队列
	Status    string // 状态: migrating, throttled
	LogName   string // 日志文件路径
}

// GetProgress 获取gh-ost执行进度
func (m *GhOstManager) GetProgress(ctx context.Context, workID string) (*GhOstProgress, error) {
	// 1. 查找日志文件
	logFiles, err := filepath.Glob(fmt.Sprintf("%s/*_%s.log", m.logDir, workID))
	if err != nil {
		return nil, fmt.Errorf("查找日志文件失败: %v", err)
	}
	if len(logFiles) == 0 {
		return nil, fmt.Errorf("未找到日志文件: work_id=%s", workID)
	}

	// 获取最新的日志文件
	var latestLog string
	var latestTime time.Time
	for _, logFile := range logFiles {
		info, err := os.Stat(logFile)
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestLog = logFile
		}
	}

	if latestLog == "" {
		return nil, fmt.Errorf("未找到有效的日志文件")
	}

	// 2. 从日志文件中提取表名
	fileName := filepath.Base(latestLog)
	tableName := strings.TrimSuffix(fileName, fmt.Sprintf("_%s.log", workID))

	// 3. 解析日志文件获取进度信息
	progress, err := m.parseLogFile(ctx, latestLog)
	if err != nil {
		return nil, err
	}
	progress.LogName = latestLog

	// 4. 检查是否暂停（throttled）
	status, err := m.checkThrottled(ctx, tableName, workID)
	if err == nil {
		progress.Status = status
	} else {
		progress.Status = "migrating"
	}

	return progress, nil
}

// parseLogFile 解析日志文件获取进度信息
func (m *GhOstManager) parseLogFile(ctx context.Context, logFile string) (*GhOstProgress, error) {
	// 使用tail和grep获取最后一行包含Copy:和Applied的行
	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("tail -10000 %s | grep 'Copy:' | grep 'Applied' | tail -1", logFile))
	output, err := cmd.Output()
	if err != nil {
		return &GhOstProgress{}, nil // 如果没找到，返回空进度
	}

	line := strings.TrimSpace(string(output))
	progress := &GhOstProgress{}

	// 检查是否暂停
	if strings.Contains(line, "throttled") {
		progress.Status = "throttled"
	} else {
		progress.Status = "migrating"
	}

	// 解析进度百分比
	percentRe := regexp.MustCompile(`(\d+\.?\d*)%`)
	matches := percentRe.FindStringSubmatch(line)
	if len(matches) > 1 {
		if p, err := strconv.ParseFloat(matches[1], 64); err == nil {
			progress.Percent = int(p)
		}
	}

	// 解析Backlog
	backlogRe := regexp.MustCompile(`Backlog:\s*(\d+/\d+)`)
	matches = backlogRe.FindStringSubmatch(line)
	if len(matches) > 1 {
		progress.Backlog = matches[1]
	}

	// 解析ETA
	etaRe := regexp.MustCompile(`ETA:\s*([^;]+)`)
	matches = etaRe.FindStringSubmatch(line)
	if len(matches) > 1 {
		eta := strings.TrimSpace(matches[1])
		if eta == "due" {
			progress.Consuming = "消费增量binlog中"
		} else {
			progress.Consuming = eta
		}
	}

	return progress, nil
}

// checkThrottled 检查是否暂停
func (m *GhOstManager) checkThrottled(ctx context.Context, tableName, workID string) (string, error) {
	socketPath := m.GetSocketPath(tableName, workID)
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return "unknown", err
	}

	// 尝试读取socket状态（这里简化处理，实际可以通过socket获取更详细的状态）
	// 由于socket通信较复杂，这里先通过日志判断
	return "migrating", nil
}

// Throttle 暂停gh-ost执行
func (m *GhOstManager) Throttle(ctx context.Context, tableName, workID string) error {
	socketPath := m.GetSocketPath(tableName, workID)
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return fmt.Errorf("socket文件不存在: %s", socketPath)
	}

	// 使用socat发送throttle命令
	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("echo throttle | socat - %s", socketPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("暂停失败: %v, 输出: %s", err, string(output))
	}

	m.log.Infof("gh-ost暂停成功: table=%s, work_id=%s", tableName, workID)
	return nil
}

// NoThrottle 恢复gh-ost执行
func (m *GhOstManager) NoThrottle(ctx context.Context, tableName, workID string) error {
	socketPath := m.GetSocketPath(tableName, workID)
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return fmt.Errorf("socket文件不存在: %s", socketPath)
	}

	// 使用socat发送no-throttle命令
	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("echo no-throttle | socat - %s", socketPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("恢复失败: %v, 输出: %s", err, string(output))
	}

	m.log.Infof("gh-ost恢复成功: table=%s, work_id=%s", tableName, workID)
	return nil
}

// GhOstParams gh-ost执行参数
type GhOstParams struct {
	ChunkSize        int     // chunk-size: 100-100000
	MaxLagMillis     int     // max-lag-millis: >= 1000
	ThreadsRunning  int     // critical-load/max-load Threads_running: >= 1
	NiceRatio        float64 // nice-ratio: 0.0-100.0
}

// GetParams 获取当前gh-ost执行参数
func (m *GhOstManager) GetParams(ctx context.Context, logFile string) (*GhOstParams, error) {
	// 从日志文件中解析参数
	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("tail -10000 %s | grep 'chunk-size' | tail -1", logFile))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %v", err)
	}

	line := strings.TrimSpace(string(output))
	params := &GhOstParams{}

	// 解析chunk-size
	chunkRe := regexp.MustCompile(`chunk-size:\s*(\d+)`)
	matches := chunkRe.FindStringSubmatch(line)
	if len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil {
			params.ChunkSize = v
		}
	}

	// 解析max-lag-millis
	maxLagRe := regexp.MustCompile(`max-lag-millis:\s*(\d+)`)
	matches = maxLagRe.FindStringSubmatch(line)
	if len(matches) > 1 {
		if v, err := strconv.Atoi(strings.TrimSuffix(matches[1], "ms")); err == nil {
			params.MaxLagMillis = v
		}
	}

	// 解析critical-load Threads_running
	threadsRe := regexp.MustCompile(`critical-load:\s*Threads_running=(\d+)`)
	matches = threadsRe.FindStringSubmatch(line)
	if len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil {
			params.ThreadsRunning = v
		}
	}

	// 解析nice-ratio
	niceRe := regexp.MustCompile(`nice-ratio:\s*([\d.]+)`)
	matches = niceRe.FindStringSubmatch(line)
	if len(matches) > 1 {
		if v, err := strconv.ParseFloat(matches[1], 64); err == nil {
			params.NiceRatio = v
			if params.NiceRatio == 0.0 {
				params.NiceRatio = 0
			} else {
				params.NiceRatio = float64(int(v*10)) / 10 // 保留一位小数
			}
		}
	}

	return params, nil
}

// ModifyParams 修改gh-ost执行参数
func (m *GhOstManager) ModifyParams(ctx context.Context, tableName, workID string, params *GhOstParams) error {
	socketPath := m.GetSocketPath(tableName, workID)
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return fmt.Errorf("socket文件不存在: %s", socketPath)
	}

	// 验证参数范围
	if params.ChunkSize < 100 || params.ChunkSize >= 100000 {
		return fmt.Errorf("chunk-size取值范围是100-100000")
	}
	if params.MaxLagMillis < 1000 {
		return fmt.Errorf("max-lag-millis取值范围是大于等于1000")
	}
	if params.ThreadsRunning < 1 {
		return fmt.Errorf("Threads_running取值范围是大于等于1")
	}
	if params.NiceRatio < 0.0 || params.NiceRatio >= 100.0 {
		return fmt.Errorf("nice-ratio取值范围是0.0到100.0")
	}

	// 发送参数修改命令
	commands := []string{
		fmt.Sprintf("echo chunk-size=%d | socat - %s", params.ChunkSize, socketPath),
		fmt.Sprintf("echo max-lag-millis=%d | socat - %s", params.MaxLagMillis, socketPath),
		fmt.Sprintf("echo nice-ratio=%.1f | socat - %s", params.NiceRatio, socketPath),
		fmt.Sprintf("echo critical-load=Threads_running=%d | socat - %s", params.ThreadsRunning, socketPath),
		fmt.Sprintf("echo max-load=Threads_running=%d | socat - %s", params.ThreadsRunning, socketPath),
	}

	for _, cmdStr := range commands {
		cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
		output, err := cmd.CombinedOutput()
		if err != nil {
			m.log.Warnf("修改参数失败: cmd=%s, error=%v, output=%s", cmdStr, err, string(output))
			// 继续执行其他命令
		}
	}

	m.log.Infof("gh-ost参数修改成功: table=%s, work_id=%s, params=%+v", tableName, workID, params)
	return nil
}

// GetLog 获取gh-ost执行日志
func (m *GhOstManager) GetLog(ctx context.Context, workID string, lines int, offset int64) ([]string, int64, string, error) {
	// 1. 查找日志文件
	logFiles, err := filepath.Glob(fmt.Sprintf("%s/*_%s.log", m.logDir, workID))
	if err != nil || len(logFiles) == 0 {
		return nil, 0, "", fmt.Errorf("未找到日志文件: work_id=%s", workID)
	}

	// 获取最新的日志文件
	var latestLog string
	var latestTime time.Time
	for _, logFile := range logFiles {
		info, err := os.Stat(logFile)
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestLog = logFile
		}
	}

	if latestLog == "" {
		return nil, 0, "", fmt.Errorf("未找到有效的日志文件")
	}

	// 2. 读取文件内容
	content, err := os.ReadFile(latestLog)
	if err != nil {
		return nil, 0, "", fmt.Errorf("读取日志文件失败: %v", err)
	}

	allLines := strings.Split(string(content), "\n")
	totalLines := int64(len(allLines))

	// 3. 按需返回行
	if offset > 0 && offset < totalLines {
		// 增量获取: 从第 offset 行开始返回
		allLines = allLines[offset:]
	} else if lines > 0 && int64(lines) < totalLines {
		// 返回最后 N 行
		allLines = allLines[totalLines-int64(lines):]
	}

	return allLines, totalLines, latestLog, nil
}

// Stop 停止gh-ost执行
func (m *GhOstManager) Stop(ctx context.Context, tableName, workID string) error {
	panicPath := m.GetPanicPath(tableName, workID)

	// 创建panic文件来停止gh-ost
	file, err := os.Create(panicPath)
	if err != nil {
		return fmt.Errorf("创建panic文件失败: %v", err)
	}
	file.Close()

	m.log.Infof("gh-ost停止成功: table=%s, work_id=%s", tableName, workID)
	return nil
}
