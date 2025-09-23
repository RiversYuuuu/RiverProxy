package handler

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"riverproxy/logger"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

func (h *BaseHandler) Run(ctx context.Context) {
	// 创建监听器
	addr := ":" + strconv.Itoa(h.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.LogStartup("监听端口 %s 失败: %v", addr, err)
		os.Exit(1)
	}
	defer listener.Close()

	logger.LogStartup("%s 代理服务器启动, 监听 %s", h.Protocol, addr)

	var wg sync.WaitGroup

	go func(l net.Listener) {
		<-ctx.Done()
		l.Close()
	}(listener)

	// 监听连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				logger.LogInfo("%s 服务器正在关闭，停止接受新连接", h.Protocol)
				wg.Wait()
				logger.LogInfo("%s 服务器已关闭", h.Protocol)
				return
			default:
				logger.LogWarn("接受连接失败: %v", err)
				continue
			}
		}

		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			h.handleConnFunc(c)
		}(conn)
	}
}

func (h *BaseHandler) handleConnFunc(clientConn net.Conn) {
	startTime := time.Now()
	clientIP := clientConn.RemoteAddr().String()
	var logEntry logger.AccessLog
	logEntry.Timestamp = startTime
	logEntry.ClientIP = clientIP

	// 生成随机ID
	connectionID := uuid.New().String()
	logEntry.ConnectionID = connectionID
	logger.LogDebug("[%s] 新连接建立, 客户端IP: %s", connectionID, clientIP)

	defer func() {
		logEntry.Duration = time.Since(startTime)
		logger.LogAccess(&logEntry)
		clientConn.Close()
	}()

	// 读取客户端请求
	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			logger.LogWarn("[%s] 读取客户端请求失败: %v", connectionID, err)
		}
		return
	}

	h.HandleReqFunc(clientConn, req, &logEntry, connectionID)
}
