package main

import (
	"riverproxy/config"
	"riverproxy/logger"
	"riverproxy/server"
)

func main() {
	// 加载配置
	configFile := "config/config.yaml"
	config, err := config.LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	// 初始化日志系统
	logger.Init(config.LogCfg)
	defer logger.Close()

	// 初始化代理服务
	server.Start(config.ProxyCfg)
}
