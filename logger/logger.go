// Package logger provides a simple logging interface
package logger

import "go.uber.org/zap"

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
