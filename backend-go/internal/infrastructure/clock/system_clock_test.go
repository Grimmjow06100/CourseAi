package clock

import (
	"testing"
	"time"
)

func TestSystemClockNowReturnsCurrentTime(t *testing.T) {
	t.Parallel()

	clock := NewSystemClock()
	before := time.Now()
	got := clock.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("Now() = %s, want value between %s and %s", got, before, after)
	}
}
