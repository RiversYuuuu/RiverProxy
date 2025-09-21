package handler

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"riverproxy/handler"
	"riverproxy/logger"
)

func GetHTTPHandler(port int) *handler.BaseHandler {
	httpHandler := &handler.BaseHandler{
		Port:     port,
		Protocol: "http",
	}
	httpHandler.HandleReqFunc = handleHTTPRequest
	return httpHandler
}

func handleHTTPRequest(clientConn net.Conn, req *http.Request, logEntry *logger.AccessLog, connectionID string) {
	logEntry.Method = req.Method
	logEntry.Host = req.Host
	logEntry.Path = req.URL.Path
	logEntry.Protocol = req.Proto
	logEntry.UserAgent = req.Header.Get("User-Agent")
	logEntry.Referer = req.Header.Get("Referer")
	logEntry.IsHTTPS = false

	host := req.URL.Host
	if host == "" {
		host = req.Host
	}

	if !strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host += ":80"
	}

	logger.LogInfo("[%s] HTTP 代理请求: %s %s", connectionID, req.Method, host)

	targetConn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		logger.LogError("[%s] 连接目标服务器 %s 失败: %v", connectionID, host, err)
		http.Error(&handler.ResponseWriter{Conn: clientConn}, "Failed to connect to target", http.StatusBadGateway)
		logEntry.StatusCode = 502
		return
	}
	defer targetConn.Close()

	req.URL.Scheme = "http"
	req.URL.Host = host

	err = req.Write(targetConn)
	if err != nil {
		logger.LogError("[%s] 转发请求失败: %v", connectionID, err)
		logEntry.StatusCode = 502
		return
	}

	targetReader := bufio.NewReader(targetConn)
	resp, err := http.ReadResponse(targetReader, req)
	if err != nil {
		logger.LogError("[%s] 读取响应失败: %v", connectionID, err)
		logEntry.StatusCode = 502
		return
	}
	defer resp.Body.Close()

	logEntry.StatusCode = resp.StatusCode

	bytesWritten, err := io.Copy(clientConn, resp.Body)
	if err != nil {
		logger.LogError("[%s] 转发响应失败: %v", connectionID, err)
		return
	}
	logEntry.Bytes = bytesWritten

	logger.LogDebug("[%s] HTTP 代理完成: %s %s -> %d (%d bytes)", connectionID, req.Method, host, resp.StatusCode, bytesWritten)
}
