package retry

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPolicy_Do_Success(t *testing.T) {
	p := NewPolicy()
	calls := 0

	err := p.Do(func() error {
		calls++
		if calls < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestPolicy_Do_MaxAttempts(t *testing.T) {
	p := NewPolicy(MaxAttempts(3))
	calls := 0

	err := p.Do(func() error {
		calls++
		return errors.New("always fails")
	})

	assert.Error(t, err)
	assert.Equal(t, 3, calls)
	assert.Contains(t, err.Error(), "max retries exceeded")
}

func TestPolicy_Do_MaxDuration(t *testing.T) {
	p := NewPolicy(MaxDuration(100 * time.Millisecond))
	calls := 0

	err := p.Do(func() error {
		calls++
		time.Sleep(60 * time.Millisecond)
		return errors.New("always fails")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

func TestPolicy_DoWithContext_ContextCanceled(t *testing.T) {
	p := NewPolicy()
	ctx, cancel := context.WithCancel(context.Background())
	calls := int32(0)

	done := make(chan error)
	go func() {
		done <- p.DoWithContext(ctx, func(ctx context.Context) error {
			if atomic.AddInt32(&calls, 1) == 2 {
				cancel()
			}
			time.Sleep(10 * time.Millisecond)
			return errors.New("fails")
		})
	}()

	err := <-done
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
	assert.True(t, atomic.LoadInt32(&calls) <= 3)
}

func TestExponentialBackoff(t *testing.T) {
	b := NewExponentialBackoff(
		BaseDelay(10*time.Millisecond),
		Multiplier(2),
		WithJitter(),
	)

	// Jitter affects all attempts including 0
	// Attempt 0: 10ms ± 25% = 7.5ms to 12.5ms
	delay0 := b.Delay(0)
	assert.True(t, delay0 >= 7*time.Millisecond && delay0 <= 13*time.Millisecond)

	// Attempt 1: 10ms * 2 = 20ms ± 25% = 15ms to 25ms
	delay1 := b.Delay(1)
	assert.True(t, delay1 >= 15*time.Millisecond && delay1 <= 25*time.Millisecond)

	// Attempt 2: 10ms * 4 = 40ms ± 25% = 30ms to 50ms
	delay2 := b.Delay(2)
	assert.True(t, delay2 >= 30*time.Millisecond && delay2 <= 50*time.Millisecond)
}

func TestExponentialBackoff_MaxDelay(t *testing.T) {
	b := NewExponentialBackoff(
		BaseDelay(10*time.Millisecond),
		Multiplier(10),
		MaxDelay(50*time.Millisecond),
	)

	// Jitter affects attempt 0 too
	delay0 := b.Delay(0)
	assert.True(t, delay0 >= 7*time.Millisecond && delay0 <= 13*time.Millisecond)

	// With max delay and jitter (default 0.25 ratio):
	// Base delay is capped at 50ms, jitterRange = 50ms * 0.25 = 12.5ms
	// Range: (50ms - 12.5ms) to (50ms - 12.5ms + 2*12.5ms) = 37.5ms to 62.5ms
	delay10 := b.Delay(10)
	assert.True(t, delay10 >= 37*time.Millisecond && delay10 <= 63*time.Millisecond)
}

func TestExponentialBackoff_NoJitter(t *testing.T) {
	b := NewExponentialBackoff(
		BaseDelay(10*time.Millisecond),
		Multiplier(2),
	)
	// Disable jitter for deterministic test
	b.jitter = false

	assert.Equal(t, 10*time.Millisecond, b.Delay(0))
	assert.Equal(t, 20*time.Millisecond, b.Delay(1))
	assert.Equal(t, 40*time.Millisecond, b.Delay(2))
}

func TestLinearBackoff(t *testing.T) {
	l := NewLinearBackoff(
		LinearBaseDelay(10*time.Millisecond),
		LinearIncrement(5*time.Millisecond),
	)

	assert.Equal(t, 10*time.Millisecond, l.Delay(0))
	assert.Equal(t, 15*time.Millisecond, l.Delay(1))
	assert.Equal(t, 20*time.Millisecond, l.Delay(2))
}

func TestLinearBackoff_MaxDelay(t *testing.T) {
	l := NewLinearBackoff(
		LinearBaseDelay(10*time.Millisecond),
		LinearIncrement(100*time.Millisecond),
		LinearMaxDelay(50*time.Millisecond),
	)

	assert.Equal(t, 10*time.Millisecond, l.Delay(0))
	assert.Equal(t, 50*time.Millisecond, l.Delay(10))
}

func TestConstantBackoff(t *testing.T) {
	c := NewConstantBackoff(50 * time.Millisecond)

	assert.Equal(t, 50*time.Millisecond, c.Delay(0))
	assert.Equal(t, 50*time.Millisecond, c.Delay(10))
}

func TestDo_Convenience(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 3 {
			return errors.New("temp")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDoWithContext_Convenience(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := DoWithContext(ctx, func(ctx context.Context) error {
		calls++
		if calls < 2 {
			return errors.New("temp")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestBackoff_Option(t *testing.T) {
	p := NewPolicy(
		MaxAttempts(5),
		MaxDuration(time.Hour),
		Backoff(NewConstantBackoff(10*time.Millisecond)),
	)

	calls := 0
	start := time.Now()
	_ = p.Do(func() error {
		calls++
		if calls < 3 {
			return errors.New("temp")
		}
		return nil
	})
	elapsed := time.Since(start)

	assert.Equal(t, 3, calls)
	// With 2 retries at 10ms each, should be around 20ms
	assert.True(t, elapsed >= 15*time.Millisecond && elapsed < 100*time.Millisecond)
}

func TestRetryIf(t *testing.T) {
	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("non-retryable")

	calls := 0
	// Only retry the specific retryableErr
	check := func(err error) bool {
		return err == retryableErr
	}

	fn := RetryIf(func(ctx context.Context) error {
		calls++
		if calls == 1 {
			return retryableErr
		}
		return nonRetryableErr
	}, check)

	p := NewPolicy()
	err := p.DoWithContext(context.Background(), fn)

	assert.Error(t, err)
	// Should stop on non-retryable error after first retry
	assert.Equal(t, 2, calls)
}

func TestRetrySpecificErrors(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	check := RetrySpecificErrors(err1, err2)

	assert.True(t, check(err1))
	assert.True(t, check(err2))
	assert.False(t, check(err3))
}

func TestPolicy_ZeroAttempts(t *testing.T) {
	p := NewPolicy(MaxAttempts(1))
	calls := 0

	err := p.Do(func() error {
		calls++
		return errors.New("fails")
	})

	assert.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestExponentialBackoff_CustomJitterRatio(t *testing.T) {
	b := NewExponentialBackoff(
		BaseDelay(100*time.Millisecond),
		JitterRatio(0.5), // 50% jitter
	)

	delay := b.Delay(1)
	// 100ms * 2 = 200ms, with +/- 100ms jitter = 100-300ms range
	assert.True(t, delay >= 100*time.Millisecond && delay <= 300*time.Millisecond)
}

func TestPolicy_ContextAlreadyCanceled(t *testing.T) {
	p := NewPolicy()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.DoWithContext(ctx, func(ctx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
}

func TestPolicy_LongRunningSuccess(t *testing.T) {
	p := NewPolicy(MaxAttempts(1))

	err := p.Do(func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	assert.NoError(t, err)
}

func TestPolicy_FastFailure(t *testing.T) {
	p := NewPolicy(MaxAttempts(5), MaxDuration(time.Hour), Backoff(NewConstantBackoff(10*time.Millisecond)))

	calls := 0
	start := time.Now()
	err := p.Do(func() error {
		calls++
		return errors.New("fail fast")
	})
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, 5, calls)
	// 5 attempts, 4 delays of 10ms each = 40ms minimum
	assert.True(t, elapsed >= 40*time.Millisecond)
}

func TestNonRetryableError(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped := &nonRetryableError{err: baseErr}

	assert.Contains(t, wrapped.Error(), "non-retryable")
	assert.Equal(t, baseErr, wrapped.Unwrap())
}

func TestRetryIf_WithNonRetryable(t *testing.T) {
	calls := 0
	check := func(err error) bool {
		return err.Error() != "stop"
	}

	fn := RetryIf(func(ctx context.Context) error {
		calls++
		if calls == 3 {
			return errors.New("stop")
		}
		return errors.New("continue")
	}, check)

	p := NewPolicy()
	err := p.DoWithContext(context.Background(), fn)

	assert.Error(t, err)
	// Should stop at 3rd call (non-retryable)
	assert.Equal(t, 3, calls)
}

func TestPolicy_DoWithContext_PassingContext(t *testing.T) {
	p := NewPolicy()
	ctx := context.Background()

	calls := 0
	err := p.DoWithContext(ctx, func(ctx context.Context) error {
		calls++
		assert.NotNil(t, ctx)
		if calls < 2 {
			return errors.New("temp")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestBackoff_Strategies(t *testing.T) {
	strategies := []struct {
		name     BackoffStrategy
		expected []time.Duration
		jittered bool
	}{
		{NewConstantBackoff(50 * time.Millisecond), []time.Duration{50 * time.Millisecond, 50 * time.Millisecond, 50 * time.Millisecond}, false},
		{NewLinearBackoff(LinearBaseDelay(10*time.Millisecond), LinearIncrement(10*time.Millisecond)), []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond}, false},
		{NewExponentialBackoff(BaseDelay(10*time.Millisecond), Multiplier(2), MaxDelay(100*time.Millisecond)), []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond}, true},
	}

	for _, s := range strategies {
		t.Run(fmt.Sprintf("%T", s.name), func(t *testing.T) {
			for i, expected := range s.expected {
				delay := s.name.Delay(i)
				if s.jittered {
					// Allow tolerance for jittered backoff
					assert.InDelta(t, float64(expected), float64(delay), float64(expected)*0.3)
				} else {
					assert.Equal(t, expected, delay)
				}
			}
		})
	}
}

func TestPolicy_MaxDurationWithFastFail(t *testing.T) {
	p := NewPolicy(MaxDuration(50*time.Millisecond), Backoff(NewConstantBackoff(30*time.Millisecond)))

	calls := 0
	err := p.Do(func() error {
		calls++
		time.Sleep(10 * time.Millisecond)
		return errors.New("fail")
	})

	assert.Error(t, err)
	// Should hit max duration before max attempts
	assert.LessOrEqual(t, calls, 2)
}

func TestDefaultPolicy(t *testing.T) {
	p := NewPolicy()

	assert.Equal(t, 3, p.maxAttempts)
	assert.Equal(t, time.Minute, p.maxDuration)
	assert.IsType(t, &exponentialBackoff{}, p.backoff)
}

func TestPolicy_LastErrorReturned(t *testing.T) {
	p := NewPolicy()

	expectedErr := errors.New("final error")
	err := p.Do(func() error {
		return expectedErr
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
	assert.Contains(t, err.Error(), "final error")
}
