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

// Config 顶层配置结构体
type Config struct {
	Database  DatabaseConfig  `mapstructure:"database"`
	MochiMQTT MochiMQTTConfig `mapstructure:"mochi_mqtt"`
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
