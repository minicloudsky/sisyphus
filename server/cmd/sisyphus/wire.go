//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	"sisyphus/server/internal/biz"
	"sisyphus/server/internal/conf"
	"sisyphus/server/internal/data"
	asynqPkg "sisyphus/server/internal/pkg/asynq"
	"sisyphus/server/internal/server"
	"sisyphus/server/internal/service"
)

// wireApp 初始化应用
func wireApp(*conf.Server, *conf.Data, *conf.Auth, *conf.Inception, *conf.GhOst, *conf.Tools, *conf.CKCheck, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		// 提供AuthUsecase，从配置中获取secret和expire
		provideAuthUsecase,
		// 提供ArchiveUsecase，需要DatabaseInstanceUsecase
		provideArchiveUsecase,
		// 提供PartitionUsecase，需要DatabaseInstanceUsecase
		providePartitionUsecase,
		// 提供GhOstManager，从配置中获取目录
		provideGhOstManager,
		// 提供SQLExecutor，需要GhOstManager
		provideSQLExecutor,
		// 提供InceptionConfig，从配置中获取
		provideInceptionConfig,
		// 提供CKCheckConfig，从配置中获取
		provideCKCheckConfig,
		// 提供NotificationService，从配置中获取
		provideNotificationService,
		// 提供RollbackUsecase，需要InceptionConfig
		provideRollbackUsecase,
		// 提供FavoriteUsecase
		provideFavoriteUsecase,
		// 提供MyBatisGenerator，从配置中获取API URL
		provideMyBatisGenerator,
		// 提供Asynq Server和Client
		provideAsynqServer,
		provideAsynqClient,
		// 提供TaskHandler
		provideTaskHandler,
		// 提供ArchiveTaskScheduler
		provideArchiveTaskScheduler,
		// 提供应用，包含定时任务启动和停止
		provideAppWithCron,
	))
}

// provideAuthUsecase 提供AuthUsecase
func provideAuthUsecase(userRepo biz.UserRepo, auth *conf.Auth) *biz.AuthUsecase {
	secret := auth.JwtSecret
	if secret == "" {
		secret = "default-secret-key"
	}
	expire := auth.JwtExpire
	if expire == 0 {
		expire = 86400 // 默认24小时
	}
	return biz.NewAuthUsecase(userRepo, secret, expire)
}

// provideArchiveUsecase 提供ArchiveUsecase(需要DatabaseInstanceUsecase和工具路径)
func provideArchiveUsecase(repo biz.ArchiveRepo, instanceUsecase *biz.DatabaseInstanceUsecase, tools *conf.Tools, logger log.Logger, cronLock biz.CronLeaderLock) *biz.ArchiveUsecase {
	ptArchiverPath := "/usr/local/bin/pt-archiver"
	if tools != nil && tools.PtArchiverPath != "" {
		ptArchiverPath = tools.PtArchiverPath
	}
	return biz.NewArchiveUsecase(repo, instanceUsecase, ptArchiverPath, logger, cronLock)
}

// providePartitionUsecase 提供PartitionUsecase(需要DatabaseInstanceUsecase)
func providePartitionUsecase(repo biz.PartitionRepo, instanceUsecase *biz.DatabaseInstanceUsecase, logger log.Logger, cronLock biz.CronLeaderLock) *biz.PartitionUsecase {
	return biz.NewPartitionUsecase(repo, instanceUsecase, logger, cronLock)
}

// provideGhOstManager 提供GhOstManager(从配置中获取目录)
func provideGhOstManager(ghost *conf.GhOst, logger log.Logger) *biz.GhOstManager {
	socketDir := ghost.SocketDir
	if socketDir == "" {
		socketDir = "/data/ghost/socket/"
	}
	logDir := ghost.LogDir
	if logDir == "" {
		logDir = "/data/ghost/log/"
	}
	panicDir := ghost.PanicDir
	if panicDir == "" {
		panicDir = "/data/ghost/panic/"
	}
	return biz.NewGhOstManager(socketDir, logDir, panicDir, logger)
}

// provideSQLExecutor 提供SQLExecutor(需要GhOstManager和工具路径)
func provideSQLExecutor(ghostManager *biz.GhOstManager, tools *conf.Tools, logger log.Logger) *biz.SQLExecutor {
	ghOstPath := "/usr/local/bin/gh-ost"
	if tools != nil && tools.GhOstPath != "" {
		ghOstPath = tools.GhOstPath
	}
	return biz.NewSQLExecutor(ghostManager, ghOstPath, logger)
}

// provideInceptionConfig 提供InceptionConfig(从配置中获取)
func provideInceptionConfig(inception *conf.Inception) *biz.InceptionConfig {
	return biz.NewInceptionConfig(
		inception.Host,
		inception.Port,
		inception.Username,
		inception.Password,
		inception.BackupHost,
		inception.BackupPort,
		inception.BackupUsername,
		inception.BackupPassword,
	)
}

// provideCKCheckConfig 提供CKCheckConfig(从配置中获取)
func provideCKCheckConfig(ck *conf.CKCheck) *biz.CKCheckConfig {
	if ck == nil {
		return &biz.CKCheckConfig{}
	}
	return &biz.CKCheckConfig{
		TestHost:     ck.TestHost,
		TestPort:     int(ck.TestPort),
		TestUsername: ck.TestUsername,
		TestPassword: ck.TestPassword,
	}
}

