package gkit_zerolog

import (
	"bytes"
	"fmt"
	gormLogger "gorm.io/gorm/logger"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Log struct {
	ZeroLog zerolog.Logger
	GormLog gormLogger.Interface
}

// ChannelType 定义日志输出类型
type ChannelType int

const (
	// ConsoleChannel 控制台输出
	ConsoleChannel ChannelType = iota
	// FileChannel 文件输出
	FileChannel
)

// String 将OutputType转换为字符串
func (o ChannelType) String() string {
	switch o {
	case ConsoleChannel:
		return "console"
	case FileChannel:
		return "file"
	default:
		return "console"
	}
}

// ParseChannelType 将字符串转换为OutputType
func ParseChannelType(outputStr string) (ChannelType, error) {
	switch strings.ToLower(outputStr) {
	case "console":
		return ConsoleChannel, nil
	case "file":
		return FileChannel, nil
	default:
		return ConsoleChannel, fmt.Errorf("未知的输出类型: '%s'，默认使用控制台输出", outputStr)
	}
}

// LogConfig 日志配置
type LogConfig struct {
	// Level 日志级别
	Level            zerolog.Level
	SqlSlowThreshold time.Duration
	// Channel 输出类型
	Channel []ChannelType
	// HumanReadable 是否使用人类可读格式
	HumanReadable bool
	// LogDir 日志目录
	LogDir string
	// LogFileName 日志文件名
	LogFileName string
	// MaxSize 单个日志文件最大大小，单位MB
	MaxSize int
	// MaxBackups 保留的旧文件最大数量
	MaxBackups int
	// MaxAge 保留的最大天数
	MaxAge int
	// Compress 是否压缩
	Compress bool
}

// DefaultLogConfig 返回默认日志配置
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:         zerolog.WarnLevel,
		Channel:       []ChannelType{ConsoleChannel},
		HumanReadable: true,
		LogDir:        "logs",
		LogFileName:   "app.log",
		MaxSize:       100,
		MaxBackups:    30,
		MaxAge:        30,
		Compress:      true,
	}
}

// NewDevLogConfig 返回开发环境日志配置
func NewDevLogConfig() *LogConfig {
	config := DefaultLogConfig()
	config.Level = zerolog.TraceLevel
	return config
}

// NewProdLogConfig 返回生产环境日志配置
func NewProdLogConfig() *LogConfig {
	config := DefaultLogConfig()
	config.Level = zerolog.ErrorLevel
	config.Channel = []ChannelType{FileChannel}
	config.HumanReadable = false
	return config
}

// MarshalStack implements pkg/errors stack trace marshaling.
func marshalStack(err error) interface{} {
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}
	var sterr stackTracer
	var ok bool
	for err != nil {
		sterr, ok = err.(stackTracer)
		if ok {
			break
		}

		u, ok := err.(interface {
			Unwrap() error
		})
		if !ok {
			return nil
		}

		err = u.Unwrap()
	}
	if sterr == nil {
		return nil
	}

	return fmt.Sprintf("\n%+v", sterr)
}

// New 根据配置创建新的zerolog.Logger
func New(config *LogConfig) zerolog.Logger {
	if config == nil {
		config = DefaultLogConfig()
	}

	zerolog.SetGlobalLevel(config.Level)

	var output []io.Writer

	for _, channel := range config.Channel {
		switch channel {
		case ConsoleChannel:
			output = append(output, createConsoleOutput(config.HumanReadable))
		case FileChannel:
			output = append(output, createFileOutput(config))
		default:
			if len(output) == 0 {
				output = append(output, createConsoleOutput(config.HumanReadable))
			}
		}
	}

	return zerolog.New(zerolog.MultiLevelWriter(output...))
}

// createConsoleOutput 创建控制台输出
func createConsoleOutput(humanReadable bool) io.Writer {
	zerolog.ErrorStackMarshaler = marshalStack

	if !humanReadable {
		return os.Stderr
	}

	output := zerolog.ConsoleWriter{Out: os.Stderr}
	output.TimeFormat = time.DateTime
	output.FieldsExclude = []string{zerolog.ErrorStackFieldName, zerolog.ErrorFieldName}
	output.FormatExtra = func(m map[string]interface{}, b *bytes.Buffer) error {
		if stack, ok := m[zerolog.ErrorStackFieldName]; ok {
			if val, ok := stack.(string); ok {
				b.WriteString(val)
			}
		} else if err, ok := m[zerolog.ErrorFieldName]; ok {
			b.WriteString(fmt.Sprintf(" \033[36;1m%s=\033[0m%s\n", zerolog.ErrorFieldName, err))
		}
		return nil
	}
	return output
}

// createFileOutput 创建文件输出
func createFileOutput(config *LogConfig) io.Writer {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339

	// 确保日志目录存在
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		panic(err)
	}

	// 配置日志分割
	lumberLogger := &lumberjack.Logger{
		Filename:   filepath.Join(config.LogDir, config.LogFileName),
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}
	return lumberLogger
}
