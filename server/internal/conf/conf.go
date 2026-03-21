package conf

import (
	"time"

	"github.com/go-kratos/kratos/v2/config"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Bootstrap 配置引导
type Bootstrap struct {
	Server    *Server    `yaml:"server"`
	Data      *Data      `yaml:"data"`
	Auth      *Auth      `yaml:"auth"`
	Inception *Inception `yaml:"inception"`
	GhOst     *GhOst     `yaml:"ghost"`
	Tools     *Tools     `yaml:"tools"`
	Telemetry *Telemetry `yaml:"telemetry"`
	CKCheck   *CKCheck   `yaml:"ckcheck"`
}

// Server 服务器配置
type Server struct {
	HTTP *HTTP `yaml:"http"`
	GRPC *GRPC `yaml:"grpc"`
}

// HTTP HTTP配置
type HTTP struct {
	Network string        `yaml:"network"`
	Addr    string        `yaml:"addr"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
}

// GRPC gRPC配置
type GRPC struct {
	Network string        `yaml:"network"`
	Addr    string        `yaml:"addr"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
}

// Data 数据配置
type Data struct {
	Database *Database `yaml:"database"`
	Redis    *Redis    `yaml:"redis"`
}

// Database 数据库配置
type Database struct {
	Driver string `yaml:"driver"`
	Source string `yaml:"source"`
}

// Redis Redis配置
type Redis struct {
	Network      string        `yaml:"network"`
	Addr         string        `yaml:"addr"`
	Password     string        `yaml:"password"`
	Db           int32         `yaml:"db"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// Auth 认证配置
type Auth struct {
	JwtSecret string `yaml:"jwt_secret"`
	JwtExpire int64  `yaml:"jwt_expire"`
}

// Inception goInception配置
type Inception struct {
	Host           string `yaml:"host"            json:"host"`
	Port           int32  `yaml:"port"            json:"port"`
	Username       string `yaml:"username"        json:"username"`
	Password       string `yaml:"password"        json:"password"`
	BackupHost     string `yaml:"backup_host"     json:"backup_host"`
	BackupPort     int32  `yaml:"backup_port"     json:"backup_port"`
	BackupUsername string `yaml:"backup_username" json:"backup_username"`
	BackupPassword string `yaml:"backup_password" json:"backup_password"`
}

// GhOst gh-ost配置
type GhOst struct {
	SocketDir string `yaml:"socket_dir"`
	LogDir    string `yaml:"log_dir"`
	PanicDir  string `yaml:"panic_dir"`
	BinaryPath string `yaml:"binary_path"`
}

// Tools 工具二进制文件路径配置
type Tools struct {
	PtArchiverPath  string `yaml:"pt_archiver_path"`
	GhOstPath       string `yaml:"gh_ost_path"`
	GoInceptionPath string `yaml:"goinception_path"`
}

// Telemetry OpenTelemetry配置
type Telemetry struct {
	Enabled       bool   `yaml:"enabled"`
	OTLPEndpoint  string `yaml:"otlp_endpoint"`
	ServiceName   string `yaml:"service_name"`
	ServiceVersion string `yaml:"service_version"`
}

// CKCheck ClickHouse审核测试实例配置
type CKCheck struct {
	TestHost     string `yaml:"test_host"     json:"test_host"`
	TestPort     int32  `yaml:"test_port"     json:"test_port"`
	TestUsername string `yaml:"test_username" json:"test_username"`
	TestPassword string `yaml:"test_password" json:"test_password"`
}

// Notification 通知配置
type Notification struct {
	Email    *Email    `yaml:"email"`
	DingTalk *DingTalk `yaml:"dingtalk"`
}

// Email 邮件配置
type Email struct {
	Host      string   `yaml:"host"`
	Port      int32    `yaml:"port"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
	Receivers []string `yaml:"receivers"`
}

// DingTalk 钉钉配置
type DingTalk struct {
	Webhook string   `yaml:"webhook"`
	Phones  []string `yaml:"phones"`
}

// Load 加载配置
func Load(c config.Config) (*Bootstrap, error) {
	var bc Bootstrap
	if err := c.Scan(&bc); err != nil {
		return nil, err
	}
	return &bc, nil
}

// AsDuration 转换为duration
func (d *Data) AsDuration() *durationpb.Duration {
	return durationpb.New(d.Redis.DialTimeout)
}
