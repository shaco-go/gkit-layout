package gkit_zerolog

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
	"time"
)

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

// New 预配置zero log
// @param isDev 是否是开发环境
// @param level 默认开发trace,生产error
// 开发环境(本地):用友好的输出方式打印在终端
// 非开发环境:json格式输出在文件中
// 返回为原始配置不含Timestamp,Stack,Caller()
func New(isDev bool, arg ...string) zerolog.Logger {
	level := "warn"
	if isDev {
		level = "trace"
	}
	if len(arg) > 0 {
		level = arg[0]
	}
	parseLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		panic(fmt.Errorf("解析日志级别错误:%w", err))
	}
	zerolog.SetGlobalLevel(parseLevel)
	var output io.Writer
	if isDev {
		output = ConsoleDevOutput()
	} else {
		output = JsonFileProOutput()
	}
	return zerolog.New(output)
}

// ConsoleDevOutput 开发模式(控制台)
func ConsoleDevOutput() io.Writer {
	zerolog.ErrorStackMarshaler = marshalStack
	output := zerolog.ConsoleWriter{Out: os.Stderr}
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

// JsonFileProOutput 生产模式(文件)
func JsonFileProOutput() io.Writer {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339
	// 确保日志目录存在
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(err)
	}

	// 配置日志分割
	lumberLogger := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "app.log"),
		MaxSize:    100, // 最大文件大小，单位MB
		MaxBackups: 30,  // 保留的旧文件最大数量
		MaxAge:     30,  // 保留的最大天数
		Compress:   true,
	}
	return lumberLogger
}
