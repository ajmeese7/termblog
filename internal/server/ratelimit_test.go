package server

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	// Create a limiter allowing 3 requests per 100ms
	rl := NewRateLimiter(3, 100*time.Millisecond)
	defer rl.Stop()

	ip := "192.168.1.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		allowed, count := rl.Allow(ip)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
		if count != i+1 {
			t.Errorf("Request %d: expected count %d, got %d", i+1, i+1, count)
		}
	}

	// 4th request should be denied
	allowed, count := rl.Allow(ip)
	if allowed {
		t.Error("4th request should be denied")
	}
	if count != 3 {
		t.Errorf("Expected count 3 when denied, got %d", count)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	allowed, _ = rl.Allow(ip)
	if !allowed {
		t.Error("Request after window expiry should be allowed")
	}
}

func TestRateLimiter_MultipleIPs(t *testing.T) {
	rl := NewRateLimiter(2, time.Second)
	defer rl.Stop()

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Both IPs should have independent limits
	allowed, _ := rl.Allow(ip1)
	if !allowed {
		t.Error("First request from ip1 should be allowed")
	}
	allowed, _ = rl.Allow(ip1)
	if !allowed {
		t.Error("Second request from ip1 should be allowed")
	}
	allowed, _ = rl.Allow(ip1)
	if allowed {
		t.Error("Third request from ip1 should be denied")
	}

	// ip2 should still have full quota
	allowed, _ = rl.Allow(ip2)
	if !allowed {
		t.Error("First request from ip2 should be allowed")
	}
	allowed, _ = rl.Allow(ip2)
	if !allowed {
		t.Error("Second request from ip2 should be allowed")
	}
	allowed, _ = rl.Allow(ip2)
	if allowed {
		t.Error("Third request from ip2 should be denied")
	}
}

func TestRateLimiter_SlidingWindow(t *testing.T) {
	// Create a limiter allowing 2 requests per 200ms
	rl := NewRateLimiter(2, 200*time.Millisecond)
	defer rl.Stop()

	ip := "192.168.1.1"

	// Make 2 requests
	rl.Allow(ip)
	rl.Allow(ip)

	// Wait 100ms (half the window)
	time.Sleep(100 * time.Millisecond)

	// Should still be denied (both requests within window)
	allowed, _ := rl.Allow(ip)
	if allowed {
		t.Error("Request should be denied - both previous requests still in window")
	}

	// Wait another 150ms (first request should be outside window now)
	time.Sleep(150 * time.Millisecond)

	// Should be allowed now (only 1 request in window)
	allowed, _ = rl.Allow(ip)
	if !allowed {
		t.Error("Request should be allowed - first request expired from window")
	}
}

func TestRateLimiter_Stop(t *testing.T) {
	rl := NewRateLimiter(10, time.Second)

	// Should not panic when stopped
	rl.Stop()

	// Multiple stops should be safe now with sync.Once
	rl.Stop()
	rl.Stop()

	// If we got here without panic, the test passes
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(100, time.Second)
	defer rl.Stop()

	done := make(chan bool)
	ip := "192.168.1.1"

	// Spawn multiple goroutines accessing the limiter
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				rl.Allow(ip)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without a race condition, the test passes
}
