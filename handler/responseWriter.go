package handler

import (
	"net"
	"net/http"
)

type ResponseWriter struct {
	net.Conn
}

func (rw *ResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (rw *ResponseWriter) Write(data []byte) (int, error) {
	return rw.Conn.Write(data)
}

func (rw *ResponseWriter) WriteHeader(statusCode int) {
	// 简单实现，实际可以更完善
	rw.Conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
}
