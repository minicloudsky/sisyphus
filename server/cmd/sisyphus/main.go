package main

import (
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	kratoszap "github.com/go-kratos/kratos/contrib/log/zap/v2"
	"go.uber.org/zap"

	_ "go.uber.org/automaxprocs"

	"sisyphus/server/internal/conf"
	"sisyphus/server/internal/pkg/telemetry"
)

var (
	// Name is the name of the compiled software.
	Name = "sisyphus"
	// Version is the version of the compiled software.
	Version = "v1.0.0"
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	logger := kratoszap.NewLogger(zapLogger)

	// load configs
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// 初始化OpenTelemetry
	if bc.Telemetry != nil && bc.Telemetry.Enabled {
		serviceName := bc.Telemetry.ServiceName
		if serviceName == "" {
			serviceName = "sisyphus"
		}
		serviceVersion := bc.Telemetry.ServiceVersion
		if serviceVersion == "" {
			serviceVersion = "v1.0.0"
		}
		cleanupTelemetry, err := telemetry.InitTelemetry(serviceName, serviceVersion, bc.Telemetry.OTLPEndpoint)
		if err != nil {
			log.NewHelper(logger).Errorf("初始化OpenTelemetry失败: %v", err)
		} else {
			defer cleanupTelemetry()
		}
	}

	app, cleanup, err := wireApp(bc.Server, bc.Data, bc.Auth, bc.Inception, bc.GhOst, bc.Tools, bc.CKCheck, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// 启动定时任务（需要在wire中获取archiveUsecase）
	// 由于wire的限制，这里暂时注释，实际应该通过wire注入
	// archiveUsecase.StartCron(context.Background())

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
