package vrchat

import (
	"context"
	"sync"
	"time"
)

type requestLimiter struct {
	mu         sync.Mutex
	rate       float64
	burst      float64
	tokens     float64
	last       time.Time
	blockedTil time.Time
}

func newRequestLimiter(ratePerSecond float64, burst int) *requestLimiter {
	now := time.Now()
	return &requestLimiter{rate: ratePerSecond, burst: float64(burst), tokens: float64(burst), last: now}
}

func (l *requestLimiter) Wait(ctx context.Context) error {
	for {
		wait := l.reserveDelay(time.Now())
		if wait <= 0 {
			return nil
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (l *requestLimiter) reserveDelay(now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Before(l.blockedTil) {
		return time.Until(l.blockedTil)
	}
	elapsed := now.Sub(l.last).Seconds()
	l.tokens = min(l.burst, l.tokens+elapsed*l.rate)
	l.last = now
	if l.tokens >= 1 {
		l.tokens--
		return 0
	}
	return time.Duration((1 - l.tokens) / l.rate * float64(time.Second))
}

func (l *requestLimiter) Block(duration time.Duration) {
	if duration <= 0 {
		duration = 2 * time.Second
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	until := time.Now().Add(duration)
	if until.After(l.blockedTil) {
		l.blockedTil = until
	}
}
