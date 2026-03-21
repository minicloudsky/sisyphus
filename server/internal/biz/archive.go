package biz

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
)

// ArchiveScheduler 归档任务调度器接口（用于解耦，避免循环依赖）
type ArchiveScheduler interface {
	ScheduleArchiveTask(ctx context.Context, configID uint64)
}

// ArchiveUsecase 数据归档用例
type ArchiveUsecase struct {
	repo            ArchiveRepo
	instanceUsecase *DatabaseInstanceUsecase
	cron            *cron.Cron
	ptArchiverPath  string // pt-archiver二进制文件路径
	log             *log.Helper
	scheduler       ArchiveScheduler // 归档任务调度器接口
	cronLock        CronLeaderLock   // 多副本下避免每个 Pod 重复触发同一归档调度
}

// NewArchiveUsecase 创建数据归档用例
func NewArchiveUsecase(repo ArchiveRepo, instanceUsecase *DatabaseInstanceUsecase, ptArchiverPath string, logger log.Logger, cronLock CronLeaderLock) *ArchiveUsecase {
	if ptArchiverPath == "" {
		ptArchiverPath = "/usr/local/bin/pt-archiver"
	}
	return &ArchiveUsecase{
		repo:            repo,
		instanceUsecase: instanceUsecase,
		cron:            cron.New(cron.WithSeconds()),
		ptArchiverPath:  ptArchiverPath,
		log:             log.NewHelper(logger),
		cronLock:        cronLock,
	}
}

// SetScheduler 设置归档任务调度器
func (uc *ArchiveUsecase) SetScheduler(scheduler ArchiveScheduler) {
	uc.scheduler = scheduler
}

