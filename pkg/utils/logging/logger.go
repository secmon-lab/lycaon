package logging

import (
	"io"
	"log/slog"
	"os"

	"github.com/m-mizutani/clog"
	"golang.org/x/term"
)

// Format represents the log output format
type Format int

const (
	FormatAuto Format = iota
	FormatConsole
	FormatJSON
)

// NewLogger creates a new slog.Logger with automatic format detection
// If output is a terminal, use clog for colored console output
// Otherwise, use JSON format for structured logging
func NewLogger(level slog.Level, w io.Writer) *slog.Logger {
	return NewLoggerWithFormat(level, w, FormatAuto)
}

// NewLoggerWithFormat creates a new slog.Logger with specified format
func NewLoggerWithFormat(level slog.Level, w io.Writer, format Format) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}

	var handler slog.Handler

	switch format {
	case FormatConsole:
		// Force console output with colors
		handler = clog.New(
			clog.WithWriter(w),
			clog.WithLevel(level),
			clog.WithTimeFmt("15:04:05"),
			clog.WithSource(false),
			clog.WithAttrHook(clog.GoerrHook),
		)
	case FormatJSON:
		// Force JSON output
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})

	case FormatAuto:
		// Auto-detect based on terminal
		isTerminal := false
		if f, ok := w.(*os.File); ok {
			isTerminal = term.IsTerminal(int(f.Fd()))
		}

		if isTerminal {
			// Console output with colors
			handler = clog.New(
				clog.WithWriter(w),
				clog.WithLevel(level),
				clog.WithTimeFmt("15:04:05"),
				clog.WithSource(false),
				clog.WithAttrHook(clog.GoerrHook),
			)
		} else {
			// JSON output for non-terminal (logs, CI/CD, etc.)
			handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
				Level: level,
			})
		}
	}

	return slog.New(handler)
}

// ParseLogLevel parses a string log level to slog.Level
func ParseLogLevel(level string) slog.Level {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO", "":
		return slog.LevelInfo
	case "warn", "warning", "WARN", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
