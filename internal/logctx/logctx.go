package logctx

import (
	"context"
	"errors"
	"log/slog"
)

var ErrNoLogger = errors.New("logger not in context")

type ContextLogger struct {
	Logger   *slog.Logger
	levelVar *slog.LevelVar
}

type logctxKey struct{}

func New(logger *slog.Logger, levelVar *slog.LevelVar) *ContextLogger {
	return &ContextLogger{
		Logger:   logger,
		levelVar: levelVar,
	}
}

// WithLogger wraps context with a logger.
func WithLogger(ctx context.Context, cl *ContextLogger) context.Context {
	if cl == nil {
		return ctx
	}

	return context.WithValue(ctx, logctxKey{}, cl)
}

func GetLogger(ctx context.Context) *ContextLogger {
	if logger, ok := ctx.Value(logctxKey{}).(*ContextLogger); ok && logger != nil {
		return logger
	}

	return nil
}

// Logger returns the logger stored in a context, if any.
func Logger(ctx context.Context) *slog.Logger {
	if logger := GetLogger(ctx); logger != nil {
		return logger.Logger
	}

	return nil
}

func (cl *ContextLogger) SetLogLevel(level slog.Level) error {
	if cl != nil {
		if cl.levelVar != nil {
			cl.levelVar.Set(level)

			return nil
		}
	}

	return ErrNoLogger
}

func (cl *ContextLogger) GetLogLevel() (*slog.Level, error) {
	if cl != nil {
		if cl.levelVar != nil {
			level := cl.levelVar.Level()

			return &level, nil
		}
	}

	return nil, ErrNoLogger
}
