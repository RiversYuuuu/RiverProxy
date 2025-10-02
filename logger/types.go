package logger

import "time"

// AccessLog 是外部传入的日志结构体（由 proxy 包定义）
type AccessLog struct {
	Timestamp    time.Time     `json:"timestamp"`
	ClientIP     string        `json:"client_ip"`
	Method       string        `json:"method"`
	Host         string        `json:"host"`
	Path         string        `json:"path"`
	StatusCode   int           `json:"status_code"`
	Duration     time.Duration `json:"duration"`
	Bytes        int64         `json:"bytes"`
	UserAgent    string        `json:"user_agent"`
	Referer      string        `json:"referer"`
	Protocol     string        `json:"protocol"`
	ConnectionID string        `json:"connection_id"`
}

// LogLevel 日志级别枚举
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelNotice
	LogLevelWarn
	LogLevelError
)
