package retry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPredicates(t *testing.T) {
	errBoom := errors.New("boom")
	tests := []struct {
		name string
		fn   IsRetryable
		err  error
		want bool
	}{
		{"always retries on error", AlwaysRetry, errBoom, true},
		{"always retries on nil", AlwaysRetry, nil, true},
		{"never retries on error", NeverRetry, errBoom, false},
		{"never retries on nil", NeverRetry, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.fn(tt.err))
		})
	}
}

func TestRetryIf_ClassifiesErrors(t *testing.T) {
	retryable := errors.New("retryable")
	matches := func(err error) bool { return err == retryable }

	matched := RetryIf(func(context.Context) error { return retryable }, matches)
	assert.False(t, IsNonRetryable(matched(context.Background())),
		"matching errors pass through unwrapped (retryable)")

	wrapped := RetryIf(func(context.Context) error { return errors.New("other") }, matches)
	assert.True(t, IsNonRetryable(wrapped(context.Background())),
		"non-matching errors are wrapped as non-retryable")
}
