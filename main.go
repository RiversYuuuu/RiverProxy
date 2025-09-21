package main

import (
	"riverproxy/logger"
	"riverproxy/server"
)

func main() {
	// 初始化日志系统
	loggerConfig := "/home/junjyu/Documents/project/RiverProxy/config/log.yaml"
	logger.Init(loggerConfig)
	defer logger.Close()

	// 启动服务
	server.Start()
}
