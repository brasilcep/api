package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log   *zap.Logger
	sugar *zap.SugaredLogger
)

func Init(level string, enableFile bool, filePath string) error {
	lvl := parseLevel(level)

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.TimeKey = "ts"

	consoleEnc := zapcore.NewConsoleEncoder(encCfg)
	jsonEnc := zapcore.NewJSONEncoder(encCfg)

	cores := []zapcore.Core{
		zapcore.NewCore(consoleEnc, zapcore.Lock(os.Stdout), lvl),
	}

	if enableFile {
		if filePath == "" {
			filePath = "app.log"
		}
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		fileSync := zapcore.AddSync(f)
		cores = append(cores, zapcore.NewCore(jsonEnc, fileSync, lvl))
	}

	core := zapcore.NewTee(cores...)
	log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	sugar = log.Sugar()
	return nil
}

// Logger returns the underlying *zap.Logger. May be nil if Init wasn't called.
func Logger() *zap.Logger {
	return log
}

// Sugar returns the *zap.SugaredLogger. May be nil if Init wasn't called.
func Sugar() *zap.SugaredLogger {
	return sugar
}

// Sync flushes any buffered log entries.
func Sync() error {
	if log == nil {
		return nil
	}
	return log.Sync()
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
