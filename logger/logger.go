package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
	sugar *zap.SugaredLogger
}

func NewLogger(level string) *Logger {
	lvl := parseLevel(level)

	l := &Logger{}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.TimeKey = "ts"

	consoleEnc := zapcore.NewConsoleEncoder(encCfg)

	cores := []zapcore.Core{
		zapcore.NewCore(consoleEnc, zapcore.Lock(os.Stdout), lvl),
	}

	core := zapcore.NewTee(cores...)
	l.Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	l.sugar = l.Logger.Sugar()

	return l
}

func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.sugar
}

func (l *Logger) Sync() error {
	if l.Logger == nil {
		return nil
	}
	return l.Logger.Sync()
}

func parseLevel(l string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(l)) {
	case "debug":
		return zapcore.DebugLevel
	case "info", "":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
