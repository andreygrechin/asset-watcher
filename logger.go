package main

import (
	"log/slog"
	"os"
	"time"
)

func setupLogging(cfg *Config) *slog.Logger {
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}

	// Use json as our base logging format.
	jsonHandler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{ReplaceAttr: convertSlogToCloudLogging, Level: logLevel},
	)
	// Add span context attributes when Context is passed to logging calls.
	instrumentedHandler := handlerWithSpanContext(jsonHandler)

	logger := slog.New(instrumentedHandler)

	return logger
}

// spanContextLogHandler is a slog.Handler which adds attributes from the
// span context.
type spanContextLogHandler struct {
	slog.Handler
}

func convertSlogToCloudLogging(_ []string, a slog.Attr) slog.Attr {
	// Rename attribute keys to match Cloud Logging structured log format
	switch a.Key {
	case slog.LevelKey:
		a.Key = "severity"
		// Map slog.Level string values to Cloud Logging LogSeverity
		// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#LogSeverity
		if level, ok := a.Value.Any().(slog.Level); ok && level == slog.LevelWarn {
			a.Value = slog.StringValue("WARNING")
		}
	case slog.TimeKey:
		a.Key = "timestamp"
		if t, ok := a.Value.Any().(time.Time); ok {
			a.Value = slog.TimeValue(t.UTC())
		}
	case slog.MessageKey:
		a.Key = "message"
	}

	return a
}

// handlerWithSpanContext adds attributes from the span context.
func handlerWithSpanContext(handler slog.Handler) *spanContextLogHandler {
	return &spanContextLogHandler{Handler: handler}
}