// provideNotificationService 提供NotificationService（通知配置从数据库读取）
func provideNotificationService(repo biz.NotificationConfigRepo, logger log.Logger) *biz.NotificationService {
	return biz.NewNotificationService(repo, logger)
}

// provideRollbackUsecase 提供RollbackUsecase(需要InceptionConfig)
func provideRollbackUsecase(repo biz.RollbackRepo, inceptionConfig *biz.InceptionConfig, logger log.Logger) *biz.RollbackUsecase {
	return biz.NewRollbackUsecase(repo, inceptionConfig, logger)
}

// provideFavoriteUsecase 提供FavoriteUsecase
func provideFavoriteUsecase(repo biz.FavoriteRepo, logger log.Logger) *biz.FavoriteUsecase {
	return biz.NewFavoriteUsecase(repo, logger)
}

// provideMyBatisGenerator 提供MyBatisGenerator(从配置中获取API URL)
func provideMyBatisGenerator(logger log.Logger) *biz.MyBatisGenerator {
	// TODO: 从配置文件中读取MyBatis生成器API URL
	apiURL := "http://java.bejson.com/generator/genCode"
	return biz.NewMyBatisGenerator(apiURL, logger)
}

// provideAsynqServer 提供Asynq服务器
func provideAsynqServer(data *conf.Data, logger log.Logger) *asynqPkg.Server {
	return asynqPkg.NewServer(data.Redis, logger)
}

// provideAsynqClient 提供Asynq客户端
func provideAsynqClient(data *conf.Data, logger log.Logger) *asynqPkg.Client {
	return asynqPkg.NewClient(data.Redis, logger)
}

// provideTaskHandler 提供任务处理器
func provideTaskHandler(
	sqlOrderUsecase *biz.SQLOrderUsecase,
	sqlExecutor *biz.SQLExecutor,
	instanceUsecase *biz.DatabaseInstanceUsecase,
	executorSelector *biz.SQLExecutorSelector,
	inceptionConfig *biz.InceptionConfig,
	ckCheckConfig *biz.CKCheckConfig,
	notificationService *biz.NotificationService,
	rollbackUsecase *biz.RollbackUsecase,
	binlogUsecase *biz.BinlogRollbackUsecase,
	redisAnalysisUsecase *biz.RedisAnalysisUsecase,
	archiveUsecase *biz.ArchiveUsecase,
	logger log.Logger,
) *biz.TaskHandler {
	return biz.NewTaskHandler(
		sqlOrderUsecase,
		sqlExecutor,
		instanceUsecase,
		executorSelector,
		inceptionConfig,
		ckCheckConfig,
		notificationService,
		rollbackUsecase,
		binlogUsecase,
		redisAnalysisUsecase,
		archiveUsecase,
		logger,
	)
}

// provideArchiveTaskScheduler 提供归档任务调度器
func provideArchiveTaskScheduler(
	archiveUsecase *biz.ArchiveUsecase,
	asynqClient *asynqPkg.Client,
	logger log.Logger,
) *service.ArchiveTaskScheduler {
	return service.NewArchiveTaskScheduler(archiveUsecase, asynqClient, logger)
}

// provideAppWithCron 提供应用，包含定时任务启动和停止
func provideAppWithCron(
	logger log.Logger,
	gs *grpc.Server,
	hs *http.Server,
	archiveUsecase *biz.ArchiveUsecase,
	partitionUsecase *biz.PartitionUsecase,
	asynqServer *asynqPkg.Server,
	taskHandler *biz.TaskHandler,
	archiveScheduler *service.ArchiveTaskScheduler,
) (*kratos.App, func(), error) {
	// 注册任务处理器
	asynqServer.RegisterHandler(asynqPkg.TaskTypeSQLOrderExecute, taskHandler.HandleSQLOrderExecute)
	asynqServer.RegisterHandler(asynqPkg.TaskTypeBinlogParse, taskHandler.HandleBinlogParse)
	asynqServer.RegisterHandler(asynqPkg.TaskTypeRedisRDBParse, taskHandler.HandleRedisRDBParse)
	asynqServer.RegisterHandler(asynqPkg.TaskTypeArchiveExecute, taskHandler.HandleArchiveExecute)

	// 设置归档任务调度器
	archiveUsecase.SetScheduler(archiveScheduler)

	// 启动Asynq服务器（在goroutine中运行）
	go func() {
		if err := asynqServer.Start(); err != nil {
			log.NewHelper(logger).Errorf("Asynq服务器启动失败: %v", err)
		}
	}()

	app := kratos.New(
		kratos.ID("sisyphus"),
		kratos.Name("sisyphus"),
		kratos.Version("v1.0.0"),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
	
	// 启动定时任务
	ctx := context.Background()
	if err := archiveUsecase.StartCron(ctx); err != nil {
		return nil, nil, err
	}
	if err := partitionUsecase.StartCron(ctx); err != nil {
		return nil, nil, err
	}
	
	return app, func() {
		// 停止定时任务
		archiveUsecase.StopCron()
		partitionUsecase.StopCron()
		// 关闭Asynq服务器
		asynqServer.Shutdown()
	}, nil
}
