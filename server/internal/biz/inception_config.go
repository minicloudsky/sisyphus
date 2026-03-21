package biz

// InceptionConfig goInception配置
type InceptionConfig struct {
	Host           string
	Port           int32
	Username       string
	Password       string
	BackupHost     string
	BackupPort     int32
	BackupUsername string
	BackupPassword string
}

// NewInceptionConfig 从配置创建InceptionConfig
func NewInceptionConfig(host string, port int32, username, password, backupHost string, backupPort int32, backupUsername, backupPassword string) *InceptionConfig {
	return &InceptionConfig{
		Host:           host,
		Port:           port,
		Username:       username,
		Password:       password,
		BackupHost:     backupHost,
		BackupPort:     backupPort,
		BackupUsername: backupUsername,
		BackupPassword: backupPassword,
	}
}
