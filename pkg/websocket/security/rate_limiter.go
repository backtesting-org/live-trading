package security

import (
	"sync"
	"time"
)

type rateLimiter struct {
	tokens     int
	capacity   int
	refillRate time.Duration
	lastRefill time.Time
	mutex      sync.Mutex
}

func NewRateLimiter(capacity int, refillRate time.Duration) RateLimiter {
	return &rateLimiter{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (rl *rateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	if now.Sub(rl.lastRefill) >= rl.refillRate {
		rl.tokens = rl.capacity
		rl.lastRefill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

func (rl *rateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.tokens = rl.capacity
	rl.lastRefill = time.Now()
}
