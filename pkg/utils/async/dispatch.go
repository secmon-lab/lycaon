package async

import (
	"context"
	"runtime/debug"

	"github.com/m-mizutani/ctxlog"
)

// Dispatch executes a handler function asynchronously with proper context and panic recovery
// This allows Slack handlers to respond immediately while processing continues in background
func Dispatch(ctx context.Context, handler func(ctx context.Context) error) {
	// Create a new background context preserving important values
	newCtx := newBackgroundContext(ctx)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				ctxlog.From(newCtx).Error("Panic in async handler",
					"recover", r,
					"stack", string(stack),
				)
			}
		}()

		if err := handler(newCtx); err != nil {
			ctxlog.From(newCtx).Error("Error in async handler",
				"error", err,
			)
		}
	}()
}

// newBackgroundContext creates a new background context preserving important values
func newBackgroundContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	// Preserve logger
	logger := ctxlog.From(ctx)
	if logger != nil {
		newCtx = ctxlog.With(newCtx, logger)
	}

	// Add any other context values that need to be preserved
	// For example, request ID, user ID, etc.

	return newCtx
}
