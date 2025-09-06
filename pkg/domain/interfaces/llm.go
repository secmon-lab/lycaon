package interfaces

import (
	"context"
)

// LLMClient defines the interface for LLM operations
// This will use gollem.Client directly in implementation
type LLMClient interface {
	// GenerateResponse generates a response based on the input prompt
	GenerateResponse(ctx context.Context, prompt string) (string, error)

	// AnalyzeMessage analyzes a message and returns insights
	AnalyzeMessage(ctx context.Context, message string) (string, error)
}
