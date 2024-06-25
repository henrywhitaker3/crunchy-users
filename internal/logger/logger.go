package logger

import (
	"context"
	"os"

	"github.com/henrywhitaker3/ctxgen"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logger = "logger"
)

var (
	l *zap.SugaredLogger
)

func Wrap(ctx context.Context) context.Context {
	return ctxgen.WithValue(ctx, logger, newLogger())
}

func Logger(ctx context.Context) *zap.SugaredLogger {
	logger, ok := ctxgen.ValueOk[*zap.SugaredLogger](ctx, logger)
	if !ok {
		return newLogger()
	}
	return logger
}

func newLogger() *zap.SugaredLogger {
	if l == nil {
		conf := zap.NewProductionConfig()
		conf.OutputPaths = []string{"stdout"}
		var level zapcore.Level
		switch os.Getenv("LOG_LEVEL") {
		case "debug":
			level = zap.DebugLevel
		case "error":
			level = zap.ErrorLevel
		case "info":
			fallthrough
		default:
			level = zap.InfoLevel
		}
		conf.Level = zap.NewAtomicLevelAt(level)
		logger, _ := conf.Build()
		l = logger.Sugar()
	}
	return l
}
