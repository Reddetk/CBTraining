// Package logger log all
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

type Field = zap.Field

type zapLogger struct {
	l *zap.Logger
}

func NewZapLogger(l *zap.Logger) Logger {
	return &zapLogger{l: l}
}

func (z *zapLogger) Info(msg string, fields ...Field) {
	z.l.Info(msg, fields...)
}

func (z *zapLogger) Error(msg string, fields ...Field) {
	z.l.Error(msg, fields...)
}

// New — создаёт готовый логгер с читаемым форматом
func New() Logger {
	cfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"),
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder, // только file.go:line
			EncodeDuration: zapcore.StringDurationEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	l, _ := cfg.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1), // пропускаем обёртку — показывает реальный вызывающий
		zap.AddStacktrace(zap.ErrorLevel),
	)

	return &zapLogger{l: l}
}
