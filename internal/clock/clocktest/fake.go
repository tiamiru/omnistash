// Package clocktest provides a fake clock.Clock for deterministic tests.
package clocktest

import (
	"sync"
	"time"

	"github.com/tiamiru/omnistash/internal/clock"
)

var _ clock.Clock = &FakeClock{}

type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func NewFake(now time.Time) *FakeClock {
	return &FakeClock{now: now}
}

func (f *FakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.now
}

func (f *FakeClock) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = t
}

func (f *FakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}

func (f *FakeClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- f.Now().Add(d)

	return ch
}
