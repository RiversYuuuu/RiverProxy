package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"riverproxy/config"
	"strings"
)

var (
	accessLogger  *log.Logger
	startupLogger *log.Logger
	serviceLogger *log.Logger

	// 最小日志级别（用于 service.log 过滤）
	minLevel LogLevel
)

// parseLogLevel 解析日志级别
func parseLogLevel(s string) (LogLevel, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "DEBUG":
		return LogLevelDebug, nil
	case "INFO":
		return LogLevelInfo, nil
	case "NOTICE":
		return LogLevelNotice, nil
	case "WARN", "WARNING":
		return LogLevelWarn, nil
	case "ERROR":
		return LogLevelError, nil
	default:
		return -1, fmt.Errorf("无效的日志级别: %s", s)
	}
}

// --- 公共接口 ---

// Init 初始化日志系统
func Init(logCfg config.LogConfig) {
	// 创建 logs 目录
	accessLogPath := logCfg.LogDir + "/access.log"
	if err := os.MkdirAll(filepath.Dir(accessLogPath), 0755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}
	startupLogPath := logCfg.LogDir + "/startup.log"
	if err := os.MkdirAll(filepath.Dir(startupLogPath), 0755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}
	serviceLogPath := logCfg.LogDir + "/service.log"
	if err := os.MkdirAll(filepath.Dir(serviceLogPath), 0755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}

	// 打开文件
	openFile := func(path string) *os.File {
		file, err := os.OpenFile(path,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic("无法打开日志文件 " + path + ": " + err.Error())
		}
		return file
	}

	// 创建 logger 实例
	accessLogger = log.New(openFile(accessLogPath), "", log.LstdFlags|log.Lshortfile) // 无前缀，纯 JSON
	startupLogger = log.New(openFile(startupLogPath), "", log.LstdFlags|log.Lshortfile)
	serviceLogger = log.New(openFile(serviceLogPath), "", log.LstdFlags|log.Lshortfile)

	// 如果启用控制台，添加多路输出
	if logCfg.EnableConsole {
		multiWriter := io.MultiWriter(
			os.Stdout,
			openFile(startupLogPath),
		)
		startupLogger.SetOutput(multiWriter)

		multiWriterService := io.MultiWriter(
			os.Stdout,
			openFile(serviceLogPath),
		)
		serviceLogger.SetOutput(multiWriterService)
	}

	// 配置日志级别
	minL, err := parseLogLevel(logCfg.MinLevel)
	if err != nil {
		minLevel = LogLevelWarn
	} else {
		minLevel = minL
	}
	LogStartup("日志系统初始化完成, level=%v", logCfg.MinLevel)
}

// Close 关闭所有日志文件
func Close() {
	closeLogger := func(l *log.Logger) {
		if l != nil {
			if w, ok := l.Writer().(*os.File); ok {
				w.Close()
			}
		}
	}

	closeLogger(accessLogger)
	closeLogger(startupLogger)
	closeLogger(serviceLogger)
}

// LogAccess 写入访问日志
func LogAccess(logEntry *AccessLog) {
	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		serviceLogger.Printf("序列化访问日志失败: %v", err)
		return
	}
	accessLogger.Println(string(jsonData)) // 自动换行
}

// LogStartup 写入启动日志
func LogStartup(format string, args ...any) {
	startupLogger.Printf(format, args...)
}

// LogDebug 写入Debug级别服务日志（根据 min_level 过滤）
func LogDebug(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelDebug < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[DEBUG] %s", msg)
}

// LogInfo 写入Info级别服务日志（根据 min_level 过滤）
func LogInfo(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelInfo < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[INFO] %s", msg)
}

// LogNotice 写入Notice级别服务日志（根据 min_level 过滤）
func LogNotice(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelNotice < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[NOTICE] %s", msg)
}

// LogWarn 写入Warn级别服务日志（根据 min_level 过滤）
func LogWarn(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelWarn < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[WARN] %s", msg)
}

// LogError 写入Error级别服务日志（根据 min_level 过滤）
func LogError(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelError < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[ERROR] %s", msg)
}
