package handler

import (
	"net"
	"net/http"
	"riverproxy/logger"
)

type BaseHandler struct {
	Port          int
	Protocol      string
	HandleReqFunc func(clientConn net.Conn, req *http.Request, logEntry *logger.AccessLog, connectionID string)
}

type ResponseWriter struct {
	net.Conn
}
