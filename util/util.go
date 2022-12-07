package util

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(loggerName string) *zap.SugaredLogger {
	logLevel := Getenv("LOG_LEVEL", "INFO")
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}
	loggingConfig := zap.NewProductionConfig()
	loggingConfig.Level.SetLevel(level)

	l, _ := loggingConfig.Build(zap.WithCaller(false), zap.AddStacktrace(zapcore.PanicLevel))
	return l.Sugar().With(zap.String("logger", loggerName))
}

func Getenv(key string, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v
}

func PrintLatency(logger *zap.SugaredLogger, item string, start time.Time) {
	logger.With(zap.Int64("latencyMs", time.Since(start).Milliseconds()), zap.String("call", item)).Debug("Latency")
}

func CheckContextError(err error) bool {
	return err != nil && err != context.Canceled && err != context.DeadlineExceeded
}
