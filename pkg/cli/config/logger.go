package config

import (
	"log/slog"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

// Logger holds logger configuration
type Logger struct {
	Level  string
	Format string
}

// Flags returns CLI flags for Logger configuration
func (l *Logger) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "Log level (debug, info, warn, error)",
			Category:    "Logging",
			Value:       "info",
			Sources:     cli.EnvVars("LYCAON_LOG_LEVEL"),
			Destination: &l.Level,
		},
		&cli.StringFlag{
			Name:        "log-format",
			Usage:       "Log format (console, json, auto)",
			Category:    "Logging",
			Value:       "auto",
			Sources:     cli.EnvVars("LYCAON_LOG_FORMAT"),
			Destination: &l.Format,
		},
	}
}

// Configure sets up the logger based on configuration
func (l *Logger) Configure() (*slog.Logger, error) {
	level := logging.ParseLogLevel(l.Level)

	// Parse format option
	format := logging.FormatAuto
	switch l.Format {
	case "console":
		format = logging.FormatConsole
	case "json":
		format = logging.FormatJSON
	case "auto", "":
		format = logging.FormatAuto
	default:
		return nil, goerr.New("invalid log format", goerr.V("format", l.Format))
	}

	logger := logging.NewLoggerWithFormat(level, os.Stdout, format)
	return logger, nil
}

// LogValue returns structured log value
func (l Logger) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("level", l.Level),
		slog.String("format", l.Format),
	)
}

// Validate validates the logger configuration
func (l *Logger) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[l.Level] {
		return goerr.New("invalid log level", goerr.V("level", l.Level))
	}

	validFormats := map[string]bool{
		"console": true,
		"json":    true,
		"auto":    true,
		"":        true, // empty means auto
	}
	if !validFormats[l.Format] {
		return goerr.New("invalid log format", goerr.V("format", l.Format))
	}

	return nil
}
