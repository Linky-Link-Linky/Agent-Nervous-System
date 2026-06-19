package mcp

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu      sync.Mutex
	perMin  int
	buckets map[string]*tokenBucket
}

type tokenBucket struct {
	tokens int
	last   time.Time
}

func NewRateLimiter(perMin int) *RateLimiter {
	return &RateLimiter{
		perMin:  perMin,
		buckets: make(map[string]*tokenBucket),
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	if rl.perMin <= 0 {
		return true
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[key]
	now := time.Now()
	if !ok {
		b = &tokenBucket{tokens: rl.perMin, last: now}
		rl.buckets[key] = b
	}
	elapsed := now.Sub(b.last)
	refill := int(elapsed.Minutes() * float64(rl.perMin))
	if refill > 0 {
		b.tokens += refill
		if b.tokens > rl.perMin {
			b.tokens = rl.perMin
		}
		b.last = now
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

type TokenBudget struct {
	mu        sync.Mutex
	maxTokens int
	period    time.Duration
	usage     map[string]*agentUsage
}

type agentUsage struct {
	tokens  int
	resetAt time.Time
}

func NewTokenBudget(maxTokens int) *TokenBudget {
	return &TokenBudget{
		maxTokens: maxTokens,
		period:    time.Minute,
		usage:     make(map[string]*agentUsage),
	}
}

func (tb *TokenBudget) Allow(key string, tokens int) bool {
	if tb.maxTokens <= 0 {
		return true
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	u, ok := tb.usage[key]
	now := time.Now()
	if !ok || now.After(u.resetAt) {
		u = &agentUsage{tokens: 0, resetAt: now.Add(tb.period)}
		tb.usage[key] = u
	}
	if u.tokens+tokens > tb.maxTokens {
		return false
	}
	u.tokens += tokens
	return true
}

func (tb *TokenBudget) Remaining(key string) int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	u, ok := tb.usage[key]
	if !ok || time.Now().After(u.resetAt) {
		return tb.maxTokens
	}
	remaining := tb.maxTokens - u.tokens
	if remaining < 0 {
		return 0
	}
	return remaining
}
