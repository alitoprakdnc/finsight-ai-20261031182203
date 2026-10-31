package util

import (
	"context"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

var (
	logger     *zap.Logger
	initErr    error
	initOnce   sync.Once
	defaultSvc = "finSight-api"
)

// InitLogger configures a global zap logger.
// It should be called once at application start (e.g., in cmd/api/main.go).
// The logger is JSON‑encoded, writes to stdout/stderr and includes a
// configurable log level via the LOG_LEVEL environment variable.
func InitLogger() error {
	initOnce.Do(func() {
		levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
		if levelStr == "" {
			levelStr = "info"
		}
		var lvl zapcore.Level
		if err := lvl.UnmarshalText([]byte(levelStr)); err != nil {
			initErr = err
			return
		}

		encCfg := zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		cfg := zap.Config{
			Level:            zap.NewAtomicLevelAt(lvl),
			Development:      false,
			Encoding:         "json",
			EncoderConfig:    encCfg,
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
			DisableStacktrace: true,
		}

		var l *zap.Logger
		l, initErr = cfg.Build(zap.AddCallerSkip(1))
		if initErr != nil {
			return
		}
		// Attach a static field identifying the service.
		logger = l.With(zap.String("service", defaultSvc))
	})
	return initErr
}

// GetLogger returns the globally configured zap logger.
// If InitLogger has not been called or failed, a no‑op logger is returned.
func GetLogger() *zap.Logger {
	if logger == nil {
		// fallback to a no‑op logger to avoid panics
		return zap.NewNop()
	}
	return logger
}

// WithCorrelationID returns a logger that includes the supplied correlation ID.
// This is useful for tracing a request across services.
func WithCorrelationID(l *zap.Logger, correlationID string) *zap.Logger {
	if correlationID == "" {
		return l
	}
	return l.With(zap.String("correlation_id", correlationID))
}

// FromContext extracts a logger from the provided context.
// If a logger is not present, the global logger is returned.
// The extracted logger will already contain any correlation ID that was
// previously injected via ContextWithLogger.
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return GetLogger()
	}
	if v := ctx.Value(ctxKey{}); v != nil {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}
	return GetLogger()
}

// ContextWithLogger returns a new context containing the supplied logger.
// This is typically used in HTTP middleware to attach a request‑scoped logger.
func ContextWithLogger(parent context.Context, l *zap.Logger) context.Context {
	return context.WithValue(parent, ctxKey{}, l)
}

// ContextWithCorrelationID extracts (or generates) a correlation ID from the
// incoming context, attaches it to a logger and returns the enriched logger
// together with a new context that carries the logger forward.
// If the incoming context already holds a logger, it will be reused.
func ContextWithCorrelationID(parent context.Context, correlationID string) (context.Context, *zap.Logger) {
	base := FromContext(parent)
	loggerWithID := WithCorrelationID(base, correlationID)
	return ContextWithLogger(parent, loggerWithID), loggerWithID
}