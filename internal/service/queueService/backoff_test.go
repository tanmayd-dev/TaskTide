package queueService

import (
	"testing"
	"time"
)

// expectedBackoff is the pre-jitter backoff for a retry count: 2s base with
// a shift cap of 10 and a 300s hard cap.
func expectedBackoff(retryCount int32) time.Duration {
	shift := min(retryCount, 10)
	backoff := min(2*time.Second*(1<<uint(shift)), 300*time.Second)
	return backoff
}

func TestNextBackoff(t *testing.T) {
	cases := []struct {
		name       string
		retryCount int32
		// nextBackoff is backoff/2 + up to backoff/2 of jitter, so every
		// sample must land within [backoff/2, backoff].
		min time.Duration
		max time.Duration
	}{
		{"base 2s", 0, 1 * time.Second, 2 * time.Second},
		{"doubles to 4s", 1, 2 * time.Second, 4 * time.Second},
		{"doubles to 8s", 2, 4 * time.Second, 8 * time.Second},
		{"saturates at shift cap 10", 10, 150 * time.Second, 300 * time.Second},
		{"still saturated above shift cap", 20, 150 * time.Second, 300 * time.Second},
		{"hard cap 300s at retry 63", 63, 150 * time.Second, 300 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < 500; i++ {
				got := nextBackoff(tc.retryCount)
				if got < tc.min || got > tc.max {
					t.Fatalf("nextBackoff(%d) = %v, want within [%v, %v]", tc.retryCount, got, tc.min, tc.max)
				}
			}
		})
	}
}

func TestNextBackoff_ProgressionFollowsBackoffBounds(t *testing.T) {
	// For every retry count the returned delay must stay within the
	// [backoff/2, backoff] window of its own backoff budget, which doubles
	// 2s -> 4s -> 8s -> ... until the shift/hard caps.
	for c := int32(0); c <= 12; c++ {
		backoff := expectedBackoff(c)
		for i := 0; i < 200; i++ {
			got := nextBackoff(c)
			if got < backoff/2 || got > backoff {
				t.Fatalf("nextBackoff(%d) = %v, want within [%v, %v]", c, got, backoff/2, backoff)
			}
		}
	}
}
