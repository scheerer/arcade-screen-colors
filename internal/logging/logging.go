package logging

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfg = zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		// Sampling: &zap.SamplingConfig{
		// 	Initial:    100,
		// 	Thereafter: 100,
		// },
		Encoding: "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stdout"},
	}
	leveler = &levelSetter{
		levelers: make(map[string]zap.AtomicLevel),
	}
)

type Leveler interface {
	SetLevel(name string, level zapcore.Level)
	GetLevel(name string) zapcore.Level
}

type levelSetter struct {
	levelers map[string]zap.AtomicLevel
	mu       sync.RWMutex
}

var _ Leveler = (*levelSetter)(nil)

func GetLeveler() Leveler {
	return leveler
}

func (lw *levelSetter) SetLevel(name string, level zapcore.Level) {
	_ = lw.setLevel(name, level)
}

func (lw *levelSetter) GetLevel(name string) zapcore.Level {
	lw.mu.RLock()
	defer lw.mu.RUnlock()

	if l, ok := lw.levelers[name]; ok {
		return l.Level()
	}

	return zap.InfoLevel
}

func (lw *levelSetter) setLevel(name string, level zapcore.Level) zap.AtomicLevel {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if _, ok := lw.levelers[name]; !ok {
		lw.levelers[name] = zap.NewAtomicLevelAt(level)
	}

	lw.levelers[name].SetLevel(level)

	return lw.levelers[name]
}

func New(name string) *zap.SugaredLogger {
	c := cfg
	c.Level = leveler.setLevel(name, zap.InfoLevel)
	return zap.Must(c.Build(zap.AddStacktrace(zapcore.PanicLevel))).Named(name).Sugar()
}
