package cli

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	controller "github.com/secmon-lab/lycaon/pkg/controller/http"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/urfave/cli/v3"
)

func cmdServe() *cli.Command {
	var (
		serverCfg    config.Server
		slackCfg     config.Slack
		firestoreCfg config.Firestore
		geminiCfg    config.Gemini
	)

	flags := joinFlags(
		serverCfg.Flags(),
		slackCfg.Flags(),
		firestoreCfg.Flags(),
		geminiCfg.Flags(),
	)

	return &cli.Command{
		Name:  "serve",
		Usage: "Start HTTP server",
		Flags: flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			// Get logger from root command metadata
			logger := ctxlog.From(ctx)

			logger.Info("Starting lycaon server",
				slog.String("addr", serverCfg.Addr),
				slog.Any("slack", slackCfg),
				slog.Any("firestore", firestoreCfg),
				slog.Any("gemini", geminiCfg),
			)

			// Create repository using config
			repo, err := firestoreCfg.Configure(ctx)
			if err != nil {
				return err
			}
			defer repo.Close()

			// Create gollem LLM client using Gemini configuration
			gollemClient := geminiCfg.ConfigureOptional(ctx, logger)
			if gollemClient == nil {
				return goerr.New("LLM client configuration is required. Please configure Gemini settings")
			}
			if closer, ok := gollemClient.(interface{ Close() error }); ok && closer != nil {
				defer closer.Close()
			}

			// Get Slack token from config
			slackToken := ""
			if slackCfg.OAuthToken != "" {
				slackToken = slackCfg.OAuthToken
			}

			// Create Slack client
			var slackClient interfaces.SlackClient
			if slackToken != "" {
				slackClient = slackSvc.New(slackToken)
			} else {
				return goerr.New("Slack client configuration is required. Please provide LYCAON_SLACK_OAUTH_TOKEN")
			}

			// Debug log Slack configuration
			logger.Debug("Slack configuration loaded",
				"has_signing_secret", slackCfg.SigningSecret != "",
				"signing_secret_length", len(slackCfg.SigningSecret),
				"has_oauth_token", slackCfg.OAuthToken != "",
			)

			// Create use cases
			authUC := usecase.NewAuth(ctx, repo, &slackCfg)
			messageUC, err := usecase.NewSlackMessage(ctx, repo, gollemClient, slackClient)
			if err != nil {
				return goerr.Wrap(err, "failed to create message use case")
			}
			incidentUC := usecase.NewIncident(repo, slackClient)

			// Create HTTP server
			server, err := controller.NewServer(
				ctx,
				serverCfg.Addr,
				&slackCfg,
				repo,
				authUC,
				messageUC,
				incidentUC,
				false, // DevMode removed
				serverCfg.FrontendURL,
			)
			if err != nil {
				return goerr.Wrap(err, "failed to create HTTP server")
			}

			// Start server in goroutine
			go func() {
				logger.Info("HTTP server starting", slog.String("addr", serverCfg.Addr))
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("HTTP server error", slog.Any("error", err))
				}
			}()

			// Wait for interrupt signal
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			select {
			case <-ctx.Done():
				logger.Info("Context cancelled, shutting down...")
			case sig := <-sigChan:
				logger.Info("Signal received, shutting down...", slog.Any("signal", sig))
			}

			// Graceful shutdown
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				return goerr.Wrap(err, "failed to shutdown server gracefully")
			}

			logger.Info("Server shutdown complete")
			return nil
		},
	}
}
