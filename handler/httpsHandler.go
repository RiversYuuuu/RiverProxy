package handler

import (
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"riverproxy/logger"
)

func GetHTTPSHandler(port []int) *BaseHandler {
	httpsHandler := &BaseHandler{
		Ports:    port,
		Protocol: "https",
	}
	httpsHandler.HandleReqFunc = handleHTTPSRequest
	return httpsHandler
}

func handleHTTPSRequest(clientConn net.Conn, req *http.Request, logEntry *logger.AccessLog, connectionID string) {
	logEntry.Method = "CONNECT"
	logEntry.Host = req.Host
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

	// 连接到目标服务器
	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		logger.LogError("[%s] 连接目标服务器 %s 失败: %v", connectionID, target, err)
		http.Error(&ResponseWriter{Conn: clientConn}, "Failed to connect to target", http.StatusBadGateway)
		logEntry.StatusCode = 502
		return
	}
	defer targetConn.Close()

	// 发送连接建立成功的响应
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		logger.LogError("[%s] 发送 CONNECT 响应失败: %v", connectionID, err)
		logEntry.StatusCode = 500
		return
	}

	logEntry.StatusCode = 200
	logger.LogDebug("[%s] HTTPS 隧道建立成功: %s", connectionID, target)

	// 双向转发数据（建立隧道）
	go transfer_data(targetConn, clientConn, connectionID)
	transfer_data(clientConn, targetConn, connectionID)
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
		switch {
		case err == io.EOF:
			logger.LogDebug("[%s] 数据转发错误: %v", connectionID, err)

		case strings.Contains(err.Error(), "use of closed network connection"):
			logger.LogDebug("[%s] 数据转发错误: %v", connectionID, err)

		case errors.Is(err, os.ErrDeadlineExceeded):
			logger.LogDebug("[%s] 数据转发错误: %v", connectionID, err)

		default:
			logger.LogWarn("[%s] 数据转发错误: %v", connectionID, err)
		}
	}
}
