package async

import (
	"context"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/m-mizutani/ctxlog"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
)

// Dispatch executes a handler function asynchronously with panic recovery
// This is a simple dispatcher that runs a function in a goroutine with error handling
// Context preservation should be handled by the caller
func Dispatch(ctx context.Context, handler func(ctx context.Context) error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				ctxlog.From(ctx).Error("Panic in async handler",
					"recover", r,
					"stack", string(stack),
				)
			}
		}()

		if err := handler(ctx); err != nil {
			ctxlog.From(ctx).Error("Error in async handler",
				"error", err,
			)
		}
	}()
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// traceIDKey is for distributed tracing (OpenTelemetry, etc.)
	traceIDKey contextKey = "traceID"
)

// WithTraceID adds a trace ID to the context for distributed tracing
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from the context
func GetTraceID(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDKey).(string)
	return traceID, ok
}

// NewBackgroundContext creates a new background context with auth context preserved
// This should be used when spawning async operations that need to maintain auth info
func NewBackgroundContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	// Preserve logger with all its fields
	logger := ctxlog.From(ctx)
	if logger != nil {
		newCtx = ctxlog.With(newCtx, logger)
	}

	// Get or create auth context from the original context
	authCtx := model.GetOrCreateAuthContext(ctx)

	// Attach auth context to the new background context
	newCtx = model.WithAuthContext(newCtx, authCtx)

	// If we have a request ID from chi middleware, add it to logger for consistent logging
	if reqID := middleware.GetReqID(ctx); reqID != "" && logger != nil {
		enrichedLogger := logger.With("request_id", reqID)
		newCtx = ctxlog.With(newCtx, enrichedLogger)
		// Also preserve request ID in context for chi middleware
		newCtx = context.WithValue(newCtx, middleware.RequestIDKey, reqID)
	}

	// Preserve trace ID separately for distributed tracing
	if traceID, ok := GetTraceID(ctx); ok {
		newCtx = WithTraceID(newCtx, traceID)
	}

	return newCtx
}

