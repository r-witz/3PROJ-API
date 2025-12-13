package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(logLevel string) {
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		level = zap.InfoLevel
	}

	cfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       level == zap.DebugLevel,
		DisableStacktrace: level < zap.ErrorLevel,
		Encoding:          "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	Logger, err = cfg.Build()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		_ = Logger.Sync()
	}()
}