// ArchiveConfig 归档配置
type ArchiveConfig struct {
	ID              uint64
	InstanceID      uint64
	SourceDatabase  string
	SourceTable     string
	DestInstanceID  *uint64
	DestDatabase    string
	DestTable       string
	ArchiveField    string // 归档字段
	RetentionDays   int    // 保留天数
	ArchiveCondition string // 归档条件
	IsEnabled       bool
	CronExpression  string
	Owner           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ArchiveRecord 归档记录
type ArchiveRecord struct {
	ID           uint64
	ConfigID     uint64
	WorkID       string
	Status       int // 0-待执行, 1-执行中, 2-成功, 3-失败
	ArchiveCount int
	ExecuteTime  *time.Time
	FinishedTime *time.Time
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ArchiveRepo 归档仓储接口
type ArchiveRepo interface {
	CreateConfig(ctx context.Context, config *ArchiveConfig) (*ArchiveConfig, error)
	UpdateConfig(ctx context.Context, config *ArchiveConfig) error
	DeleteConfig(ctx context.Context, id uint64) error
	GetConfigByID(ctx context.Context, id uint64) (*ArchiveConfig, error)
	ListConfigs(ctx context.Context, page, pageSize int) ([]*ArchiveConfig, int64, error)
	GetEnabledConfigs(ctx context.Context) ([]*ArchiveConfig, error)
	
	CreateRecord(ctx context.Context, record *ArchiveRecord) (*ArchiveRecord, error)
	UpdateRecord(ctx context.Context, record *ArchiveRecord) error
	GetRecordByID(ctx context.Context, id uint64) (*ArchiveRecord, error)
	ListRecords(ctx context.Context, configID uint64, page, pageSize int) ([]*ArchiveRecord, int64, error)
}

// StartCron 启动定时任务（现在通过Asynq执行，cron只用于调度）
func (uc *ArchiveUsecase) StartCron(ctx context.Context) error {
	// 注意：归档任务现在通过Asynq执行，cron只用于调度
	// 获取所有启用的归档配置
	configs, err := uc.repo.GetEnabledConfigs(ctx)
	if err != nil {
		return err
	}

	// 为每个配置添加定时任务（定时任务会创建Asynq任务）
	for _, config := range configs {
		configID := config.ID
		_, err := uc.cron.AddFunc(config.CronExpression, func() {
			runCronAsLeader(uc.log, uc.cronLock, context.Background(), fmt.Sprintf("sisyphus:cron:archive:config:%d", configID), 30*time.Minute, func() {
				// 定时任务触发时，通过调度器入队Asynq任务
				if uc.scheduler != nil {
					uc.scheduler.ScheduleArchiveTask(context.Background(), configID)
				} else {
					// 如果没有设置调度器，回退到直接执行（兼容模式）
					uc.log.Warnf("归档调度器未设置，使用直接执行模式: config_id=%d", configID)
					uc.executeArchive(context.Background(), configID)
				}
			})
		})
		if err != nil {
			uc.log.Errorf("添加归档定时任务失败: config_id=%d, error=%v", configID, err)
			continue
		}
	}

	uc.cron.Start()
	uc.log.Info("归档定时任务已启动（注意：实际执行已迁移到Asynq）")
	return nil
}

// StopCron 停止定时任务
func (uc *ArchiveUsecase) StopCron() {
	uc.cron.Stop()
	uc.log.Info("归档定时任务已停止")
}

// executeArchive 执行归档（保留用于cron定时任务，实际执行已迁移到Asynq）
// TriggerArchiveNow 手动触发一次归档执行
func (uc *ArchiveUsecase) TriggerArchiveNow(ctx context.Context, configID uint64) {
	go uc.executeArchive(context.Background(), configID)
}

func (uc *ArchiveUsecase) executeArchive(ctx context.Context, configID uint64) {
	config, err := uc.repo.GetConfigByID(ctx, configID)
	if err != nil {
		uc.log.Errorf("获取归档配置失败: config_id=%d, error=%v", configID, err)
		return
	}

	if !config.IsEnabled {
		return
	}

	// 创建归档记录
	workID := generateWorkID()
	record := &ArchiveRecord{
		ConfigID: configID,
		WorkID:   workID,
		Status:   1, // 执行中
	}
	now := time.Now()
	record.ExecuteTime = &now
	record, err = uc.repo.CreateRecord(ctx, record)
	if err != nil {
		uc.log.Errorf("创建归档记录失败: error=%v", err)
		return
	}

	// 执行pt-archiver
	err = uc.runPtArchiver(ctx, config, record)
	if err != nil {
		record.Status = 3 // 失败
		record.ErrorMessage = err.Error()
		finished := time.Now()
		record.FinishedTime = &finished
		uc.repo.UpdateRecord(ctx, record)
		uc.log.Errorf("归档执行失败: config_id=%d, error=%v", configID, err)
		return
	}

	// 更新记录
	record.Status = 2 // 成功
	finished := time.Now()
	record.FinishedTime = &finished
	uc.repo.UpdateRecord(ctx, record)
	uc.log.Infof("归档执行成功: config_id=%d, work_id=%s", configID, workID)
}

// EnqueueArchiveTask 入队归档任务（供外部调用，使用Asynq）
func (uc *ArchiveUsecase) EnqueueArchiveTask(ctx context.Context, configID uint64, workID string, command string, executeTime *time.Time) error {
	// 这个方法供外部调用，实际入队由Service层完成
	// 这里只是提供接口，实际实现在Service层
	return nil
}

// runPtArchiver 运行pt-archiver工具
func (uc *ArchiveUsecase) runPtArchiver(ctx context.Context, config *ArchiveConfig, record *ArchiveRecord) error {
	// 获取源实例和目标实例信息
	sourceInstanceInfo, err := uc.getDatabaseInstance(ctx, config.InstanceID)
	if err != nil {
		return err
	}

	var destInstance *InstanceInfo
	if config.DestInstanceID != nil {
		destInstance, err = uc.getDatabaseInstance(ctx, *config.DestInstanceID)
		if err != nil {
			return err
		}
	}

	// 构建归档条件
	archiveCondition := config.ArchiveCondition
	if archiveCondition == "" {
		// 根据保留天数自动生成条件
		cutoffDate := time.Now().AddDate(0, 0, -config.RetentionDays)
		archiveCondition = fmt.Sprintf("%s < '%s'", config.ArchiveField, cutoffDate.Format("2006-01-02"))
	}

	// 构建pt-archiver命令
	cmd := fmt.Sprintf(`%s --no-version-check \
		--statistics \
		--txn-size=1000 \
		--limit=1000 \
		--source h=%s,P=%d,u=%s,p='%s',D=%s,t=%s \
		--dest h=%s,P=%d,u=%s,p='%s',D=%s,t=%s \
		--where "%s" \
		--sentinel="/tmp/%s_pt-archiver-sentinel"`,
		uc.ptArchiverPath,
		sourceInstanceInfo.Host, sourceInstanceInfo.Port, sourceInstanceInfo.Username, sourceInstanceInfo.Password,
		config.SourceDatabase, config.SourceTable,
		destInstance.Host, destInstance.Port, destInstance.Username, destInstance.Password,
		config.DestDatabase, config.DestTable,
		archiveCondition, record.WorkID)

	uc.log.Infof("执行pt-archiver: %s", cmd)

	// 执行命令
	execCmd := exec.CommandContext(ctx, "sh", "-c", cmd)
	output, err := execCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pt-archiver执行失败: %v, 输出: %s", err, string(output))
	}

	// 解析输出，获取归档数量
	// 这里简化处理，实际应该解析pt-archiver的输出
	uc.log.Infof("pt-archiver输出: %s", string(output))
	
	return nil
}

// getDatabaseInstance 获取数据库实例信息
func (uc *ArchiveUsecase) getDatabaseInstance(ctx context.Context, instanceID uint64) (*InstanceInfo, error) {
	instance, err := uc.instanceUsecase.GetInstanceByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	return &InstanceInfo{
		Host:     instance.Host,
		Port:     instance.Port,
		Username: instance.Username,
		Password: instance.Password,
	}, nil
}

// CreateArchiveConfig 创建归档配置
func (uc *ArchiveUsecase) CreateArchiveConfig(ctx context.Context, config *ArchiveConfig) (*ArchiveConfig, error) {
	return uc.repo.CreateConfig(ctx, config)
}

// UpdateArchiveConfig 更新归档配置
func (uc *ArchiveUsecase) UpdateArchiveConfig(ctx context.Context, config *ArchiveConfig) error {
	return uc.repo.UpdateConfig(ctx, config)
}

// DeleteArchiveConfig 删除归档配置
func (uc *ArchiveUsecase) DeleteArchiveConfig(ctx context.Context, id uint64) error {
	return uc.repo.DeleteConfig(ctx, id)
}

// GetConfigByID 获取归档配置
func (uc *ArchiveUsecase) GetConfigByID(ctx context.Context, id uint64) (*ArchiveConfig, error) {
	return uc.repo.GetConfigByID(ctx, id)
}

// ListConfigs 列出归档配置
func (uc *ArchiveUsecase) ListConfigs(ctx context.Context, page, pageSize int) ([]*ArchiveConfig, int64, error) {
	return uc.repo.ListConfigs(ctx, page, pageSize)
}

// ListRecords 列出归档记录
func (uc *ArchiveUsecase) ListRecords(ctx context.Context, configID uint64, page, pageSize int) ([]*ArchiveRecord, int64, error) {
	return uc.repo.ListRecords(ctx, configID, page, pageSize)
}

// CreateRecord 创建归档记录
func (uc *ArchiveUsecase) CreateRecord(ctx context.Context, record *ArchiveRecord) (*ArchiveRecord, error) {
	return uc.repo.CreateRecord(ctx, record)
}

// UpdateRecord 更新归档记录
func (uc *ArchiveUsecase) UpdateRecord(ctx context.Context, record *ArchiveRecord) error {
	return uc.repo.UpdateRecord(ctx, record)
}

// GetRecordByID 获取归档记录
func (uc *ArchiveUsecase) GetRecordByID(ctx context.Context, id uint64) (*ArchiveRecord, error) {
	return uc.repo.GetRecordByID(ctx, id)
}

// BuildArchiveCommand 构建归档命令（供外部调用）
func (uc *ArchiveUsecase) BuildArchiveCommand(ctx context.Context, config *ArchiveConfig, workID string) (string, error) {
	// 获取源实例和目标实例信息
	sourceInstanceInfo, err := uc.getDatabaseInstance(ctx, config.InstanceID)
	if err != nil {
		return "", err
	}

	var destInstance *InstanceInfo
	if config.DestInstanceID != nil {
		destInstance, err = uc.getDatabaseInstance(ctx, *config.DestInstanceID)
		if err != nil {
			return "", err
		}
	} else {
		// 如果没有目标实例，使用源实例（删除模式）
		destInstance = sourceInstanceInfo
	}

	// 构建归档条件
	archiveCondition := config.ArchiveCondition
	if archiveCondition == "" {
		// 根据保留天数自动生成条件
		cutoffDate := time.Now().AddDate(0, 0, -config.RetentionDays)
		archiveCondition = fmt.Sprintf("%s < '%s'", config.ArchiveField, cutoffDate.Format("2006-01-02"))
	}

	// 构建pt-archiver命令
	cmd := fmt.Sprintf(`%s --no-version-check \
		--statistics \
		--txn-size=1000 \
		--limit=1000 \
		--source h=%s,P=%d,u=%s,p='%s',D=%s,t=%s \
		--dest h=%s,P=%d,u=%s,p='%s',D=%s,t=%s \
		--where "%s" \
		--sentinel="/tmp/%s_pt-archiver-sentinel"`,
		uc.ptArchiverPath,
		sourceInstanceInfo.Host, sourceInstanceInfo.Port, sourceInstanceInfo.Username, sourceInstanceInfo.Password,
		config.SourceDatabase, config.SourceTable,
		destInstance.Host, destInstance.Port, destInstance.Username, destInstance.Password,
		config.DestDatabase, config.DestTable,
		archiveCondition, workID)

	return cmd, nil
}
