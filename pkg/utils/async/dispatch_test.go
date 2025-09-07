package async_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
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
