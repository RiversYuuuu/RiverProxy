package main

import (
	"riverproxy/logger"
	"riverproxy/proxy"
)

func main() {
	// 初始化日志系统
	loggerConfig := "/home/junjyu/Documents/project/RiverProxy/config/log.yaml"
	logger.Init(loggerConfig)
	defer logger.Close()

	proxy.Start()
}
