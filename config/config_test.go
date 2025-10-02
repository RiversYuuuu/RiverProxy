package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 直接使用项目中的配置文件进行测试
	config, err := LoadConfig("config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// 验证日志配置
	if config.LogCfg.LogDir != "/var/log/riverproxy" {
		t.Errorf("Expected LogDir '/var/log/riverproxy', got '%s'", config.LogCfg.LogDir)
	}

	if !config.LogCfg.EnableConsole {
		t.Errorf("Expected EnableConsole true, got %v", config.LogCfg.EnableConsole)
	}

	if config.LogCfg.MinLevel != "INFO" {
		t.Errorf("Expected MinLevel 'INFO', got '%s'", config.LogCfg.MinLevel)
	}

	// 验证代理配置
	if len(config.ProxyCfg) != 3 {
		t.Fatalf("Expected 3 proxy configs, got %d", len(config.ProxyCfg))
	}

	if config.ProxyCfg[0].Protocol != "http" {
		t.Errorf("Expected first proxy protocol 'http', got '%s'", config.ProxyCfg[0].Protocol)
	}

	if len(config.ProxyCfg[0].Port) != 1 || config.ProxyCfg[0].Port[0] != 8080 {
		t.Errorf("Expected first proxy port [8080], got %v", config.ProxyCfg[0].Port)
	}

	if config.ProxyCfg[1].Protocol != "https" {
		t.Errorf("Expected second proxy protocol 'https', got '%s'", config.ProxyCfg[1].Protocol)
	}

	if len(config.ProxyCfg[1].Port) != 1 || config.ProxyCfg[1].Port[0] != 8443 {
		t.Errorf("Expected second proxy port [8443], got %v", config.ProxyCfg[1].Port)
	}

	if config.ProxyCfg[2].Protocol != "aggregate" {
		t.Errorf("Expected third proxy protocol 'aggregate', got '%s'", config.ProxyCfg[2].Protocol)
	}

	if len(config.ProxyCfg[2].Port) != 1 || config.ProxyCfg[2].Port[0] != 8081 {
		t.Errorf("Expected third proxy port [8081], got %v", config.ProxyCfg[2].Port)
	}

	// 测试错误情况：文件不存在
	_, err = LoadConfig("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error when loading nonexistent file, got nil")
	}
}
