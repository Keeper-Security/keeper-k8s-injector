package retry

import (
	"context"
	"fmt"
	"time"
)

// Config defines retry behavior with exponential backoff.
// Matches Infisical's retry pattern: 3 attempts, 200ms base, 5s max.
type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultConfig returns standard retry configuration.
// Based on industry best practices (Infisical, cloud SDK defaults).
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    5 * time.Second,
	}
}

// WithRetry executes fn with exponential backoff retry logic.
// Retries on any error, with delays: 200ms, 400ms, 800ms (capped at maxDelay).
//
// Context cancellation is respected - returns immediately if ctx.Done().
//
// Example:
//
//	err := retry.WithRetry(ctx, retry.DefaultConfig(), func() error {
//	    return fetchFromAPI()
//	})
func WithRetry(ctx context.Context, cfg Config, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Don't sleep after last attempt
		if attempt < cfg.MaxAttempts-1 {
			// Exponential backoff: baseDelay * 2^attempt, capped at maxDelay
			delay := cfg.BaseDelay * (1 << uint(attempt))
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}

			// Respect context cancellation during delay
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return fmt.Errorf("retry canceled: %w", ctx.Err())
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
