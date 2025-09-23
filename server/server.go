package server

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	httphandler "riverproxy/handler/http"
	httpshandler "riverproxy/handler/https"
	"riverproxy/logger"
)

func Start() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// HTTP 代理
	httpHandler := httphandler.GetHTTPHandler(8080)
	wg.Add(1)
	go func(context.Context) {
		defer wg.Done()
		httpHandler.Run(ctx)
	}(ctx)

	// HTTPS 代理（隧道模式）
	httpsHandler := httpshandler.GetHTTPSHandler(8081)
	wg.Add(1)
	go func(context.Context) {
		defer wg.Done()
		httpsHandler.Run(ctx)
	}(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.LogInfo("收到终止信号, 开始关闭服务器")

	cancel()

	wg.Wait()
}
