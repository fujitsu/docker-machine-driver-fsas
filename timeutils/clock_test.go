package timeutils

import (
	"testing"
	"time"
)

func TestRealClock_Now(t *testing.T) {
	clock := NewRealClock()
	now := clock.Now()

	// Check if the returned time is close to the current time
	// Allow for a small margin of error (e.g., 100 milliseconds)
	if time.Since(now) > 100*time.Millisecond {
		t.Errorf("RealClock.Now() returned a time too far in the past or future. Expected close to now, got %v", now)
	}
}

func TestRealClock_Since(t *testing.T) {
	clock := NewRealClock()
	past := time.Now().Add(-5 * time.Second) // 5 seconds ago
	duration := clock.Since(past)

	// Check if the duration is approximately 5 seconds
	// Allow for a small margin of error (e.g., 100 milliseconds)
	expectedMin := 5*time.Second - 100*time.Millisecond
	expectedMax := 5*time.Second + 100*time.Millisecond

	if duration < expectedMin || duration > expectedMax {
		t.Errorf("RealClock.Since() returned unexpected duration. Expected around 5s, got %v", duration)
	}
}

func TestRealClock_Sleep(t *testing.T) {
	clock := NewRealClock()
	sleepDuration := 100 * time.Millisecond // Sleep for a short duration

	start := time.Now()
	clock.Sleep(sleepDuration)
	end := time.Now()

	elapsed := end.Sub(start)

	// Check if the elapsed time is approximately the sleep duration
	// Allow for a small margin of error (e.g., 50 milliseconds)
	expectedMin := sleepDuration - 50*time.Millisecond
	expectedMax := sleepDuration + 50*time.Millisecond

	if elapsed < expectedMin || elapsed > expectedMax {
		t.Errorf("RealClock.Sleep() did not sleep for the expected duration. Expected around %v, slept for %v", sleepDuration, elapsed)
	}
}

func TestNewRealClock(t *testing.T) {
	var clock Clock = NewRealClock()
	if clock == nil {
		t.Error("NewRealClock() returned nil, expected a non-nil RealClock instance")
	}
	// Optionally, you can check if the returned type is indeed *RealClock
	if _, ok := clock.(*RealClock); !ok {
		t.Errorf("NewRealClock() returned type %T, expected *RealClock", clock)
	}
}