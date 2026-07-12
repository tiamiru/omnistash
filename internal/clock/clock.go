// Package clock abstracts time-based operations so callers can inject a fake
// implementation in tests instead of depending on the wall clock.
package clock

import "time"

var _ Clock = &realClock{}

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func NewClock() *realClock {
	return &realClock{}
}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
