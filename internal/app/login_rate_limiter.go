package app

import (
	"fmt"
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

type trustedProxySet struct {
	networks []*net.IPNet
}

func parseTrustedProxyCIDRs(values []string) (trustedProxySet, error) {
	set := trustedProxySet{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return trustedProxySet{}, fmt.Errorf("parse trusted proxy CIDR %q: %w", value, err)
		}
		set.networks = append(set.networks, network)
	}
	return set, nil
}

func (s trustedProxySet) contains(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, network := range s.networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (a *App) loginRateLimitKey(r *http.Request) string {
	host := a.clientIP(r)
	if host == "" {
		return "unknown"
	}
	return host
}

func (a *App) clientIP(r *http.Request) string {
	remote := remoteIP(r.RemoteAddr)
	if remote == nil {
		return "unknown"
	}
	if !a.trustedProxies.contains(remote) {
		return remote.String()
	}

	forwarded := forwardedForIPs(r.Header.Get("X-Forwarded-For"))
	for i := len(forwarded) - 1; i >= 0; i-- {
		if !a.trustedProxies.contains(forwarded[i]) {
			return forwarded[i].String()
		}
	}
	if len(forwarded) > 0 {
		return forwarded[0].String()
	}

	if realIP := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); realIP != nil {
		return realIP.String()
	}
	return remote.String()
}

func remoteIP(remoteAddr string) net.IP {
	host := strings.TrimSpace(remoteAddr)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	return net.ParseIP(host)
}

func forwardedForIPs(header string) []net.IP {
	parts := strings.Split(header, ",")
	ips := make([]net.IP, 0, len(parts))
	for _, part := range parts {
		ip := net.ParseIP(strings.TrimSpace(part))
		if ip != nil {
			ips = append(ips, ip)
		}
	}
	return ips
}
