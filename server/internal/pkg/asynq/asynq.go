package asynq

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"

	"sisyphus/server/internal/conf"
)

// TaskType 任务类型常量
const (
	// SQL工单执行任务
	TaskTypeSQLOrderExecute = "sql_order:execute"
	
	// Binlog解析任务
	TaskTypeBinlogParse = "binlog:parse"
	
	// Redis RDB解析任务
	TaskTypeRedisRDBParse = "redis_rdb:parse"
	
	// 数据归档任务
	TaskTypeArchiveExecute = "archive:execute"
)

// Server Asynq服务器
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	log    *log.Helper
}

// Client Asynq客户端
type Client struct {
	client *asynq.Client
	log    *log.Helper
}

// NewServer 创建Asynq服务器
func NewServer(redisConf *conf.Redis, logger log.Logger) *Server {
	// 创建Redis连接选项
	redisOpt := asynq.RedisClientOpt{
		Addr:         redisConf.Addr,
		Password:     redisConf.Password,
		DB:           int(redisConf.Db),
		DialTimeout:  redisConf.DialTimeout,
		ReadTimeout:  redisConf.ReadTimeout,
		WriteTimeout: redisConf.WriteTimeout,
	}

	// 创建Asynq服务器配置
	config := asynq.Config{
		Concurrency: 10, // 默认并发数
		Queues: map[string]int{
			"critical": 6, // SQL工单执行（高优先级）
			"default":  3, // 默认队列
			"low":      1, // 低优先级任务
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			log.NewHelper(logger).Errorf("任务执行失败: type=%s, payload=%s, error=%v", task.Type(), string(task.Payload()), err)
		}),
		Logger: &asynqLogger{logger: logger},
	}

	srv := asynq.NewServer(redisOpt, config)
	mux := asynq.NewServeMux()

	return &Server{
		server: srv,
		mux:    mux,
		log:    log.NewHelper(logger),
	}
}

// NewClient 创建Asynq客户端
func NewClient(redisConf *conf.Redis, logger log.Logger) *Client {
	redisOpt := asynq.RedisClientOpt{
		Addr:         redisConf.Addr,
		Password:     redisConf.Password,
		DB:           int(redisConf.Db),
		DialTimeout:  redisConf.DialTimeout,
		ReadTimeout:  redisConf.ReadTimeout,
		WriteTimeout: redisConf.WriteTimeout,
	}

	client := asynq.NewClient(redisOpt)

	return &Client{
		client: client,
		log:    log.NewHelper(logger),
	}
}

// RegisterHandler 注册任务处理器
func (s *Server) RegisterHandler(taskType string, handler func(context.Context, *asynq.Task) error) {
	s.mux.HandleFunc(taskType, handler)
}

// Start 启动Asynq服务器
func (s *Server) Start() error {
	s.log.Infof("启动Asynq服务器...")
	return s.server.Run(s.mux)
}

// Shutdown 关闭Asynq服务器
func (s *Server) Shutdown() {
	s.log.Infof("关闭Asynq服务器...")
	s.server.Shutdown()
}

// Enqueue 入队任务
func (c *Client) Enqueue(taskType string, payload []byte, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	task := asynq.NewTask(taskType, payload, opts...)
	info, err := c.client.Enqueue(task)
	if err != nil {
		c.log.Errorf("入队任务失败: type=%s, error=%v", taskType, err)
		return nil, err
	}
	c.log.Infof("任务已入队: type=%s, id=%s", taskType, info.ID)
	return info, nil
}

// EnqueueWithDelay 延迟入队任务
func (c *Client) EnqueueWithDelay(taskType string, payload []byte, delay time.Duration, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = append(opts, asynq.ProcessIn(delay))
	return c.Enqueue(taskType, payload, opts...)
}

// GetTaskInfo 获取任务信息（需要传入Redis配置）
func (c *Client) GetTaskInfo(redisConf *conf.Redis, queue, taskID string) (*asynq.TaskInfo, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:         redisConf.Addr,
		Password:     redisConf.Password,
		DB:           int(redisConf.Db),
		DialTimeout:  redisConf.DialTimeout,
		ReadTimeout:  redisConf.ReadTimeout,
		WriteTimeout: redisConf.WriteTimeout,
	}
	inspector := asynq.NewInspector(redisOpt)
	defer inspector.Close()
	
	return inspector.GetTaskInfo(queue, taskID)
}

// DeleteTask 删除任务（需要传入Redis配置）
func (c *Client) DeleteTask(redisConf *conf.Redis, queue, taskID string) error {
	redisOpt := asynq.RedisClientOpt{
		Addr:         redisConf.Addr,
		Password:     redisConf.Password,
		DB:           int(redisConf.Db),
		DialTimeout:  redisConf.DialTimeout,
		ReadTimeout:  redisConf.ReadTimeout,
		WriteTimeout: redisConf.WriteTimeout,
	}
	inspector := asynq.NewInspector(redisOpt)
	defer inspector.Close()
	
	return inspector.DeleteTask(queue, taskID)
}

// Close 关闭客户端
func (c *Client) Close() error {
	return c.client.Close()
}

// asynqLogger Asynq日志适配器
type asynqLogger struct {
	logger log.Logger
}

func (l *asynqLogger) Debug(args ...interface{}) {
	log.NewHelper(l.logger).Debug(args...)
}

func (l *asynqLogger) Info(args ...interface{}) {
	log.NewHelper(l.logger).Info(args...)
}

func (l *asynqLogger) Warn(args ...interface{}) {
	log.NewHelper(l.logger).Warn(args...)
}

func (l *asynqLogger) Error(args ...interface{}) {
	log.NewHelper(l.logger).Error(args...)
}

func (l *asynqLogger) Fatal(args ...interface{}) {
	log.NewHelper(l.logger).Fatal(args...)
}
