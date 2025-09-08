package async_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/utils/async"
)

func TestDispatch(t *testing.T) {
	t.Run("Execute handler asynchronously", func(t *testing.T) {
		ctx := context.Background()
		var wg sync.WaitGroup
		executed := false

		wg.Add(1)
		async.Dispatch(ctx, func(ctx context.Context) error {
			defer wg.Done()
			executed = true
			return nil
		})

		// Wait for async execution with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			gt.True(t, executed)
		case <-time.After(1 * time.Second):
			t.Fatal("Async handler did not execute within timeout")
		}
	})

	t.Run("Handle errors in async handler", func(t *testing.T) {
		ctx := context.Background()
		var wg sync.WaitGroup

		wg.Add(1)
		async.Dispatch(ctx, func(ctx context.Context) error {
			defer wg.Done()
			return goerr.New("test error")
		})

		// Wait for async execution
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Test passes if no panic occurs
		case <-time.After(1 * time.Second):
			t.Fatal("Async handler did not complete within timeout")
		}
	})

	t.Run("Recover from panic in async handler", func(t *testing.T) {
		ctx := context.Background()
		var wg sync.WaitGroup

		wg.Add(1)
		async.Dispatch(ctx, func(ctx context.Context) error {
			defer wg.Done()
			panic("test panic")
		})

		// Wait for async execution
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Test passes if panic is recovered
		case <-time.After(1 * time.Second):
			t.Fatal("Async handler did not recover from panic within timeout")
		}
	})

	t.Run("Multiple async dispatches", func(t *testing.T) {
		ctx := context.Background()
		var wg sync.WaitGroup
		counter := 0
		var mu sync.Mutex

		for i := 0; i < 10; i++ {
			wg.Add(1)
			async.Dispatch(ctx, func(ctx context.Context) error {
				defer wg.Done()
				mu.Lock()
				counter++
				mu.Unlock()
				return nil
			})
		}

		// Wait for all async executions
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			gt.Equal(t, 10, counter)
		case <-time.After(2 * time.Second):
			t.Fatal("Async handlers did not complete within timeout")
		}
	})
}

func TestContextPreservation(t *testing.T) {
	t.Run("AuthContext is preserved with model functions", func(t *testing.T) {
		ctx := context.Background()
		authCtx := model.NewAuthContext()
		authCtx.UserID = "U123456789"
		authCtx.SlackUserID = "U987654321"
		authCtx.SessionID = "session-789"
		ctx = model.WithAuthContext(ctx, authCtx)

		var wg sync.WaitGroup
		var preservedAuthCtx *model.AuthContext

		wg.Add(1)
		async.Dispatch(ctx, func(ctx context.Context) error {
			defer wg.Done()
			preservedAuthCtx, _ = model.GetAuthContext(ctx)
			return nil
		})

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			gt.NotEqual(t, nil, preservedAuthCtx)
			gt.Equal(t, authCtx.UserID, preservedAuthCtx.UserID)
			gt.Equal(t, authCtx.SlackUserID, preservedAuthCtx.SlackUserID)
			gt.Equal(t, authCtx.SessionID, preservedAuthCtx.SessionID)
		case <-time.After(1 * time.Second):
			t.Fatal("Async handler did not complete within timeout")
		}
	})

	t.Run("Logger is preserved in background context", func(t *testing.T) {
		ctx := context.Background()
		logger := ctxlog.From(context.Background()) // Get default logger
		ctx = ctxlog.With(ctx, logger)

		var wg sync.WaitGroup
		var hasLogger bool

		wg.Add(1)
		async.Dispatch(ctx, func(ctx context.Context) error {
			defer wg.Done()
			hasLogger = ctxlog.From(ctx) != nil
			return nil
		})

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			gt.True(t, hasLogger)
		case <-time.After(1 * time.Second):
			t.Fatal("Async handler did not complete within timeout")
		}
	})



	t.Run("Each dispatch preserves its own AuthContext", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(map[string]string)
		var mu sync.Mutex

		for i := 0; i < 5; i++ {
			ctx := context.Background()
			authCtx := model.NewAuthContext()
			authCtx.UserID = fmt.Sprintf("U%09d", i)
			ctx = model.WithAuthContext(ctx, authCtx)

			wg.Add(1)
			localID := authCtx.UserID // Capture for closure
			async.Dispatch(ctx, func(ctx context.Context) error {
				defer wg.Done()
				
				// Simulate some work
				time.Sleep(10 * time.Millisecond)
				
				preservedAuthCtx, _ := model.GetAuthContext(ctx)
				mu.Lock()
				if preservedAuthCtx != nil {
					results[localID] = preservedAuthCtx.UserID
				} else {
					results[localID] = ""
				}
				mu.Unlock()
				
				return nil
			})
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Verify each goroutine preserved its own user ID
			for i := 0; i < 5; i++ {
				expectedID := fmt.Sprintf("U%09d", i)
				gt.Equal(t, expectedID, results[expectedID])
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Async handlers did not complete within timeout")
		}
	})
}

