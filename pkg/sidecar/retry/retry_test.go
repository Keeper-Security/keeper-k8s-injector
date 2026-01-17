package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithRetry_Success(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0

	err := WithRetry(context.Background(), cfg, func() error {
		callCount++
		return nil  // Success on first try
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestWithRetry_FailThenSuccess(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0

	err := WithRetry(context.Background(), cfg, func() error {
		callCount++
		if callCount < 2 {
			return errors.New("temporary failure")
		}
		return nil  // Success on second try
	})

	if err != nil {
		t.Errorf("expected success after retry, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestWithRetry_AllFailures(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0
	testErr := errors.New("persistent failure")

	err := WithRetry(context.Background(), cfg, func() error {
		callCount++
		return testErr
	})

	if err == nil {
		t.Error("expected error after all retries exhausted")
	}
	if callCount != cfg.MaxAttempts {
		t.Errorf("expected %d attempts, got %d", cfg.MaxAttempts, callCount)
	}
	if !errors.Is(err, testErr) {
		t.Errorf("error chain should contain original error")
	}
}

func TestWithRetry_ContextCanceled(t *testing.T) {
	cfg := Config{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	callCount := 0
	start := time.Now()

	err := WithRetry(ctx, cfg, func() error {
		callCount++
		return errors.New("always fails")
	})

	elapsed := time.Since(start)

	// Should return quickly due to context cancellation
	if elapsed > 300*time.Millisecond {
		t.Errorf("took too long: %v (context should cancel retry)", elapsed)
	}

	if err == nil {
		t.Error("expected error when context canceled")
	}

	// Should have attempted at least once, possibly twice
	if callCount < 1 || callCount > 3 {
		t.Errorf("unexpected call count: %d", callCount)
	}
}

func TestExponentialBackoff_Timing(t *testing.T) {
	cfg := Config{
		MaxAttempts: 4,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	attempts := []time.Time{}

	_ = WithRetry(context.Background(), cfg, func() error {
		attempts = append(attempts, time.Now())
		return errors.New("fail")
	})

	if len(attempts) != 4 {
		t.Fatalf("expected 4 attempts, got %d", len(attempts))
	}

	// Check delays between attempts (with some tolerance for timing)
	delays := []time.Duration{
		attempts[1].Sub(attempts[0]),  // Should be ~100ms
		attempts[2].Sub(attempts[1]),  // Should be ~200ms
		attempts[3].Sub(attempts[2]),  // Should be ~400ms
	}

	// Verify exponential growth with 50% tolerance
	expectedDelays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	for i, expected := range expectedDelays {
		if delays[i] < expected/2 || delays[i] > expected*2 {
			t.Errorf("delay[%d] = %v, expected ~%v", i, delays[i], expected)
		}
	}
}

func TestWithRetry_MaxDelayRespected(t *testing.T) {
	cfg := Config{
		MaxAttempts: 10,
		BaseDelay:   1 * time.Second,
		MaxDelay:    2 * time.Second,  // Cap at 2s
	}

	attempts := []time.Time{}

	_ = WithRetry(context.Background(), cfg, func() error {
		attempts = append(attempts, time.Now())
		if len(attempts) >= 4 {
			return nil  // Stop after 4 attempts
		}
		return errors.New("fail")
	})

	// Check that later delays don't exceed maxDelay
	for i := 2; i < len(attempts)-1; i++ {
		delay := attempts[i+1].Sub(attempts[i])
		if delay > cfg.MaxDelay*12/10 {  // 20% tolerance
			t.Errorf("delay[%d] = %v exceeds maxDelay %v", i, delay, cfg.MaxDelay)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", cfg.MaxAttempts)
	}
	if cfg.BaseDelay != 200*time.Millisecond {
		t.Errorf("BaseDelay = %v, want 200ms", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 5*time.Second {
		t.Errorf("MaxDelay = %v, want 5s", cfg.MaxDelay)
	}
}
