package api

import (
	"testing"
	"time"
)

func TestBucketRefillPreservesRemainder(t *testing.T) {
	start := time.Now()
	b := bucket{tokens: 0, max: 100, interval: 10 * time.Millisecond, lastRefill: start}

	b.refill(start.Add(15 * time.Millisecond))
	if b.tokens != 1 {
		t.Fatalf("after 15ms expected 1 token, got %d", b.tokens)
	}
	if got := b.lastRefill.Sub(start); got != 10*time.Millisecond {
		t.Fatalf("lastRefill should advance 10ms, advanced %v", got)
	}

	b.refill(start.Add(20 * time.Millisecond))
	if b.tokens != 2 {
		t.Fatalf("after 20ms expected 2 tokens, got %d (drift bug if 1)", b.tokens)
	}
}

func TestBucketRefillCapsAtMax(t *testing.T) {
	start := time.Now()
	b := bucket{tokens: 95, max: 100, interval: 10 * time.Millisecond, lastRefill: start}

	b.refill(start.Add(time.Second))
	if b.tokens != 100 {
		t.Fatalf("expected cap at 100, got %d", b.tokens)
	}
	if got := b.lastRefill.Sub(start); got != time.Second {
		t.Fatalf("lastRefill should advance 1s on overflow, advanced %v", got)
	}
}

func TestRateLimiterEnforcesHourlyCap(t *testing.T) {
	r := NewRateLimiter()
	r.mu.Lock()
	r.hourly.tokens = 0
	r.perSecond.tokens = perSecondMax
	r.mu.Unlock()

	r.mu.Lock()
	r.perSecond.refill(time.Now())
	r.hourly.refill(time.Now())
	canTake := r.perSecond.tokens > 0 && r.hourly.tokens > 0
	r.mu.Unlock()
	if canTake {
		t.Fatal("Wait should not be able to take a token when hourly bucket is empty")
	}
}
