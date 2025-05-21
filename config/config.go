package config

import (
	"os"
	"strconv"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Port              string
	MaxUploadFiles    int
	MaxFileSize       int64
	DefaultUser       string
	DefaultApiTimeout int
	Environment       string
}

// Config 应用配置
var Config = &ServerConfig{
	Port:              "3010",
	MaxUploadFiles:    10,                // 最多可上传10个文件
	MaxFileSize:       100 * 1024 * 1024, // 默认最大100MB，实际由Dify接口限制
	DefaultUser:       "user",
	DefaultApiTimeout: 120, // 默认API超时时间（秒）
	Environment:       "development",
}

// GetPort 获取服务端口
func GetPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return Config.Port
}

// GetEnvironment 获取当前环境
func GetEnvironment() string {
	if env := os.Getenv("GO_ENV"); env != "" {
		return env
	}
	return Config.Environment
}

// InitConfig 初始化配置
func InitConfig() {
	// 设置环境
	if env := os.Getenv("GO_ENV"); env != "" {
		Config.Environment = env
	}

	if maxFiles := os.Getenv("MAX_UPLOAD_FILES"); maxFiles != "" {
		if val, err := strconv.Atoi(maxFiles); err == nil && val > 0 {
			Config.MaxUploadFiles = val
		}
	}

	if defaultUser := os.Getenv("DEFAULT_USER"); defaultUser != "" {
		Config.DefaultUser = defaultUser
	}

	if timeout := os.Getenv("API_TIMEOUT"); timeout != "" {
		if val, err := strconv.Atoi(timeout); err == nil && val > 0 {
			Config.DefaultApiTimeout = val
		}
	}
}
