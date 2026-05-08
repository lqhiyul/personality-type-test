package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultLoginFailureLimit = 5
	defaultLoginCooldown     = 5 * time.Minute
)

type loginRateLimiter struct {
	mu          sync.Mutex
	attempts    map[string]loginAttempt
	maxFailures int
	cooldown    time.Duration
	now         func() time.Time
}

type loginAttempt struct {
	failures    int
	lockedUntil time.Time
}

func newLoginRateLimiter(maxFailures int, cooldown time.Duration) *loginRateLimiter {
	if maxFailures < 1 {
		maxFailures = defaultLoginFailureLimit
	}
	if cooldown <= 0 {
		cooldown = defaultLoginCooldown
	}
	return &loginRateLimiter{
		attempts:    map[string]loginAttempt{},
		maxFailures: maxFailures,
		cooldown:    cooldown,
		now:         time.Now,
	}
}

func (l *loginRateLimiter) allow(key string) (time.Duration, bool) {
	if l == nil {
		return 0, true
	}

	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt := l.attempts[key]
	if attempt.lockedUntil.After(now) {
		return attempt.lockedUntil.Sub(now), false
	}
	if !attempt.lockedUntil.IsZero() {
		delete(l.attempts, key)
	}
	return 0, true
}

func (l *loginRateLimiter) recordFailure(key string) (time.Duration, bool) {
	if l == nil {
		return 0, false
	}

	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt := l.attempts[key]
	if attempt.lockedUntil.After(now) {
		return attempt.lockedUntil.Sub(now), true
	}
	if !attempt.lockedUntil.IsZero() {
		attempt = loginAttempt{}
	}

	attempt.failures++
	if attempt.failures >= l.maxFailures {
		attempt.failures = 0
		attempt.lockedUntil = now.Add(l.cooldown)
		l.attempts[key] = attempt
		return l.cooldown, true
	}

	l.attempts[key] = attempt
	return 0, false
}

func (l *loginRateLimiter) recordSuccess(key string) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func loginRateLimitKey(r *http.Request) string {
	host := strings.TrimSpace(r.RemoteAddr)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	if host == "" {
		return "unknown"
	}
	return host
}
