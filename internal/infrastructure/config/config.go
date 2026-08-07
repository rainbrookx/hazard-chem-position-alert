package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Load 加载配置，返回 Config 结构体
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 设置默认值（兜底）
	setDefault(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败：%w", err)
	}

	// 解析配置文件，反序列化到结构体
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败：%w", err)
	}

	// TODO 校验：必填项，格式是否正确

	return &config, nil
}

func setDefault(v *viper.Viper) {

}

// AlertEngineConfig 告警引擎配置。
type AlertEngineConfig struct {
	MQTTTopic             string `mapstructure:"mqtt_topic"`              // 默认 "position/cleaned"
	DebounceSeconds       int    `mapstructure:"debounce_seconds"`        // 进入去抖窗口（秒），默认 3
	GridCellMeters        int    `mapstructure:"grid_cell_meters"`        // 默认网格大小（米），默认 30
	StateCleanupSeconds   int    `mapstructure:"state_cleanup_seconds"`   // 状态清理间隔（秒），默认 60
	StateTTLSeconds       int    `mapstructure:"state_ttl_seconds"`       // 状态条目 TTL（秒），默认 300
	JWTSecret             string `mapstructure:"jwt_secret"`              // JWT 签名密钥
	DefaultUsername       string `mapstructure:"default_username"`        // 默认管理员用户名
	DefaultPassword       string `mapstructure:"default_password"`        // 默认管理员密码
	RSAKeyBits            int    `mapstructure:"rsa_key_bits"`            // RSA 密钥位数，默认 2048
	OneKeyCooldownSeconds int    `mapstructure:"onekey_cooldown_seconds"` // 一键报警去重冷却（秒），默认 60
}

// Config 顶层配置结构体
type Config struct {
	Database    DatabaseConfig    `mapstructure:"database"`
	MochiMQTT   MochiMQTTConfig   `mapstructure:"mochi_mqtt"`
	HTTPServer  HTTPServerConfig  `mapstructure:"http_server"`
	GRPCServer  GRPCServerConfig  `mapstructure:"grpc_server"`
	AlertEngine AlertEngineConfig `mapstructure:"alert_engine"`
}

type DatabaseConfig struct {
	Default string       `mapstructure:"default"`
	SQLite  SQLiteConfig `mapstructure:"sqlite"`
}

type SQLiteConfig struct {
	DSN string `mapstructure:"dsn"`
}

type MochiMQTTConfig struct {
	Address string `mapstructure:"address"`
}

type HTTPServerConfig struct {
	Port string `mapstructure:"port"`
}

type GRPCServerConfig struct {
	Port string `mapstructure:"port"`
}
