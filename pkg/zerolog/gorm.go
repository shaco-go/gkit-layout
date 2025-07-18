package gkit_zerolog

import (
	"context"
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"time"
)

func ZeroToGormLevel(level zerolog.Level) gormLogger.LogLevel {
	switch level {
	case zerolog.TraceLevel:
		return gormLogger.Info
	case zerolog.DebugLevel:
		return gormLogger.Info
	case zerolog.InfoLevel:
		return gormLogger.Info
	case zerolog.WarnLevel:
		return gormLogger.Warn
	case zerolog.ErrorLevel:
		return gormLogger.Error
	case zerolog.FatalLevel:
		return gormLogger.Error
	case zerolog.PanicLevel:
		return gormLogger.Error
	case zerolog.Disabled:
		return gormLogger.Silent
	case zerolog.NoLevel:
		return gormLogger.Silent
	}
	return gormLogger.Silent
}

// NewGormLogger initialize logger
func NewGormLogger(z zerolog.Logger, config gormLogger.Config) gormLogger.Interface {
	var (
		infoStr      = "%s"
		warnStr      = "%s"
		errStr       = "%s"
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	)

	if config.Colorful {
		infoStr = gormLogger.Green + "%s\n" + gormLogger.Reset
		warnStr = gormLogger.BlueBold + "%s\n" + gormLogger.Reset
		errStr = gormLogger.Magenta + "%s\n" + gormLogger.Reset
		traceStr = gormLogger.Green + "%s\n" + gormLogger.Reset + gormLogger.Yellow + "[%.3fms] " + gormLogger.BlueBold + "[rows:%v]" + gormLogger.Reset + " %s"
		traceWarnStr = gormLogger.Green + "%s " + gormLogger.Yellow + "%s\n" + gormLogger.Reset + gormLogger.RedBold + "[%.3fms] " + gormLogger.Yellow + "[rows:%v]" + gormLogger.Magenta + " %s" + gormLogger.Reset
		traceErrStr = gormLogger.RedBold + "%s " + gormLogger.MagentaBold + "%s\n" + gormLogger.Reset + gormLogger.Yellow + "[%.3fms] " + gormLogger.BlueBold + "[rows:%v]" + gormLogger.Reset + " %s"
	}

	return &customGormLogger{
		z:            z,
		Config:       config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}

type customGormLogger struct {
	gormLogger.Config
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
	z                                   zerolog.Logger
}

// LogMode log mode
func (l *customGormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info
func (l *customGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.z.Info().Ctx(ctx).Msgf(msg, data...)
}

// Warn print warn messages
func (l *customGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.z.Warn().Ctx(ctx).Msgf(msg, data...)
}

// Error print error messages
func (l *customGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.z.Error().Ctx(ctx).Msgf(msg, data...)
}

// Trace print sql message
//
//nolint:cyclop
func (l *customGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormLogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= gormLogger.Error && (!errors.Is(err, gormLogger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.z.Error().Ctx(ctx).Msgf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.z.Error().Ctx(ctx).Msgf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormLogger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.z.Warn().Ctx(ctx).Msgf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.z.Warn().Ctx(ctx).Msgf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.LogLevel == gormLogger.Info:
		sql, rows := fc()
		if rows == -1 {
			l.z.Info().Ctx(ctx).Msgf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.z.Info().Ctx(ctx).Msgf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}
