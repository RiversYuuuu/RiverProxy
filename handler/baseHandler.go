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
	"golang.org/x/sync/semaphore"
)

var sem = semaphore.NewWeighted(50)

type BaseHandler struct {
	Ports         []int
	Protocol      string
	HandleReqFunc func(clientConn net.Conn, req *http.Request, logEntry *logger.AccessLog, connectionID string)
}

func (h *BaseHandler) Run(ctx context.Context) {
	// 创建监听器
	listeners := make([]net.Listener, len(h.Ports))
	for i, port := range h.Ports {
		addr := ":" + strconv.Itoa(port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			logger.LogStartup("监听端口 %s 失败: %v", addr, err)
			os.Exit(1)
		}
		listeners[i] = listener
		defer listener.Close()
	}
	logger.LogStartup("%s 代理服务器启动, 监听 %v", h.Protocol, h.Ports)

	var wg sync.WaitGroup

	// 监听服务中止信号
	go func(listeners []net.Listener) {
		<-ctx.Done()
		for _, l := range listeners {
			l.Close()
		}
	}(listeners)

	// 使用生产者-消费者模型处理多端口连接
	connChan := make(chan net.Conn, 100) // 缓冲通道作为连接队列

	// 启动多个监听器 goroutine，每个监听器负责一个端口
	var listenerWg sync.WaitGroup
	for _, listener := range listeners {
		listenerWg.Add(1)
		go func(l net.Listener) {
			defer listenerWg.Done()
			for {
				conn, err := l.Accept()
				if err != nil {
					select {
					case <-ctx.Done():
						return
					default:
						logger.LogWarn("接受连接失败: %v", err)
						continue
					}
				}

				// 将接受到的连接发送到通道中
				select {
				case connChan <- conn:
				case <-ctx.Done():
					conn.Close()
					return
				}
			}
		}(listener)
	}

	// 处理连接 goroutine
	for {
		select {
		case <-ctx.Done():
			close(connChan)
			listenerWg.Wait()
			wg.Wait()
			logger.LogInfo("%s 服务器已关闭", h.Protocol)
			return
		case conn, ok := <-connChan:
			if !ok {
				continue
			}

			if err := sem.Acquire(ctx, 1); err != nil {
				http.Error(&ResponseWriter{Conn: conn}, "服务繁忙，请稍后再试", http.StatusTooManyRequests)
				conn.Close()
				continue
			}

			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				defer sem.Release(1)
				h.handleConnFunc(c)
			}(conn)
		}
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
