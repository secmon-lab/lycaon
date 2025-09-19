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
	slackCtrl "github.com/secmon-lab/lycaon/pkg/controller/slack"
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

	// Add config file flag
	configFlag := &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "Path to configuration file (required if LYCAON_CONFIG not set)",
		Sources: cli.EnvVars("LYCAON_CONFIG"),
	}

	flags := joinFlags(
		[]cli.Flag{configFlag},
		serverCfg.Flags(),
		slackCfg.Flags(),
		firestoreCfg.Flags(),
		geminiCfg.Flags(),
	)

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "Start HTTP server",
		Flags:   flags,
		Action: func(ctx context.Context, c *cli.Command) error {
			// Get logger from root command metadata
			logger := ctxlog.From(ctx)

			// Load categories configuration
			configPath := c.String("config")
			if configPath == "" {
				return goerr.New("configuration file path is required (use --config flag or LYCAON_CONFIG environment variable)")
			}

			categories, err := config.LoadCategoriesFromFile(configPath)
			if err != nil {
				return goerr.Wrap(err, "failed to load configuration file")
			}

			logger.Info("Starting lycaon server",
				slog.String("addr", serverCfg.Addr),
				slog.String("config", configPath),
				slog.Int("categories", len(categories.Categories)),
				slog.String("channel_prefix", slackCfg.ChannelPrefix),
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
			gollemClient := geminiCfg.ConfigureOptional(ctx)
			if gollemClient == nil {
				return goerr.New("LLM client configuration is required. Please configure Gemini settings")
			}
			if closer, ok := gollemClient.(interface{ Close() error }); ok && closer != nil {
				defer closer.Close()
			}

			// Create Slack client using config
			slackClient, err := slackCfg.Configure(ctx)
			if err != nil {
				return err
			}

			// Debug log Slack configuration
			logger.Debug("Slack configuration loaded",
				"has_signing_secret", slackCfg.SigningSecret != "",
				"signing_secret_length", len(slackCfg.SigningSecret),
				"has_oauth_token", slackCfg.OAuthToken != "",
			)

			// Create use cases
			authUC := usecase.NewAuth(ctx, repo, &slackCfg)
			messageUC, err := usecase.NewSlackMessage(ctx, repo, gollemClient, slackClient, categories)
			if err != nil {
				return goerr.Wrap(err, "failed to create message use case")
			}
			inviteUC := usecase.NewInvite(slackClient)

			// Create incident configuration with optional settings
			var incidentOpts []usecase.IncidentOption
			if slackCfg.ChannelPrefix != "" {
				incidentOpts = append(incidentOpts, usecase.WithChannelPrefix(slackCfg.ChannelPrefix))
			}
			if serverCfg.FrontendURL != "" {
				incidentOpts = append(incidentOpts, usecase.WithFrontendURL(serverCfg.FrontendURL))
			}
			incidentConfig := usecase.NewIncidentConfig(incidentOpts...)

			incidentUC := usecase.NewIncident(repo, slackClient, categories, inviteUC, incidentConfig)
			taskUC := usecase.NewTaskUseCase(repo, slackClient)
			statusUC := usecase.NewStatusUseCase(repo, slackClient)
			slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, statusUC, slackClient)

			// Create configuration
			config := controller.NewConfig(
				serverCfg.Addr,
				&slackCfg,
				categories,
				serverCfg.FrontendURL,
			)

			// Create use cases structure
			useCases := controller.NewUseCases(
				authUC,
				messageUC,
				incidentUC,
				taskUC,
				slackInteractionUC,
			)

			// Create handlers
			slackHandler := slackCtrl.NewHandler(ctx, &slackCfg, repo, messageUC, incidentUC, taskUC, slackInteractionUC, slackClient)
			authHandler := controller.NewAuthHandler(ctx, &slackCfg, authUC, serverCfg.FrontendURL)

			// Create GraphQL handler
			var graphqlHandler http.Handler
			if repo != nil && incidentUC != nil && taskUC != nil {
				graphqlHandler = controller.CreateGraphQLHandler(repo, slackClient, useCases, categories)
			}

			// Create controllers
			controllers := controller.NewController(slackHandler, authHandler, graphqlHandler)

			// Create HTTP server
			server, err := controller.NewServer(
				ctx,
				config,
				useCases,
				controllers,
				repo,
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
