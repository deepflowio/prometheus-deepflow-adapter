package log

import (
	"bytes"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *log

type log struct {
	logger *zap.Logger
	pool   *sync.Pool
}

func NewLogger(l string) *log {
	var logger *zap.Logger
	var level = getLevel(l)
	if level == zap.DebugLevel {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync()

	return &log{
		logger: logger,
		pool: &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
	}
}

func getLevel(level string) zapcore.Level {
	switch level {
	case "info":
		return zap.InfoLevel
	case "debug":
		return zap.DebugLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

func (l *log) getLog(keyvals ...any) string {
	var log string
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "")
	}
	buf := l.pool.Get().(*bytes.Buffer)
	for i := 0; i < len(keyvals); i += 2 {
		fmt.Fprintf(buf, "%s: %v", keyvals[i], keyvals[i+1])
	}
	log = buf.String()
	buf.Reset()
	l.pool.Put(buf)

	return log
}

func (l *log) Info(keyvals ...any) {
	if l == nil {
		return
	}
	l.logger.Info(l.getLog(keyvals...))
}

func (l *log) Debug(keyvals ...any) {
	if l == nil {
		return
	}
	l.logger.Debug(l.getLog(keyvals...))
}

func (l *log) Error(keyvals ...any) {
	if l == nil {
		return
	}
	l.logger.Error(l.getLog(keyvals...))
}
