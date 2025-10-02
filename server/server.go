package server

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"riverproxy/config"
	"riverproxy/handler"
	"riverproxy/logger"
)

func Start(proxyCfg []config.ProxyConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for _, cfg := range proxyCfg {
		switch cfg.Protocol {
		case "http":
			httpHandler := handler.GetHTTPHandler(cfg.Port)
			wg.Add(1)
			go func(context.Context) {
				defer wg.Done()
				httpHandler.Run(ctx)
			}(ctx)
		case "https":
			httpsHandler := handler.GetHTTPSHandler(cfg.Port)
			wg.Add(1)
			go func(context.Context) {
				defer wg.Done()
				httpsHandler.Run(ctx)
			}(ctx)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.LogInfo("收到终止信号, 开始关闭服务器")

	cancel()

	wg.Wait()
}
