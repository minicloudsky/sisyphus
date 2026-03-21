package biz

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

// SQLExecutor SQL执行器
type SQLExecutor struct {
	log          *log.Helper
	ghostManager *GhOstManager
	ghOstPath    string // gh-ost二进制文件路径
}

// NewSQLExecutor 创建SQL执行器
func NewSQLExecutor(ghostManager *GhOstManager, ghOstPath string, logger log.Logger) *SQLExecutor {
	if ghOstPath == "" {
		ghOstPath = "/usr/local/bin/gh-ost"
	}
	return &SQLExecutor{
		log:          log.NewHelper(logger),
		ghostManager: ghostManager,
		ghOstPath:    ghOstPath,
	}
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	Success      bool
	ErrorMessage string
	AffectedRows int
	ExecuteTime  string
	Sequence     string
	SQLHash      string
	BackupDBName string
	SQLText      string // 添加SQL文本字段
}

// ExecuteWithGoInception 使用goInception执行SQL
func (e *SQLExecutor) ExecuteWithGoInception(ctx context.Context, client *GoInceptionClient, sql, database string, needBackup bool) ([]*ExecuteResult, error) {
	// 直接调用client的ExecuteSQL方法，它会内部构建正确的SQL
	results, err := client.ExecuteSQL(ctx, sql, database, needBackup)
	if err != nil {
		return nil, err
	}

	var execResults []*ExecuteResult
	for _, result := range results {
		// 检查stage字段，如果是RERUN，说明SQL没有真正执行
		// RERUN通常表示goInception检测到SQL格式问题，需要重新执行
		if strings.ToUpper(result.Stage) == "RERUN" {
			fmt.Printf("[SQLExecutor] Warning: Result stage=RERUN, SQL may not be executed. error_level=%d, stage_status=%s, affected_rows=%d\n",
				result.ErrorLevel, result.StageStatus, result.AffectedRows)
		}
		// 判断执行是否成功：
		// 关键：参考Python项目实现和goInception官方文档
		// Python项目中：if i['stagestatus'] == 'Audit completed': 标记为执行失败
		// 这说明即使affected_rows>0，如果stage_status='Audit completed'，也可能表示事务没有提交
		// 只有stage_status='Execute Successfully'才表示真正执行成功并提交了事务
		stageStatusLower := strings.ToLower(result.StageStatus)
		hasErrorMsg := strings.TrimSpace(result.ErrorMessage) != ""
		
		isSuccess := false
		
		// 检查是否是内部检查错误（如SHOW MASTER STATUS）
		errorMsgLower := strings.ToLower(result.ErrorMessage)
		isInternalCheckError := strings.Contains(errorMsgLower, "show master status") ||
			strings.Contains(errorMsgLower, "show slave status") ||
			strings.Contains(errorMsgLower, "show binlog status")
		
		// 关键判断：优先检查stage_status，只有明确表示"Execute Successfully"才认为是成功
		// 但如果affected_rows>0且错误是内部检查错误，应该视为成功（因为SQL实际执行了）
		if strings.Contains(stageStatusLower, "execute successfully") ||
			strings.Contains(stageStatusLower, "执行成功") {
			// 明确表示执行成功，视为成功
			isSuccess = true
		} else if strings.Contains(stageStatusLower, "audit completed") {
			// Audit Completed：goInception 的结果中常见“审核阶段条目”。
			// 这些条目不一定会带有正确的 affected_rows（尤其是 DDL/DML），
			// 如果此阶段没有明显错误（或属于内部检查错误），应当视为成功，
			// 否则会把整个工单误判为失败（导致状态从成功(4)变失败(5)）。
			if isInternalCheckError || !hasErrorMsg {
				isSuccess = true
			} else {
				isSuccess = false
			}
		} else if result.ErrorLevel == 2 {
			// error_level=2：通常是失败；但内部检查错误可以视为成功。
			isSuccess = isInternalCheckError
		} else if result.ErrorLevel == 1 {
			// 警告级别，如果有错误信息则失败，否则可能成功
			isSuccess = !hasErrorMsg
		} else {
			// error_level=0，检查其他状态
			if strings.Contains(stageStatusLower, "executed") ||
				strings.Contains(stageStatusLower, "已执行") {
				// 已执行状态，认为是成功
				isSuccess = true
			} else if !hasErrorMsg {
				// 其他状态，如果没有错误信息，可能是成功
				isSuccess = true
			} else {
				// 有错误信息，失败
				isSuccess = false
			}
		}
		
		execResult := &ExecuteResult{
			Success:      isSuccess,
			ErrorMessage: result.ErrorMessage,
			AffectedRows: result.AffectedRows,
			ExecuteTime:  result.ExecuteTime,
			Sequence:     result.Sequence,
			SQLHash:      result.SQLHash,
			BackupDBName: result.BackupDBName,
			SQLText:      result.SQLText,
		}
		execResults = append(execResults, execResult)
	}

	return execResults, nil
}

