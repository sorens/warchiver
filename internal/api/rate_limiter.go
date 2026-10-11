package api

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Battle.net caps: 100/s (hard, 429 on overflow) and 36,000/hour (soft).
const (
	perSecondMax      = 100
	perSecondInterval = 10 * time.Millisecond
	hourlyMax         = 36000
	hourlyInterval    = 100 * time.Millisecond
)

// bucket is a token-bucket; refill credits whole intervals and preserves the remainder.
type bucket struct {
	tokens     int
	max        int
	interval   time.Duration
	lastRefill time.Time
}

func (b *bucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastRefill)
	n := int(elapsed / b.interval)
	if n <= 0 {
		return
	}
	b.tokens += n
	if b.tokens > b.max {
		b.tokens = b.max
	}
	b.lastRefill = b.lastRefill.Add(time.Duration(n) * b.interval)
}

// RateLimiter enforces both Battle.net caps via two independent token buckets.
type RateLimiter struct {
	mu        sync.Mutex
	perSecond bucket
	hourly    bucket
	inFlight  atomic.Int64
}

var defaultLimiter = NewRateLimiter()

// DefaultRateLimiter returns the process-wide limiter shared by clients.
func DefaultRateLimiter() *RateLimiter { return defaultLimiter }

// NewRateLimiter creates a limiter at the published Battle.net caps.
func NewRateLimiter() *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		perSecond: bucket{tokens: perSecondMax, max: perSecondMax, interval: perSecondInterval, lastRefill: now},
		hourly:    bucket{tokens: hourlyMax, max: hourlyMax, interval: hourlyInterval, lastRefill: now},
	}
}

// Wait blocks until a token is available in both buckets.
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		now := time.Now()
		r.perSecond.refill(now)
		r.hourly.refill(now)
		if r.perSecond.tokens > 0 && r.hourly.tokens > 0 {
			r.perSecond.tokens--
			r.hourly.tokens--
			r.mu.Unlock()
			r.inFlight.Add(1)
			return nil
		}
		wait := r.perSecond.interval
		if r.hourly.tokens == 0 {
			wait = r.hourly.interval
		}
		r.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// Done marks a request as no longer in flight. Clients should call this after
// receiving a response (or transport error).
func (r *RateLimiter) Done()           { r.inFlight.Add(-1) }
func (r *RateLimiter) InFlight() int64 { return r.inFlight.Load() }
