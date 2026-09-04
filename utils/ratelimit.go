package utils

import (
	"sync"
	"time"
)

type MemoryRateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
}

func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{
		requests: make(map[string][]time.Time),
	}
}

// Allow mengecek apakah request untuk 'key' (IP/Email) masih diizinkan
func (rl *MemoryRateLimiter) Allow(key string, maxRequests int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	// Filter timestamps yang melebihi batas durasi (1 jam)
	var valid []time.Time
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= maxRequests {
		rl.requests[key] = valid
		return false
	}

	valid = append(valid, now)
	rl.requests[key] = valid
	return true
}