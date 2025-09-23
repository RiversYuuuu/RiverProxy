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

	host := req.URL.Host
	if host == "" {
		host = req.Host
	}

	if !strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host += ":80"
	}

	targetConn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		logger.LogError("[%s] 连接目标服务器 %s 失败: %v", connectionID, host, err)
		http.Error(&handler.ResponseWriter{Conn: clientConn}, "Failed to connect to target", http.StatusBadGateway)
		logEntry.StatusCode = 502
		return
	}
	defer targetConn.Close()

	err = req.Write(targetConn)
	if err != nil {
		logger.LogError("[%s] 转发HTTP请求失败: %v", connectionID, err)
		logEntry.StatusCode = 502
		return
	}

	targetReader := bufio.NewReader(targetConn)
	resp, err := http.ReadResponse(targetReader, req)
	if err != nil {
		logger.LogError("[%s] 读取HTTP响应失败: %v", connectionID, err)
		logEntry.StatusCode = 502
		return
	}
	defer resp.Body.Close()

	logEntry.StatusCode = resp.StatusCode

	bytesWritten, err := io.Copy(clientConn, resp.Body)
	if err != nil {
		logger.LogError("[%s] 转发HTTP响应失败: %v", connectionID, err)
		return
	}
	logEntry.Bytes = bytesWritten
}
