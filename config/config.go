package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// 定义结构体，日志配置
type LogConfig struct {
	LogDir        string `yaml:"log_dir"`
	EnableConsole bool   `yaml:"enable_console"`
	MinLevel      string `yaml:"min_level"`
}

// 定义结构体，代理配置
type ProxyConfig struct {
	Protocol string `yaml:"protocol"`
	Port     []int  `yaml:"ports"`
}

// 定义结构体，配置，包含日志配置和代理配置
type Config struct {
	LogCfg   LogConfig     `yaml:"log"`
	ProxyCfg []ProxyConfig `yaml:"proxy"`
}

// 函数：读取配置文件
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