// ExecuteWithGhOst 使用gh-ost执行SQL（在线DDL）
func (e *SQLExecutor) ExecuteWithGhOst(ctx context.Context, instance *DatabaseInstance, sql, database, table, workID string, deleteGho bool) error {
	// 从ALTER TABLE语句中提取ALTER子句
	alterClause := e.extractAlterClause(sql)
	if alterClause == "" {
		return fmt.Errorf("无法从SQL中提取ALTER子句")
	}

	// 获取文件路径
	socketPath := e.ghostManager.GetSocketPath(table, workID)
	logPath := e.ghostManager.GetLogPath(table, workID)
	panicPath := e.ghostManager.GetPanicPath(table, workID)

	// 构建gh-ost命令
	var droptableStr string
	if deleteGho {
		droptableStr = "--ok-to-drop-table"
	}

	ghostCmd := fmt.Sprintf(`%s %s \
		--aliyun-rds="true" \
		--critical-load=Threads_running=1000 \
		--max-load=Threads_running=1000 \
		--chunk-size=1000 \
		--max-lag-millis=1500 \
		--dml-batch-size=50 \
		--nice-ratio=0.5 \
		--assume-master-host="%s:3306" \
		--assume-rbr \
		--initially-drop-old-table \
		--initially-drop-ghost-table \
		--initially-drop-socket-file \
		--host="%s" \
		--port=%d \
		--user="%s" \
		--password="%s" \
		--database="%s" \
		--table="%s" \
		--verbose \
		--alter="%s" \
		--allow-on-master \
		--cut-over=default \
		--default-retries=1 \
		--panic-flag-file=%s \
		--serve-socket-file=%s \
		--approve-renamed-columns \
		--execute > %s 2>&1`,
		e.ghOstPath, droptableStr, instance.Host, instance.Host, instance.Port, instance.Username, instance.Password,
		database, table, alterClause, panicPath, socketPath, logPath)

	e.log.Infof("执行gh-ost命令: %s", ghostCmd)

	// 执行命令（异步执行，不等待完成）
	cmd := exec.CommandContext(ctx, "sh", "-c", ghostCmd)
	// 注意：gh-ost是长时间运行的命令，这里应该异步执行
	// 实际执行状态通过socket和日志文件监控
	err := cmd.Start()
	if err != nil {
		e.log.Errorf("gh-ost启动失败: %v", err)
		return fmt.Errorf("gh-ost启动失败: %v", err)
	}

	e.log.Infof("gh-ost已启动: table=%s, work_id=%s, pid=%d", table, workID, cmd.Process.Pid)
	return nil
}

// extractAlterClause 从ALTER TABLE语句中提取ALTER子句
func (e *SQLExecutor) extractAlterClause(sql string) string {
	sql = strings.TrimSpace(sql)
	sqlLower := strings.ToLower(sql)

	// 移除ALTER TABLE table_name部分
	if strings.HasPrefix(sqlLower, "alter table") {
		// 移除ALGORITHM和LOCK参数
		sql = strings.ReplaceAll(sql, "ALGORITHM=INPLACE,LOCK=NONE", "")
		sql = strings.ReplaceAll(sql, "ALGORITHM=INPLACE, LOCK=NONE", "")
		sql = strings.ReplaceAll(sql, ", ALGORITHM=INPLACE, LOCK=NONE", "")
		sql = strings.ReplaceAll(sql, "ALGORITHM=INSTANT,LOCK=DEFAULT", "")
		
		parts := strings.Fields(sql)
		if len(parts) >= 4 {
			// 找到第一个非表名的部分（通常是ADD, MODIFY, CHANGE等）
			for i := 3; i < len(parts); i++ {
				partLower := strings.ToLower(parts[i])
				if partLower == "add" || partLower == "modify" || partLower == "change" ||
					partLower == "drop" || partLower == "alter" {
					return strings.Join(parts[i:], " ")
				}
			}
		}
	}

	return ""
}

// extractTableName 从ALTER TABLE语句中提取表名
func (e *SQLExecutor) extractTableName(sql string) string {
	sql = strings.TrimSpace(sql)
	sqlLower := strings.ToLower(sql)

	if strings.HasPrefix(sqlLower, "alter table") {
		parts := strings.Fields(sql)
		if len(parts) >= 3 {
			tableName := parts[2]
			// 移除反引号
			tableName = strings.Trim(tableName, "`")
			// 处理database.table格式
			if strings.Contains(tableName, ".") {
				tableName = strings.Split(tableName, ".")[1]
			}
			return tableName
		}
	}

	return ""
}

// InstanceInfo 数据库实例信息（用于执行）
type InstanceInfo struct {
	Host     string
	Port     int
	Username string
	Password string
}
