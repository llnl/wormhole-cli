package logs

import (
	"context"
	"log/slog"
)

type logger struct {
	s *slog.Logger
}

func (l logger) Debug(msg string, attr ...slog.Attr) {
	l.s.LogAttrs(context.TODO(), slog.LevelDebug, msg, attr...)
}

func (l logger) Info(msg string, attr ...slog.Attr) {
	l.s.LogAttrs(context.TODO(), slog.LevelInfo, msg, attr...)
}

func (l logger) Warn(msg string, attr ...slog.Attr) {
	l.s.LogAttrs(context.TODO(), slog.LevelWarn, msg, attr...)
}

func (l logger) Error(msg string, attr ...slog.Attr) {
	l.s.LogAttrs(context.TODO(), slog.LevelError, msg, attr...)
}

func (l logger) StringArg(k, v string) slog.Attr {
	return slog.String(k, v)
}

func (l logger) IntArg(k string, v int) slog.Attr {
	return slog.Int(k, v)
}

func (l logger) With(args ...any) Logger {
	return logger{
		s: l.s.With(args...),
	}
}
