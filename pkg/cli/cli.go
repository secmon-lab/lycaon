package cli

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	"github.com/urfave/cli/v3"
)

// Run runs the CLI application
func Run(ctx context.Context, args []string) error {
	var loggerCfg config.Logger

	app := &cli.Command{
		Name:    "lycaon",
		Usage:   "Slack-based incident management service",
		Version: "0.1.0",
		Flags:   loggerCfg.Flags(),
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			// Configure logger
			logger, err := loggerCfg.Configure()
			if err != nil {
				return nil, err
			}

			slog.SetDefault(logger)
			ctx = ctxlog.With(ctx, logger)
			return ctx, nil
		},
		Commands: []*cli.Command{
			cmdServe(),
		},
	}

	if err := app.Run(ctx, args); err != nil {
		return goerr.Wrap(err, "CLI execution failed")
	}

	return nil
}
