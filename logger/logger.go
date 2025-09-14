package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 日志配置结构
type Config struct {
	AccessLogPath  string `yaml:"access_log_path"`
	StartupLogPath string `yaml:"startup_log_path"`
	ServiceLogPath string `yaml:"service_log_path"`
	EnableConsole  bool   `yaml:"enable_console"`
	MinLevel       string `yaml:"min_level"` // 字符串形式，后面转成 LogLevel
}

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
	IsHTTPS      bool          `json:"is_https"`
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

// LoadConfig 从 config.yaml 加载配置
func LoadConfig(configFile string) error {
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
	minLevel, err = ParseLogLevel(config.MinLevel)
	if err != nil {
		return fmt.Errorf("无效的 min_level: %v", err)
	}

	consoleEnabled = config.EnableConsole

	return nil
}

func ParseLogLevel(s string) (LogLevel, error) {
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

func Init(configFile string) {
	// 1. 加载配置
	if err := LoadConfig(configFile); err != nil {
		panic("日志配置加载失败: " + err.Error())
	}

	// 2. 创建 logs 目录
	if err := os.MkdirAll(filepath.Dir(config.AccessLogPath), 0755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(config.StartupLogPath), 0755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(config.ServiceLogPath), 0755); err != nil {
		panic("无法创建日志目录: " + err.Error())
	}

	// 3. 设置时间戳前缀
	prefix := func() string {
		return "[" + time.Now().Format("2006-01-02 15:04:05") + "] "
	}

	// 4. 打开文件
	openFile := func(path string) *os.File {
		file, err := os.OpenFile(path,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic("无法打开日志文件 " + path + ": " + err.Error())
		}
		return file
	}

	// 5. 创建 logger 实例
	accessLogger = log.New(openFile(config.AccessLogPath), "", 0) // 无前缀，纯 JSON
	startupLogger = log.New(openFile(config.StartupLogPath), prefix(), log.LstdFlags)
	serviceLogger = log.New(openFile(config.ServiceLogPath), prefix(), log.LstdFlags)

	// 6. 如果启用控制台，添加多路输出
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

	// 7. 启动日志
	startupLogger.Println("日志系统已加载配置，min_level=" + config.MinLevel)
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

	startupLogger.Println("日志系统已关闭")
}

// LogAccess 写入访问日志（JSON 格式）
func LogAccess(logEntry *AccessLog) {
	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		serviceLogger.Printf("序列化访问日志失败: %v", err)
		return
	}
	accessLogger.Println(string(jsonData)) // 自动换行
}

// LogStartup 写入启动日志（带时间戳）
func LogStartup(format string, args ...any) {
	startupLogger.Printf(format, args...)
}

// LogService 写入服务日志（根据 min_level 过滤）
func LogDebug(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelDebug < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[DEBUG] %s", msg)
}

// LogService 写入服务日志（根据 min_level 过滤）
func LogInfo(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelInfo < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[INFO] %s", msg)
}

// LogService 写入服务日志（根据 min_level 过滤）
func LogNotice(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelNotice < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[NOTICE] %s", msg)
}

// LogService 写入服务日志（根据 min_level 过滤）
func LogWarn(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelWarn < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[WARN] %s", msg)
}

// LogService 写入服务日志（根据 min_level 过滤）
func LogError(format string, args ...any) {
	// 如果级别低于 min_level，不写入
	if LogLevelError < minLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	serviceLogger.Printf("[ERROR] %s", msg)
}
