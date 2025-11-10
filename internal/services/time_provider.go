package services

import (
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
)

// LiveTimeProvider provides real wall-clock time for live trading
type LiveTimeProvider struct{}

// NewLiveTimeProvider creates a new live time provider
func NewLiveTimeProvider() temporal.TimeProvider {
	return &LiveTimeProvider{}
}

// Now returns the current wall-clock time
func (ltp *LiveTimeProvider) Now() time.Time {
	return time.Now()
}

// After waits for the duration to elapse and then sends the current time on the returned channel
func (ltp *LiveTimeProvider) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// NewTimer creates a new Timer that will send the current time on its channel after at least duration d
func (ltp *LiveTimeProvider) NewTimer(d time.Duration) temporal.Timer {
	return &liveTimer{timer: time.NewTimer(d)}
}

// Since returns the time elapsed since t
func (ltp *LiveTimeProvider) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// NewTicker returns a new Ticker containing a channel that will send the current time on the channel after each tick
func (ltp *LiveTimeProvider) NewTicker(d time.Duration) temporal.Ticker {
	return &liveTicker{ticker: time.NewTicker(d)}
}

// Sleep pauses the current goroutine for at least the duration d
func (ltp *LiveTimeProvider) Sleep(d time.Duration) {
	time.Sleep(d)
}

// liveTimer wraps the standard library timer
type liveTimer struct {
	timer *time.Timer
}

func (lt *liveTimer) C() <-chan time.Time {
	return lt.timer.C
}

func (lt *liveTimer) Reset(d time.Duration) bool {
	return lt.timer.Reset(d)
}

func (lt *liveTimer) Stop() bool {
	return lt.timer.Stop()
}

// liveTicker wraps the standard library ticker
type liveTicker struct {
	ticker *time.Ticker
}

func (lt *liveTicker) C() <-chan time.Time {
	return lt.ticker.C
}

func (lt *liveTicker) Reset(d time.Duration) {
	lt.ticker.Reset(d)
}

func (lt *liveTicker) Stop() {
	lt.ticker.Stop()
}
