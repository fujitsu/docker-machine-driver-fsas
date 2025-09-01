package timeutils

import "time"

// Clock interface abstracts time functions
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Sleep(d time.Duration)
}

// RealClock implements Clock with real system time
type RealClock struct{}

var _ Clock = (*RealClock)(nil)

// NewRealClock returns new instance of RealClock and error
func NewRealClock() *RealClock {
	return &RealClock{}
}

func (RealClock) Now() time.Time {
	return time.Now()
}

func (RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (RealClock) Sleep(d time.Duration) {
	time.Sleep(d)
}
