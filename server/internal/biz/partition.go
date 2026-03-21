package biz

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
)

// PartitionUsecase 分区管理用例
type PartitionUsecase struct {
	repo              PartitionRepo
	instanceUsecase   *DatabaseInstanceUsecase
	cron              *cron.Cron
	log               *log.Helper
	partitionAddDays  int // 提前添加分区的天数，默认180天(约6个月)
	cronLock          CronLeaderLock
}

// NewPartitionUsecase 创建分区管理用例
func NewPartitionUsecase(repo PartitionRepo, instanceUsecase *DatabaseInstanceUsecase, logger log.Logger, cronLock CronLeaderLock) *PartitionUsecase {
	return &PartitionUsecase{
		repo:             repo,
		instanceUsecase:  instanceUsecase,
		cron:             cron.New(cron.WithSeconds()),
		log:              log.NewHelper(logger),
		partitionAddDays: 180, // 默认提前6个月添加分区
		cronLock:         cronLock,
	}
}

// PartitionTableInfo 分区表信息
type PartitionTableInfo struct {
	ID                uint64
	InstanceID        uint64
	InstanceName      string
	ConnectionString  string
	DatabaseName      string
	TableName         string
	PartType          string // RANGE, HASH等
	PartKey           string // 分区键表达式
	PartCycle         string // 分区周期: 天, 周, 月, 季, 半年, 年
	PartNum           int    // 分区数量
	PartMaxValue      string // 最大分区值
	PartMaxValueDay   string // 最大分区距离当前日期的天数状态: 已到期, 未到期
	HaveMaxValue      string // 是否有MAXVALUE分区: 是, 否
	TableSize         float64 // 表大小(GB)
	TableRows         int64   // 总行数
	MaxPartRows       int64   // 最大分区行数
	IsCrontab         string  // 是否定时任务维护: 是, 否
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// PartitionOperationRecord 分区操作记录
type PartitionOperationRecord struct {
	ID              uint64
	WorkID          string
	InstanceID      uint64
	InstanceName    string
	ConnectionString string
	TableSchema     string
	TableName       string
	OperateType     int    // 1-初始化, 2-添加分区, 3-数据归档, 4-数据校验, 5-分区删除, 6-数据删除
	ExeStatus       int    // 1-创建, 2-取消, 3-执行中, 4-成功, 5-失败
	CheckStatus     int    // 0-未确认, 1-确认
	ExePct          string // 执行进度
	ExeResult       string // 执行结果
	ExeCmd          string // 执行命令
	WhereCondition  string // 归档条件
	AddUser         string
	UpdateUser      string
	ExeTime         *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PartitionRepo 分区仓储接口
type PartitionRepo interface {
	// 分区表信息
	UpsertPartitionTableInfo(ctx context.Context, info *PartitionTableInfo) error
	GetPartitionTableInfo(ctx context.Context, instanceID uint64, databaseName, tableName string) (*PartitionTableInfo, error)
	GetPartitionTableInfoByID(ctx context.Context, id uint64) (*PartitionTableInfo, error)
	ListExpiredPartitionTables(ctx context.Context) ([]*PartitionTableInfo, error)
	ListPartitionTables(ctx context.Context, instanceID uint64) ([]*PartitionTableInfo, error)
	ListPartitionTablesPaged(ctx context.Context, instanceID uint64, page, pageSize int) ([]*PartitionTableInfo, int64, error)
	UpdatePartitionTableInfo(ctx context.Context, info *PartitionTableInfo) error
	DeletePartitionTableInfo(ctx context.Context, id uint64) error
	CreatePartitionTableInfo(ctx context.Context, info *PartitionTableInfo) error

	// 分区操作记录
	CreateOperationRecord(ctx context.Context, record *PartitionOperationRecord) error
	UpdateOperationRecord(ctx context.Context, record *PartitionOperationRecord) error
	GetOperationRecordByWorkID(ctx context.Context, workID string) (*PartitionOperationRecord, error)
	ListOperationRecords(ctx context.Context, instanceID uint64, operateType int, page, pageSize int) ([]*PartitionOperationRecord, int64, error)
	ListOperationRecordsByTableInfoID(ctx context.Context, tableInfoID uint64, page, pageSize int) ([]*PartitionOperationRecord, int64, error)

	// 归档配置相关(复用ArchiveRepo的部分功能)
	GetPartitionArchiveConfigs(ctx context.Context) ([]*PartitionArchiveConfig, error)
}

// PartitionArchiveConfig 分区归档配置(扩展ArchiveConfig)
type PartitionArchiveConfig struct {
	ID              uint64
	InstanceID      uint64
	InstanceName    string
	ConnectionString string
	TableSchema     string
	TableName       string
	IsPartition     int    // 是否分区表: 1-是, 0-否
	DestHost        string // 目标主机
	DestSchema      string // 目标库
	IsArchive       int    // 是否归档: 0-不归档, 1-归档
	IsChecksum      int    // 是否校验: 0-不校验, 1-校验
	IsDelete        int    // 是否删除: 0-不删除, 1-删除
	ArchField       string // 归档字段
	ExpireDays      int    // 保留天数
	IsPause         int    // 是否暂停: 0-正常, 1-暂停
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// StartCron 启动定时任务
func (uc *PartitionUsecase) StartCron(ctx context.Context) error {
	// 每天凌晨2点收集分区表信息
	_, err := uc.cron.AddFunc("0 0 2 * * *", func() {
		runCronAsLeader(uc.log, uc.cronLock, context.Background(), "sisyphus:cron:partition:collect", 4*time.Hour, func() {
			_ = uc.CollectPartitionTableInfo(context.Background())
		})
	})
	if err != nil {
		return err
	}

	// 每天凌晨3点自动添加分区
	_, err = uc.cron.AddFunc("0 0 3 * * *", func() {
		runCronAsLeader(uc.log, uc.cronLock, context.Background(), "sisyphus:cron:partition:auto_add", 4*time.Hour, func() {
			_ = uc.AutoAddPartitions(context.Background())
		})
	})
	if err != nil {
		return err
	}

	// 每天凌晨4点自动删除分区
	_, err = uc.cron.AddFunc("0 0 4 * * *", func() {
		runCronAsLeader(uc.log, uc.cronLock, context.Background(), "sisyphus:cron:partition:auto_delete", 4*time.Hour, func() {
			_ = uc.AutoDeletePartitions(context.Background())
		})
	})
	if err != nil {
		return err
	}

	uc.cron.Start()
	uc.log.Info("分区管理定时任务已启动")
	return nil
}

// StopCron 停止定时任务
func (uc *PartitionUsecase) StopCron() {
	uc.cron.Stop()
	uc.log.Info("分区管理定时任务已停止")
}

// CollectPartitionTableInfo 收集分区表信息
func (uc *PartitionUsecase) CollectPartitionTableInfo(ctx context.Context) error {
	uc.log.Info("开始收集分区表信息")
	
	// 获取所有数据库实例
	var dbType *string
	var env *string
	instances, _, err := uc.instanceUsecase.ListInstances(ctx, 1, 1000, dbType, env, false, nil)
	if err != nil {
		return fmt.Errorf("获取实例列表失败: %v", err)
	}

	for _, instance := range instances {
		if instance.Status != 1 { // 跳过非正常状态的实例
			continue
		}

		// 连接数据库获取分区表信息
		partitionTables, err := uc.getPartitionTablesFromDB(ctx, instance)
		if err != nil {
			uc.log.Errorf("获取实例 %s 的分区表信息失败: %v", instance.Name, err)
			continue
		}

		// 保存到数据库
		for _, pt := range partitionTables {
			if err := uc.repo.UpsertPartitionTableInfo(ctx, pt); err != nil {
				uc.log.Errorf("保存分区表信息失败: %v", err)
			}
		}
	}

	uc.log.Info("分区表信息收集完成")
	return nil
}

// getPartitionTablesFromDB 从数据库获取分区表信息
func (uc *PartitionUsecase) getPartitionTablesFromDB(ctx context.Context, instance *DatabaseInstance) ([]*PartitionTableInfo, error) {
	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/information_schema?charset=utf8mb4&parseTime=True&loc=Local",
		instance.Username, instance.Password, instance.Host, instance.Port)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 查询分区表信息
	query := `
		SELECT 
			table_schema,
			table_name,
			partition_method,
			partition_expression,
			COUNT(*) as part_count,
			CASE 
				WHEN partition_expression LIKE '%to_days%' THEN from_days(MAX(partition_description))
				WHEN partition_expression LIKE '%unix_timestamp%' THEN LEFT(FROM_UNIXTIME(MAX(partition_description)), 10)
				ELSE MAX(partition_description)
			END as maxpart_value,
			MAX(CASE WHEN partition_description = 'MAXVALUE' THEN '是' ELSE '否' END) as have_maxvalue,
			ROUND(SUM(data_length + index_length) / 1024 / 1024 / 1024, 2) as table_size,
			SUM(table_rows) as table_rows
		FROM information_schema.partitions
		WHERE table_schema NOT IN ('information_schema', 'test', 'mysql', 'performance_schema', '__recycle_bin__')
			AND partition_name IS NOT NULL
			AND partition_description <> 'MAXVALUE'
			AND table_name NOT LIKE '%_bak'
			AND table_name NOT LIKE '%_tmp%'
			AND table_name NOT LIKE '_%_del'
			AND table_name NOT LIKE '_%_ghm'
			AND table_name NOT LIKE '_%_gho'
		GROUP BY table_schema, table_name, partition_method, partition_expression
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var partitionTables []*PartitionTableInfo
	for rows.Next() {
		var pt PartitionTableInfo
		var maxpartValue sql.NullString
		var haveMaxValue sql.NullString
		
		err := rows.Scan(
			&pt.DatabaseName,
			&pt.TableName,
			&pt.PartType,
			&pt.PartKey,
			&pt.PartNum,
			&maxpartValue,
			&haveMaxValue,
			&pt.TableSize,
			&pt.TableRows,
		)
		if err != nil {
			continue
		}

		pt.InstanceID = instance.ID
		pt.InstanceName = instance.Name
		pt.ConnectionString = instance.Host
		if maxpartValue.Valid {
			pt.PartMaxValue = maxpartValue.String
		}
		if haveMaxValue.Valid {
			pt.HaveMaxValue = haveMaxValue.String
		}

		// 计算分区周期
		pt.PartCycle = uc.calculatePartCycle(ctx, instance, pt.DatabaseName, pt.TableName, pt.PartKey)
		
		// 计算是否到期
		pt.PartMaxValueDay = uc.calculateExpireStatus(pt.PartMaxValue)

		// 获取最大分区行数
		pt.MaxPartRows = uc.getMaxPartitionRows(ctx, instance, pt.DatabaseName, pt.TableName)

		partitionTables = append(partitionTables, &pt)
	}

	return partitionTables, nil
}

// calculatePartCycle 计算分区周期
func (uc *PartitionUsecase) calculatePartCycle(ctx context.Context, instance *DatabaseInstance, databaseName, tableName, partKey string) string {
	// 查询相邻两个分区的差值
	query := `
		SELECT 
			CASE 
				WHEN partition_expression LIKE '%to_days%' THEN 
					MAX(partition_description) - MIN(partition_description)
				WHEN partition_expression LIKE '%unix_timestamp%' THEN 
					(MAX(partition_description) - MIN(partition_description)) / 86400
				ELSE 0
			END as diff
		FROM (
			SELECT partition_description
			FROM information_schema.partitions
			WHERE table_schema = ? AND table_name = ? 
				AND partition_description <> 'MAXVALUE'
			ORDER BY partition_description DESC
			LIMIT 2
		) t
	`

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		instance.Username, instance.Password, instance.Host, instance.Port, databaseName)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return ""
	}
	defer db.Close()

	var diff sql.NullInt64
	err = db.QueryRowContext(ctx, query, databaseName, tableName).Scan(&diff)
	if err != nil || !diff.Valid {
		return ""
	}

	diffDays := int(diff.Int64)
	switch {
	case diffDays == 1:
		return "天"
	case diffDays == 7:
		return "周"
	case diffDays >= 28 && diffDays < 32:
		return "月"
	case diffDays >= 89 && diffDays <= 92:
		return "季"
	case diffDays >= 180 && diffDays <= 190:
		return "半年"
	case diffDays >= 360 && diffDays <= 369:
		return "年"
	default:
		return ""
	}
}

// calculateExpireStatus 计算分区是否到期
func (uc *PartitionUsecase) calculateExpireStatus(maxpartValue string) string {
	if maxpartValue == "" {
		return "未到期"
	}

	maxDate, err := time.Parse("2006-01-02", maxpartValue)
	if err != nil {
		return "未到期"
	}

	days := int(time.Since(maxDate).Hours() / 24)
	if days > uc.partitionAddDays {
		return "未到期"
	}
	return "已到期"
}

// getMaxPartitionRows 获取最大分区行数
func (uc *PartitionUsecase) getMaxPartitionRows(ctx context.Context, instance *DatabaseInstance, databaseName, tableName string) int64 {
	query := `
		SELECT table_rows
		FROM information_schema.partitions
		WHERE table_schema = ? AND table_name = ?
			AND partition_description <> 'MAXVALUE'
		ORDER BY partition_description DESC
		LIMIT 1
	`

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		instance.Username, instance.Password, instance.Host, instance.Port, databaseName)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return 0
	}
	defer db.Close()

	var rows sql.NullInt64
	err = db.QueryRowContext(ctx, query, databaseName, tableName).Scan(&rows)
	if err != nil || !rows.Valid {
		return 0
	}

	return rows.Int64
}

// AutoAddPartitions 自动添加分区
func (uc *PartitionUsecase) AutoAddPartitions(ctx context.Context) error {
	uc.log.Info("开始自动添加分区")

	// 获取所有已到期的分区表
	expiredTables, err := uc.repo.ListExpiredPartitionTables(ctx)
	if err != nil {
		return fmt.Errorf("获取已到期分区表失败: %v", err)
	}

	for _, table := range expiredTables {
		if err := uc.addPartitionForTable(ctx, table); err != nil {
			uc.log.Errorf("为表 %s.%s 添加分区失败: %v", table.DatabaseName, table.TableName, err)
		}
	}

	uc.log.Info("自动添加分区完成")
	return nil
}

// addPartitionForTable 为表添加分区
func (uc *PartitionUsecase) addPartitionForTable(ctx context.Context, table *PartitionTableInfo) error {
	// 获取实例信息
	instance, err := uc.instanceUsecase.GetInstanceByID(ctx, table.InstanceID)
	if err != nil {
		return err
	}

	// 生成添加分区的DDL语句
	ddl, err := uc.generateAddPartitionDDL(ctx, instance, table)
	if err != nil {
		return err
	}

	if ddl == "" {
		return fmt.Errorf("无法生成DDL语句")
	}

	// 创建操作记录
	workID := uc.generateWorkID()
	record := &PartitionOperationRecord{
		WorkID:          workID,
		InstanceID:      table.InstanceID,
		InstanceName:    table.InstanceName,
		ConnectionString: table.ConnectionString,
		TableSchema:     table.DatabaseName,
		TableName:       table.TableName,
		OperateType:     2, // 添加分区
		ExeStatus:       3, // 执行中
		ExeCmd:          ddl,
		AddUser:         "cron_add",
	}

	now := time.Now()
	record.ExeTime = &now
	if err := uc.repo.CreateOperationRecord(ctx, record); err != nil {
		return err
	}

	// 执行DDL
	err = uc.executeDDL(ctx, instance, table.DatabaseName, ddl)
	if err != nil {
		record.ExeStatus = 5 // 失败
		record.ExeResult = err.Error()
		uc.repo.UpdateOperationRecord(ctx, record)
		return err
	}

	// 更新记录
	record.ExeStatus = 4 // 成功
	record.ExeResult = "成功"
	record.CheckStatus = 1
	uc.repo.UpdateOperationRecord(ctx, record)

	// 更新分区表信息
	uc.CollectPartitionTableInfo(ctx)

	return nil
}

// generateAddPartitionDDL 生成添加分区的DDL语句
func (uc *PartitionUsecase) generateAddPartitionDDL(ctx context.Context, instance *DatabaseInstance, table *PartitionTableInfo) (string, error) {
	if table.PartType != "RANGE" {
		return "", fmt.Errorf("非RANGE分区，暂不支持")
	}

	// 判断分区函数
	var partPrefix string
	if strings.Contains(strings.ToLower(table.PartKey), "to_days") {
		partPrefix = "to_days"
	} else if strings.Contains(strings.ToLower(table.PartKey), "unix_timestamp") {
		partPrefix = "unix_timestamp"
	} else {
		return "", fmt.Errorf("RANGE分区未按照指定日期函数分区")
	}

	if table.PartCycle == "" {
		return "", fmt.Errorf("无法确定分区周期")
	}

	// 解析最大分区日期
	maxpartDate, err := time.Parse("2006-01-02", table.PartMaxValue)
	if err != nil {
		return "", fmt.Errorf("解析最大分区日期失败: %v", err)
	}

	// 计算需要添加的分区
	now := time.Now()
	targetDate := now.AddDate(0, 0, uc.partitionAddDays)
	
	var partitions []string
	currentDate := maxpartDate
	count := 0
	maxCount := 30 // 最多一次添加30个分区

	for currentDate.Before(targetDate) && count < maxCount {
		nextDate, partName := uc.getNextPartitionDate(currentDate, table.PartCycle)
		if nextDate.IsZero() {
			break
		}

		// 计算分区值
		partValue := uc.calculatePartitionValue(nextDate, partPrefix)
		if partValue == 0 {
			break
		}

		partitions = append(partitions, fmt.Sprintf("PARTITION %s VALUES LESS THAN (%d)", partName, partValue))
		currentDate = nextDate
		count++
	}

	if len(partitions) == 0 {
		return "", fmt.Errorf("无需添加分区")
	}

	// 生成DDL
	if table.HaveMaxValue == "否" {
		// 直接添加分区
		return fmt.Sprintf("ALTER TABLE `%s`.`%s` ADD PARTITION (%s)", 
			table.DatabaseName, table.TableName, strings.Join(partitions, ", ")), nil
	} else {
		// 需要reorganize MAXVALUE分区
		// 获取MAXVALUE分区名称
		maxValueName, err := uc.getMaxValuePartitionName(ctx, instance, table.DatabaseName, table.TableName)
		if err != nil {
			return "", err
		}

		// 如果第一个分区名称与MAXVALUE分区名称冲突，重命名
		firstPartName := strings.Split(partitions[0], " ")[1]
		if firstPartName == maxValueName {
			partitions[0] = strings.Replace(partitions[0], firstPartName, "pmax", 1)
		}

		return fmt.Sprintf("ALTER TABLE `%s`.`%s` REORGANIZE PARTITION %s INTO (%s, PARTITION %s VALUES LESS THAN MAXVALUE)",
			table.DatabaseName, table.TableName, maxValueName, strings.Join(partitions, ", "), maxValueName), nil
	}
}

// getNextPartitionDate 获取下一个分区日期和名称
func (uc *PartitionUsecase) getNextPartitionDate(currentDate time.Time, cycle string) (time.Time, string) {
	switch cycle {
	case "天":
		nextDate := currentDate.AddDate(0, 0, 1)
		return nextDate, "p" + currentDate.Format("20060102")
	case "周":
		nextDate := currentDate.AddDate(0, 0, 7)
		return nextDate, "p" + currentDate.Format("20060102")
	case "月":
		nextDate := currentDate.AddDate(0, 1, 0)
		return nextDate, "p" + currentDate.Format("200601")
	case "季":
		nextDate := currentDate.AddDate(0, 3, 0)
		return nextDate, "p" + currentDate.Format("200601")
	case "半年":
		nextDate := currentDate.AddDate(0, 6, 0)
		return nextDate, "p" + currentDate.Format("200601")
	case "年":
		nextDate := currentDate.AddDate(1, 0, 0)
		return nextDate, "p" + currentDate.Format("2006")
	default:
		return time.Time{}, ""
	}
}

// calculatePartitionValue 计算分区值
func (uc *PartitionUsecase) calculatePartitionValue(date time.Time, partPrefix string) int64 {
	baseDate := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	
	if partPrefix == "to_days" {
		diffDays := int64(date.Sub(baseDate).Hours() / 24)
		return diffDays + 730485 // MySQL to_days('2000-01-01') = 730485
	} else if partPrefix == "unix_timestamp" {
		return date.Unix()
	}
	return 0
}

// getMaxValuePartitionName 获取MAXVALUE分区名称
func (uc *PartitionUsecase) getMaxValuePartitionName(ctx context.Context, instance *DatabaseInstance, databaseName, tableName string) (string, error) {
	query := `
		SELECT partition_name
		FROM information_schema.partitions
		WHERE table_schema = ? AND table_name = ?
			AND partition_description = 'MAXVALUE'
		LIMIT 1
	`

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		instance.Username, instance.Password, instance.Host, instance.Port, databaseName)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var name string
	err = db.QueryRowContext(ctx, query, databaseName, tableName).Scan(&name)
	if err != nil {
		return "", err
	}

	return name, nil
}

// executeDDL 执行DDL语句
func (uc *PartitionUsecase) executeDDL(ctx context.Context, instance *DatabaseInstance, databaseName, ddl string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		instance.Username, instance.Password, instance.Host, instance.Port, databaseName)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, ddl)
	return err
}

// AutoDeletePartitions 自动删除分区
func (uc *PartitionUsecase) AutoDeletePartitions(ctx context.Context) error {
	uc.log.Info("开始自动删除分区")

	// 获取需要删除分区的配置
	configs, err := uc.repo.GetPartitionArchiveConfigs(ctx)
	if err != nil {
		return fmt.Errorf("获取分区归档配置失败: %v", err)
	}

	for _, config := range configs {
		if config.IsPause == 1 || config.IsDelete == 0 {
			continue
		}

		if config.IsPartition == 0 {
			// 非分区表，使用pt-archiver删除数据
			continue
		}

		// 分区表，删除过期分区
		if err := uc.deleteExpiredPartitions(ctx, config); err != nil {
			uc.log.Errorf("删除表 %s.%s 的分区失败: %v", config.TableSchema, config.TableName, err)
		}
	}

	uc.log.Info("自动删除分区完成")
	return nil
}

// deleteExpiredPartitions 删除过期分区
func (uc *PartitionUsecase) deleteExpiredPartitions(ctx context.Context, config *PartitionArchiveConfig) error {
	// 获取实例信息
	instance, err := uc.instanceUsecase.GetInstanceByID(ctx, config.InstanceID)
	if err != nil {
		return err
	}

	// 检查归档状态(如果配置了归档)
	if config.IsArchive == 1 {
		// 检查本月是否已归档成功
		archived, err := uc.checkArchiveStatus(ctx, config)
		if err != nil {
			return err
		}
		if !archived {
			uc.log.Infof("表 %s.%s 本月尚未归档，跳过删除", config.TableSchema, config.TableName)
			return nil
		}

		// 如果配置了校验，检查校验状态
		if config.IsChecksum == 1 {
			checksumOK, err := uc.checkChecksumStatus(ctx, config)
			if err != nil {
				return err
			}
			if !checksumOK {
				uc.log.Infof("表 %s.%s 校验未通过，跳过删除", config.TableSchema, config.TableName)
				return nil
			}
		}
	}

	// 获取最小分区
	minPartition, err := uc.getMinPartition(ctx, instance, config.TableSchema, config.TableName, config.ArchField)
	if err != nil {
		return err
	}

	if minPartition == nil {
		return fmt.Errorf("未找到分区")
	}

	// 计算过期时间
	expireDate := time.Now().AddDate(0, 0, -config.ExpireDays)
	
	// 删除所有过期的分区
	for minPartition.Date.Before(expireDate) || minPartition.Date.Equal(expireDate) {
		workID := uc.generateWorkID()
		ddl := fmt.Sprintf("ALTER TABLE `%s`.`%s` DROP PARTITION `%s`", 
			config.TableSchema, config.TableName, minPartition.Name)

		record := &PartitionOperationRecord{
			WorkID:          workID,
			InstanceID:      config.InstanceID,
			InstanceName:    config.InstanceName,
			ConnectionString: config.ConnectionString,
			TableSchema:     config.TableSchema,
			TableName:       config.TableName,
			OperateType:     5, // 分区删除
			ExeStatus:       3, // 执行中
			ExeCmd:          ddl,
			AddUser:         "cron_del_part",
		}

		now := time.Now()
		record.ExeTime = &now
		if err := uc.repo.CreateOperationRecord(ctx, record); err != nil {
			return err
		}

		// 执行删除
		err = uc.executeDDL(ctx, instance, config.TableSchema, ddl)
		if err != nil {
			record.ExeStatus = 5
			record.ExeResult = err.Error()
			uc.repo.UpdateOperationRecord(ctx, record)
			return err
		}

		record.ExeStatus = 4
		record.ExeResult = "成功"
		record.CheckStatus = 1
		uc.repo.UpdateOperationRecord(ctx, record)

		// 获取下一个最小分区
		minPartition, err = uc.getMinPartition(ctx, instance, config.TableSchema, config.TableName, config.ArchField)
		if err != nil {
			break
		}
		if minPartition == nil {
			break
		}
	}

	return nil
}

// PartitionInfo 分区信息
type PartitionInfo struct {
	Name string
	Date time.Time
}

// getMinPartition 获取最小分区
func (uc *PartitionUsecase) getMinPartition(ctx context.Context, instance *DatabaseInstance, databaseName, tableName, archField string) (*PartitionInfo, error) {
	// 获取分区表达式
	query := `
		SELECT DISTINCT partition_expression
		FROM information_schema.partitions
		WHERE table_schema = ? AND table_name = ?
			AND partition_name IS NOT NULL
		LIMIT 1
	`

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		instance.Username, instance.Password, instance.Host, instance.Port, databaseName)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var partExpr string
	err = db.QueryRowContext(ctx, query, databaseName, tableName).Scan(&partExpr)
	if err != nil {
		return nil, err
	}

	// 判断分区函数
	var dateFunc string
	if strings.Contains(strings.ToLower(partExpr), "to_days") {
		dateFunc = "from_days"
	} else if strings.Contains(strings.ToLower(partExpr), "unix_timestamp") {
		dateFunc = "FROM_UNIXTIME"
	} else {
		return nil, fmt.Errorf("不支持的分区函数")
	}

	// 获取最小分区
	query = fmt.Sprintf(`
		SELECT 
			partition_name,
			DATE_FORMAT(%s(MIN(partition_description)), '%%Y-%%m-%%d') as partition_date
		FROM information_schema.partitions
		WHERE table_schema = ? AND table_name = ?
			AND partition_description <> 'MAXVALUE'
	`, dateFunc)

	var name string
	var dateStr string
	err = db.QueryRowContext(ctx, query, databaseName, tableName).Scan(&name, &dateStr)
	if err != nil {
		return nil, err
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}

	return &PartitionInfo{
		Name: name,
		Date: date,
	}, nil
}

// checkArchiveStatus 检查归档状态
func (uc *PartitionUsecase) checkArchiveStatus(ctx context.Context, config *PartitionArchiveConfig) (bool, error) {
	// 查询本月是否已归档成功
	// 这里需要查询PartitionOperationRecord表
	// 简化实现，实际应该查询数据库
	return true, nil
}

// checkChecksumStatus 检查校验状态
func (uc *PartitionUsecase) checkChecksumStatus(ctx context.Context, config *PartitionArchiveConfig) (bool, error) {
	// 查询校验是否成功
	// 简化实现，实际应该查询数据库
	return true, nil
}

// generateWorkID 生成工单ID
func (uc *PartitionUsecase) generateWorkID() string {
	now := time.Now().Format("20060102150405")
	rand.Seed(time.Now().UnixNano())
	random := fmt.Sprintf("%04d", rand.Intn(10000))
	return now + random
}

// CreateConfig 创建分区配置
func (uc *PartitionUsecase) CreateConfig(ctx context.Context, info *PartitionTableInfo) (*PartitionTableInfo, error) {
	if err := uc.repo.CreatePartitionTableInfo(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

// ListConfigs 列出分区配置（分页）
func (uc *PartitionUsecase) ListConfigs(ctx context.Context, instanceID uint64, page, pageSize int) ([]*PartitionTableInfo, int64, error) {
	return uc.repo.ListPartitionTablesPaged(ctx, instanceID, page, pageSize)
}

// GetConfigByID 获取单个分区配置
func (uc *PartitionUsecase) GetConfigByID(ctx context.Context, id uint64) (*PartitionTableInfo, error) {
	return uc.repo.GetPartitionTableInfoByID(ctx, id)
}

// UpdateConfig 更新分区配置
func (uc *PartitionUsecase) UpdateConfig(ctx context.Context, info *PartitionTableInfo) error {
	return uc.repo.UpdatePartitionTableInfo(ctx, info)
}

// DeleteConfig 删除分区配置
func (uc *PartitionUsecase) DeleteConfig(ctx context.Context, id uint64) error {
	return uc.repo.DeletePartitionTableInfo(ctx, id)
}

// ListRecordsByConfigID 获取分区操作记录（按配置ID）
func (uc *PartitionUsecase) ListRecordsByConfigID(ctx context.Context, configID uint64, page, pageSize int) ([]*PartitionOperationRecord, int64, error) {
	return uc.repo.ListOperationRecordsByTableInfoID(ctx, configID, page, pageSize)
}
