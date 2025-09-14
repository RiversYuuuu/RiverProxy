package proxy

import (
	"bufio"
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"riverproxy/logger"
)

func HttpProxy() {
	// 1. 创建带取消功能的 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. 监听 8080 端口
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		logger.LogError("监听端口 :8080 失败: %v", err)
		os.Exit(1)
	}
	defer listener.Close()

	logger.LogInfo("HTTP/HTTPS 代理服务器启动，监听 :8080")

	// 3. 监听系统信号用于优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 4. 用于等待所有连接处理完成
	var wg sync.WaitGroup

	// 5. 启动 goroutine 处理系统信号
	go func() {
		sig := <-sigChan
		logger.LogInfo("接收到信号 %v，开始优雅关闭服务器", sig)
		cancel()         // 取消所有子 context
		listener.Close() // 关闭监听器，使 Accept() 返回错误
	}()

	// 6. 循环接受连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			// 检查是否是由于关闭监听器导致的错误
			select {
			case <-ctx.Done():
				logger.LogInfo("服务器正在关闭，停止接受新连接")
				wg.Wait() // 等待所有现有连接处理完成
				logger.LogInfo("所有连接处理完成，服务器已关闭")
				return
			default:
				logger.LogWarn("接受连接失败: %v", err)
				continue
			}
		}

		// 7. 使用 goroutine 处理每个连接
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			handle_connection(c)
		}(conn)
	}
}

func handle_connection(clientConn net.Conn) {
	startTime := time.Now()
	clientIP := clientConn.RemoteAddr().String()
	var logEntry logger.AccessLog
	logEntry.Timestamp = startTime
	logEntry.ClientIP = clientIP

	// 生成随机 ID
	connectionID := generateRandomID()
	logEntry.ConnectionID = connectionID // 新增字段存储随机 ID
	logger.LogDebug("新连接建立: ID=%s, ClientIP=%s", connectionID, clientIP)

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
			logger.LogWarn("ID=%s, 读取客户端请求失败: %v", connectionID, err)
		}
		return
	}

	// 根据方法类型处理不同请求
	if req.Method == "CONNECT" {
		// HTTPS 代理处理
		handle_https_proxy(clientConn, req, &logEntry, connectionID)
	} else {
		// HTTP 代理处理
		handle_http_proxy(clientConn, req, &logEntry, connectionID)
	}
}

func handle_https_proxy(clientConn net.Conn, req *http.Request, logEntry *logger.AccessLog, connectionID string) {
	logEntry.Method = "CONNECT"
	logEntry.Host = req.Host
	logEntry.IsHTTPS = true
	logEntry.Path = "/"

	// 获取目标主机地址
	target := req.Host
	if target == "" {
		target = req.URL.Host
	}

	// 如果没有端口，添加默认的 443
	if !strings.Contains(target, ":") {
		target += ":443"
	}

	logger.LogDebug("ID=%s, HTTPS CONNECT 请求: %s", connectionID, target)

	// 连接到目标服务器
	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		logger.LogError("ID=%s, 连接目标服务器 %s 失败: %v", connectionID, target, err)
		http.Error(&responseWriter{clientConn}, "Failed to connect to target", http.StatusBadGateway)
		logEntry.StatusCode = 502
		return
	}
	defer targetConn.Close()

	// 发送连接建立成功的响应
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		logger.LogError("ID=%s, 发送 CONNECT 响应失败: %v", connectionID, err)
		logEntry.StatusCode = 500
		return
	}

	logEntry.StatusCode = 200
	logger.LogDebug("ID=%s, HTTPS 隧道建立成功: %s", connectionID, target)

	// 双向转发数据（建立隧道）
	go transfer_data(targetConn, clientConn, connectionID)
	transfer_data(clientConn, targetConn, connectionID)
}

func handle_http_proxy(clientConn net.Conn, req *http.Request, logEntry *logger.AccessLog, connectionID string) {
	logEntry.Method = req.Method
	logEntry.Host = req.Host
	logEntry.Path = req.URL.Path
	logEntry.Protocol = req.Proto
	logEntry.UserAgent = req.Header.Get("User-Agent")
	logEntry.Referer = req.Header.Get("Referer")
	logEntry.IsHTTPS = false

	// 获取目标主机地址
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}

	// 添加默认端口
	if !strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host += ":80"
	}

	logger.LogInfo("ID=%s, HTTP 代理请求: %s %s", connectionID, req.Method, host)

	// 连接目标服务器
	targetConn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		logger.LogError("ID=%s, 连接目标服务器 %s 失败: %v", connectionID, host, err)
		http.Error(&responseWriter{clientConn}, "Failed to connect to target", http.StatusBadGateway)
		logEntry.StatusCode = 502
		return
	}
	defer targetConn.Close()

	// 重建请求 URL
	req.URL.Scheme = "http"
	req.URL.Host = host

	// 转发请求到目标服务器
	err = req.Write(targetConn)
	if err != nil {
		logger.LogError("ID=%s, 转发请求失败: %v", connectionID, err)
		logEntry.StatusCode = 502
		return
	}

	// 读取目标服务器响应
	targetReader := bufio.NewReader(targetConn)
	resp, err := http.ReadResponse(targetReader, req)
	if err != nil {
		logger.LogError("ID=%s, 读取响应失败: %v", connectionID, err)
		logEntry.StatusCode = 502
		return
	}
	defer resp.Body.Close()

	// 更新日志信息
	logEntry.StatusCode = resp.StatusCode

	// 转发响应给客户端并计算字节数
	bytesWritten, err := io.Copy(clientConn, resp.Body)
	if err != nil {
		logger.LogError("ID=%s, 转发响应失败: %v", connectionID, err)
		return
	}
	logEntry.Bytes = bytesWritten

	logger.LogDebug("ID=%s, HTTP 代理完成: %s %s -> %d (%d bytes)", connectionID, req.Method, host, resp.StatusCode, bytesWritten)
}

func transfer_data(dst, src net.Conn, connectionID string) {
	defer src.Close()
	defer dst.Close()

	// 设置超时
	src.SetReadDeadline(time.Now().Add(60 * time.Second))
	dst.SetWriteDeadline(time.Now().Add(60 * time.Second))

	// 双向转发数据
	_, err := io.Copy(dst, src)
	if err != nil {
		if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
			logger.LogWarn("ID=%s, 数据转发错误: %v", connectionID, err)
		}
	}
}

// 生成随机 ID
func generateRandomID() string {
	return randomString(32)
}

// 生成随机字符串
var letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
var src = rand.NewSource(time.Now().UnixNano())

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[src.Int63()%int64(len(letters))]
	}
	return string(b)
}

// 用于 http.Error 的简单响应写入器
type responseWriter struct {
	net.Conn
}

func (rw *responseWriter) Header() http.Header {
	return make(http.Header)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	return rw.Conn.Write(data)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	// 简单实现，实际可以更完善
	rw.Conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
}
