package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/wumitech-com/mdcp_common/logger"
	"gopkg.in/yaml.v2"
)

var (
	config     *Config
	configOnce sync.Once
)

// Config 配置结构体
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Logging logger.Config `yaml:"logging"`
	Ubuntu  UbuntuConfig  `yaml:"ubuntu"`
	Phone   PhoneConfig   `yaml:"phone"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	GRPC GRPCConfig `yaml:"grpc"`
}

// GRPCConfig gRPC 服务器配置
type GRPCConfig struct {
	Host                  string `yaml:"host"`
	Port                  int    `yaml:"port"`
	MaxReceiveMessageSize int    `yaml:"max_receive_message_size"`
	MaxSendMessageSize    int    `yaml:"max_send_message_size"`
	Keepalive             struct {
		Time                int  `yaml:"time"`
		Timeout             int  `yaml:"timeout"`
		PermitWithoutStream bool `yaml:"permit_without_stream"`
	} `yaml:"keepalive"`
}

// UbuntuConfig Ubuntu服务器配置
type UbuntuConfig struct {
	ExternalIP string `yaml:"external_ip"` // 外网IP地址
	TargetPort string `yaml:"target_port"` // 目标端口
	TableName  string `yaml:"table_name"`  // nftables表名
	ChainName  string `yaml:"chain_name"`  // nftables链名
}

// PhoneConfig 云手机操作配置
type PhoneConfig struct {
	ADBPort          int     `yaml:"adb_port"`           // ADB端口
	PingTimeout      int     `yaml:"ping_timeout"`       // Ping超时时间（秒）
	ADBTimeout       int     `yaml:"adb_timeout"`        // ADB超时时间（秒）
	LatencyThreshold float64 `yaml:"latency_threshold"` // Ping延迟阈值（毫秒）
}

var (
	configInstance *Config
	configMutex    sync.RWMutex
)

// LoadConfig 加载配置文件
func LoadConfig(configPath string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	configInstance = &cfg
	return nil
}

// GetConfig 获取配置
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return configInstance
}

