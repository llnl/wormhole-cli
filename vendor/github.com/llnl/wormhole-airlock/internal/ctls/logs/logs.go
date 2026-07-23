package logs

import (
	"io"
	"log/slog"
	"log/syslog"
	"os"
	"strings"

	"github.com/llnl/wormhole-airlock/internal/args"
)

const (
	defaultLevel = slog.LevelInfo
	loggerTag    = "airlock"
)

type Logger interface {
	Debug(msg string, attr ...slog.Attr)
	Info(msg string, attr ...slog.Attr)
	Warn(msg string, attr ...slog.Attr)
	Error(msg string, attr ...slog.Attr)
	StringArg(k, v string) slog.Attr
	IntArg(k string, v int) slog.Attr
	With(args ...any) Logger
}

func Initialize(loggingArgs args.Logging) (Logger, error) {
	hOpts := &slog.HandlerOptions{Level: level(loggingArgs.Level)}
	if loggingArgs.Disable {
		return logger{
			s: slog.New(slog.DiscardHandler),
		}, nil
	}

	w, err := writer(loggingArgs)
	if err != nil {
		return nil, err
	}

	fields := fields()

	return logger{
		s: slog.New(slog.NewJSONHandler(w, hOpts).WithAttrs(fields)),
	}, nil
}

// InitializeDiscard creates a logger that discards all log messages.
func InitializeDiscard() Logger {
	return logger{
		s: slog.New(slog.DiscardHandler),
	}
}

//

func writer(loggingArgs args.Logging) (io.Writer, error) {
	switch strings.ToLower(loggingArgs.Location) {
	case "syslog", "":
		if loggingArgs.Address != "" {
			return syslog.Dial(
				loggingArgs.Network,
				loggingArgs.Address,
				syslog.LOG_DEBUG,
				loggerTag,
			)
		}

		return syslog.New(syslog.LOG_DEBUG, loggerTag)
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	}

	return newLogFile(loggingArgs.Location)
}

func level(lvl string) slog.Level {
	switch strings.ToLower(lvl) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return defaultLevel
	}
}

func fields() []slog.Attr {
	f := []slog.Attr{
		slog.Int("processID", os.Getpid()),
	}

	host, err := os.Hostname()
	if err == nil {
		// Failure to identify hostname should not result in a job failure.
		f = append(f, slog.String("hostname", host))
	}

	return f
}
