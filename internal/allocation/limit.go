package allocation

import (
	"errors"
	"sync"
	"time"
)

const (
	defaultRequestsPerSecond = 10
	defaultRateBurst         = 20
)

type rateLimitError struct{}

func (rateLimitError) Error() string {
	return "allocation request rate limit exceeded"
}

// IsRateLimited reports whether err is the outbound allocation rate limit.
func IsRateLimited(err error) bool {
	return errors.Is(err, rateLimitError{})
}

type limiter struct {
	mu        sync.Mutex
	perSecond float64
	burst     float64
	tokens    float64
	updated   time.Time
}

func newLimiter(perSecond float64, burst int) *limiter {
	if perSecond <= 0 {
		perSecond = defaultRequestsPerSecond
	}
	if burst <= 0 {
		burst = defaultRateBurst
	}
	return &limiter{
		perSecond: perSecond,
		burst:     float64(burst),
		tokens:    float64(burst),
	}
}

func (l *limiter) allow(now time.Time) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.updated.IsZero() {
		l.updated = now
	}
	elapsed := now.Sub(l.updated).Seconds()
	if elapsed > 0 {
		l.tokens += elapsed * l.perSecond
		if l.tokens > l.burst {
			l.tokens = l.burst
		}
		l.updated = now
	}
	if l.tokens < 1 {
		return rateLimitError{}
	}
	l.tokens--
	return nil
}
