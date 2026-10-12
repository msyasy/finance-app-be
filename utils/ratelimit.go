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
	rl := &MemoryRateLimiter{
		requests: make(map[string][]time.Time),
	}

	// Tanpa ini, tiap key (IP atau email) yang pernah masuk tetap tersimpan di memori selamanya.
	go rl.cleanupRoutine()

	return rl
}

// Allow mengecek apakah request untuk 'key' (IP/Email) masih diizinkan
func (rl *MemoryRateLimiter) Allow(key string, maxRequests int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	// Filter timestamps yang melebihi batas durasi
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

// cleanupRoutine menghapus key yang request terakhirnya sudah lebih dari 1 jam lalu.
// Semua pemakaian di LapKeu memakai window maksimal 1 jam, jadi aman.
func (rl *MemoryRateLimiter) cleanupRoutine() {
	for {
		time.Sleep(10 * time.Minute)

		rl.mu.Lock()
		cutoff := time.Now().Add(-1 * time.Hour)
		for key, timestamps := range rl.requests {
			if len(timestamps) == 0 || timestamps[len(timestamps)-1].Before(cutoff) {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}