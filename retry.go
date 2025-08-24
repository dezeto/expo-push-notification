package expo

import (
	"context"
	"math"
	"net/http"
	"time"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// DefaultRetryConfig provides sensible defaults for retry logic
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:      3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}
}

// IsRetryableError checks if an HTTP response indicates a retryable error
func IsRetryableError(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// ExponentialBackoff calculates the backoff duration for a given attempt
func (c *RetryConfig) ExponentialBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return c.InitialInterval
	}

	duration := float64(c.InitialInterval) * math.Pow(c.Multiplier, float64(attempt-1))
	backoff := time.Duration(duration)

	if backoff > c.MaxInterval {
		backoff = c.MaxInterval
	}

	return backoff
}

// WithRetry executes a function with exponential backoff retry logic
func (c *Client) WithRetry(ctx context.Context, retryConfig *RetryConfig, fn func() (*http.Response, error)) (*http.Response, error) {
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}

	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := retryConfig.ExponentialBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				// Continue with retry
			}
		}

		resp, lastErr = fn()
		if lastErr != nil {
			continue
		}

		if resp != nil && IsRetryableError(resp.StatusCode) {
			resp.Body.Close()
			lastErr = &ServerError{
				Message:  "retryable error",
				Response: resp,
			}
			continue
		}

		// Success or non-retryable error
		return resp, lastErr
	}

	return nil, lastErr
}
