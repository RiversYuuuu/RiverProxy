package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	accessLogger  *log.Logger
	startupLogger *log.Logger
	serviceLogger *log.Logger

	// 全局配置
	config Config

	// 最小日志级别（用于 service.log 过滤）
	minLevel LogLevel

	// 控制台输出是否启用
	consoleEnabled bool
)

// loadConfig 从 config.yaml 加载配置
func loadConfig(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("无法读取日志配置文件: %w", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("解析日志配置文件失败: %w", err)
	}

	// 默认值
	if config.AccessLogPath == "" {
		config.AccessLogPath = "/var/log/riverproxy/access.log"
	}
	if config.StartupLogPath == "" {
		config.StartupLogPath = "/var/log/riverproxy/startup.log"
	}
	if config.ServiceLogPath == "" {
		config.ServiceLogPath = "/var/log/riverproxy/service.log"
	}
	if config.MinLevel == "" {
		config.MinLevel = "INFO"
	}

	// 解析最小日志级别
	minLevel, err = parseLogLevel(config.MinLevel)
	if err != nil {
		return fmt.Errorf("无效的 min_level: %v", err)
	}

	consoleEnabled = config.EnableConsole

	return nil
}

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
func Init(configFile string) {
	// 加载配置
	if err := loadConfig(configFile); err != nil {
		panic("日志配置加载失败: " + err.Error())
	}

	// 创建 logs 目录
	if err := os.MkdirAll(filepath.Dir(config.AccessLogPath), 0755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(config.StartupLogPath), 0755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(config.ServiceLogPath), 0755); err != nil {
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
	accessLogger = log.New(openFile(config.AccessLogPath), "", log.LstdFlags|log.Lshortfile) // 无前缀，纯 JSON
	startupLogger = log.New(openFile(config.StartupLogPath), "", log.LstdFlags|log.Lshortfile)
	serviceLogger = log.New(openFile(config.ServiceLogPath), "", log.LstdFlags|log.Lshortfile)

	// 如果启用控制台，添加多路输出
	if consoleEnabled {
		multiWriter := io.MultiWriter(
			os.Stdout,
			openFile(config.StartupLogPath),
		)
		startupLogger.SetOutput(multiWriter)

		multiWriterService := io.MultiWriter(
			os.Stdout,
			openFile(config.ServiceLogPath),
		)
		serviceLogger.SetOutput(multiWriterService)
	}

	startupLogger.Println("日志系统已加载配置, min_level=" + config.MinLevel)
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
