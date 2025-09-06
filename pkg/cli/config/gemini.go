package config

import (
	"context"
	"log/slog"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/option"
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
			Sources:     cli.EnvVars("LYCAON_GEMINI_PROJECT"),
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
			Value:       "gemini-1.5-flash",
			Sources:     cli.EnvVars("LYCAON_GEMINI_MODEL"),
			Destination: &g.Model,
		},
	}
}

// Configure creates and returns a Gemini LLM client
func (g *Gemini) Configure(ctx context.Context) (interfaces.LLMClient, error) {
	if !g.IsConfigured() {
		return nil, nil
	}

	// Create Vertex AI client
	client, err := genai.NewClient(ctx, g.Project, g.Location, option.WithCredentialsFile(""))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Vertex AI client")
	}

	model := client.GenerativeModel(g.Model)
	return &vertexAdapter{
		model:  model,
		client: client,
	}, nil
}

// ConfigureOptional creates a Gemini LLM client if configured, returns nil if not
func (g *Gemini) ConfigureOptional(ctx context.Context, logger *slog.Logger) interfaces.LLMClient {
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

// vertexAdapter adapts Vertex AI model to interfaces.LLMClient
type vertexAdapter struct {
	model  *genai.GenerativeModel
	client *genai.Client
}

// Close closes the underlying client
func (a *vertexAdapter) Close() error {
	if a.client != nil {
		return a.client.Close()
	}
	return nil
}

func (a *vertexAdapter) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	resp, err := a.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}
	if len(resp.Candidates) == 0 {
		return "", goerr.New("no response candidates")
	}

	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			result += string(text)
		}
	}
	return result, nil
}

func (a *vertexAdapter) AnalyzeMessage(ctx context.Context, message string) (string, error) {
	prompt := "Analyze this message and provide insights: " + message
	return a.GenerateResponse(ctx, prompt)
}
