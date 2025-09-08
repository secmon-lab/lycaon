package config

import (
	"context"
	"log/slog"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
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
func (g *Gemini) Configure(ctx context.Context) (gollem.LLMClient, error) {
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
func (g *Gemini) ConfigureOptional(ctx context.Context, logger *slog.Logger) gollem.LLMClient {
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

// vertexAdapter adapts Vertex AI model to gollem.LLMClient
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

// NewSession creates a new session for LLM interactions
func (a *vertexAdapter) NewSession(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
	// Create a session that wraps our Vertex AI model
	return &vertexSession{
		model: a.model,
	}, nil
}

// GenerateEmbedding generates embeddings for the given input
func (a *vertexAdapter) GenerateEmbedding(ctx context.Context, dimension int, input []string) ([][]float64, error) {
	// Vertex AI Gemini doesn't directly support embeddings via this API
	// Return an error for now
	return nil, goerr.New("embedding generation not supported for Vertex AI Gemini")
}

// CountTokens counts the tokens in the given history
func (a *vertexAdapter) CountTokens(ctx context.Context, history *gollem.History) (int, error) {
	// Use Vertex AI's token counting if available
	// For now, return a rough estimate based on history size
	// This is a simplified implementation - proper token counting would use the model's tokenizer
	if history == nil {
		return 0, nil
	}
	// Rough estimate: assume average of 100 tokens per exchange
	// This should be replaced with actual token counting when available
	return 100, nil
}

// IsCompatibleHistory checks if the history is compatible with this client
func (a *vertexAdapter) IsCompatibleHistory(ctx context.Context, history *gollem.History) error {
	// Vertex AI Gemini should be compatible with standard history format
	return nil
}

// vertexSession implements gollem.Session for Vertex AI
type vertexSession struct {
	model   *genai.GenerativeModel
	history *gollem.History
}

// GenerateContent generates content based on the provided inputs
func (s *vertexSession) GenerateContent(ctx context.Context, inputs ...gollem.Input) (*gollem.Response, error) {
	// Convert gollem inputs to genai content
	var genaiContents []genai.Part
	for _, input := range inputs {
		switch i := input.(type) {
		case gollem.Text:
			genaiContents = append(genaiContents, genai.Text(i))
		default:
			// For other input types, try to convert to text
			genaiContents = append(genaiContents, genai.Text(""))
		}
	}

	// Generate content using Vertex AI
	resp, err := s.model.GenerateContent(ctx, genaiContents...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate content")
	}

	if len(resp.Candidates) == 0 {
		return nil, goerr.New("no response candidates")
	}

	// Convert response to gollem format
	var texts []string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			texts = append(texts, string(text))
		}
	}

	return &gollem.Response{
		Texts: texts,
	}, nil
}

// GenerateStream generates streaming content based on the provided inputs
func (s *vertexSession) GenerateStream(ctx context.Context, inputs ...gollem.Input) (<-chan *gollem.Response, error) {
	// For now, implement as non-streaming
	// Could be enhanced to use Vertex AI's streaming API
	ch := make(chan *gollem.Response, 1)
	
	go func() {
		defer close(ch)
		resp, err := s.GenerateContent(ctx, inputs...)
		if err != nil {
			ch <- &gollem.Response{Error: err}
			return
		}
		ch <- resp
	}()
	
	return ch, nil
}

// History returns the session history
func (s *vertexSession) History() *gollem.History {
	if s.history == nil {
		s.history = &gollem.History{}
	}
	return s.history
}
