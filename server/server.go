package server

import (
	"context"
	"log"

	httphandler "riverproxy/handler/http"
	httpshandler "riverproxy/handler/https"
)

func Start() {
	log.Println("代理服务器启动...")
	ctx := context.Background()

	// HTTP 代理
	httpHandler := httphandler.GetHTTPHandler(8080)
	go httpHandler.Run(ctx)

	// HTTPS 代理（隧道模式）
	httpsHandler := httpshandler.GetHTTPSHandler(8081)
	go httpsHandler.Run(ctx)

	// 如果你有 SOCKS5 或其他，继续加
	// go startListener(":1080", &proxy.SOCKS5Proxy{})

	// 阻塞主 goroutine
	select {}
}
