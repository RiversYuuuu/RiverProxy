package main

import (
	"riverproxy/logger"
	"riverproxy/proxy"
)

func main() {
	loggerConfig := "/home/junjyu/Documents/project/RiverProxy/config/log.yaml"
	logger.Init(loggerConfig)
	proxy.HttpProxy()
}
