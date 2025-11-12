package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

// Config 日志配置
type Config struct {
	Level      string `yaml:"level"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`    // MB
	MaxBackups int    `yaml:"max_backups"` // 文件数量
	MaxAge     int    `yaml:"max_age"`     // 天数
	Format     string `yaml:"format"`      // json 或 text
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	With(fields ...zap.Field) *zap.Logger
}

var globalLogger *zap.Logger

// Init 初始化日志系统
func Init(config Config) error {
	// 创建日志目录
	if err := os.MkdirAll(filepath.Dir(config.FilePath), 0755); err != nil {
		return err
	}

	// 设置日志级别
	level := zap.InfoLevel
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		level = zap.InfoLevel
	}

	// 创建编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 设置编码器
	var encoder zapcore.Encoder
	if config.Format == "text" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建文件写入器
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.FilePath,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   true,
	})

	// 创建控制台写入器
	consoleWriter := zapcore.AddSync(os.Stdout)

	// 创建核心
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, fileWriter, level),
		zapcore.NewCore(encoder, consoleWriter, level),
	)

	// 创建日志器
	globalLogger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))

	return nil
}

// NewLogger 创建新的日志器
func NewLogger() (*zap.Logger, error) {
	// 使用默认配置创建日志器
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// GetLogger 获取全局日志器
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		// 返回一个默认的日志器
		logger, _ := zap.NewProduction()
		return logger
	}
	return globalLogger
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// With 添加字段
func With(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}

// Sync 刷新日志缓冲区
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}