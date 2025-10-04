package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/urfave/cli/v3"
)

// Gemini holds Gemini configuration
type Gemini struct {
	Project  string
	Location string
	Model    string
}

// Flags returns CLI flags for Gemini configuration
func (g *Gemini) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "gemini-project",
			Usage:       "GCP project ID for Gemini",
			Category:    "Gemini",
			Sources:     cli.EnvVars("LYCAON_GEMINI_PROJECT_ID"),
			Destination: &g.Project,
		},
		&cli.StringFlag{
			Name:        "gemini-location",
			Usage:       "Gemini location",
			Category:    "Gemini",
			Value:       "us-central1",
			Sources:     cli.EnvVars("LYCAON_GEMINI_LOCATION"),
			Destination: &g.Location,
		},
		&cli.StringFlag{
			Name:        "gemini-model",
			Usage:       "Gemini model name",
			Category:    "Gemini",
			Value:       "gemini-2.5-flash",
			Sources:     cli.EnvVars("LYCAON_GEMINI_MODEL"),
			Destination: &g.Model,
		},
	}
}

// Configure creates and returns a Gemini LLM client
func (g *Gemini) Configure(ctx context.Context) (gollem.LLMClient, error) {
	if !g.IsConfigured() {
		return nil, goerr.New("Gemini configuration is required. Please provide LYCAON_GEMINI_PROJECT_ID")
	}

	// Create Gemini client using gollem's gemini package
	client, err := gemini.New(ctx, g.Project, g.Location,
		gemini.WithModel(g.Model),
		gemini.WithThinkingBudget(0),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Gemini client",
			goerr.V("project", g.Project),
			goerr.V("location", g.Location),
			goerr.V("model", g.Model),
		)
	}

	return client, nil
}

// ConfigureOptional creates a Gemini LLM client if configured, returns nil if not
func (g *Gemini) ConfigureOptional(ctx context.Context) gollem.LLMClient {
	logger := ctxlog.From(ctx)
	if !g.IsConfigured() {
		logger.Info("Gemini not configured")
		return nil
	}

	logger.Info("Configuring Gemini LLM",
		slog.String("projectID", g.Project),
		slog.String("location", g.Location),
		slog.String("model", g.Model),
	)

	client, err := g.Configure(ctx)
	if err != nil {
		logger.Warn("Failed to create Vertex AI client", slog.Any("error", err))
		return nil
	}

	return client
}

// IsConfigured checks if Gemini is properly configured
func (g *Gemini) IsConfigured() bool {
	return g.Project != ""
}

// LogValue returns structured log value
func (g Gemini) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("project", g.Project),
		slog.String("location", g.Location),
		slog.String("model", g.Model),
	)
}
