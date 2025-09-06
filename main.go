package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args); err != nil {
		slog.Error("CLI execution failed", slog.Any("error", err))

		// Print error details with goerr
		if goErr := goerr.Unwrap(err); goErr != nil {
			for k, v := range goErr.Values() {
				slog.Error("Error detail", slog.String("key", k), slog.Any("value", v))
			}
		}
		os.Exit(1)
	}
}
